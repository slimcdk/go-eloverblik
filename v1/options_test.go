package eloverblik

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

// newTestServer serves responses with a known set of headers for both the token
// endpoint and the export endpoints.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Test-Header", "test-value")
		w.Header().Add("X-Multi-Value", "first")
		w.Header().Add("X-Multi-Value", "second")

		if strings.HasSuffix(r.URL.Path, "/token") {
			_, _ = io.WriteString(w, `{"result":"test-access-token"}`)
			return
		}

		_, _ = io.WriteString(w, exportedCSV)
	}))
	t.Cleanup(server.Close)

	return server
}

const exportedCSV = "column1;column2\nvalue1;value2\n"

// newTestCustomer creates a Customer client pointed at the given base URL.
func newTestCustomer(t *testing.T, baseURL string, opts ...Option) *client {
	t.Helper()

	c, ok := NewCustomer("test-refresh-token", opts...).(*client)
	assert.True(t, ok, "client should be of internal type *client")
	c.resty.SetBaseURL(baseURL)

	return c
}

func TestWithResponseHeaderOutput(t *testing.T) {
	server := newTestServer(t)

	meteringPointIDs := []string{"571313180100000001"}
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	exportPath := fmt.Sprintf("/meterdata/timeseries/export/%s/%s/%s", from.In(cph).Format(time.DateOnly), to.In(cph).Format(time.DateOnly), Hour)

	tests := []struct {
		name     string
		call     func(t *testing.T, c *client)
		expected []string
	}{
		{
			name: "parsed call",
			call: func(t *testing.T, c *client) {
				accessToken, err := c.GetDataAccessToken()
				assert.NoError(t, err)
				assert.Equal(t, "test-access-token", accessToken)
			},
			expected: []string{
				"< GET " + server.URL + "/token -> 200 OK",
				"< Content-Type: application/json; charset=utf-8",
				"< X-Multi-Value: first",
				"< X-Multi-Value: second",
				"< X-Test-Header: test-value",
			},
		},
		{
			// The export endpoints use SetDoNotParseResponse(true), which makes resty
			// skip its after-response middlewares. The transport wrapper still sees them.
			name: "export call streaming the body",
			call: func(t *testing.T, c *client) {
				stream, err := c.ExportTimeSeries(meteringPointIDs, from, to, Hour)
				assert.NoError(t, err)
				assert.NotNil(t, stream)
				defer func() { _ = stream.Close() }()

				content, err := io.ReadAll(stream)
				assert.NoError(t, err)
				assert.Equal(t, exportedCSV, string(content), "the response body must not be consumed by the header printer")
			},
			expected: []string{
				"< GET " + server.URL + "/token -> 200 OK",
				"< POST " + server.URL + exportPath + " -> 200 OK",
				"< X-Test-Header: test-value",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			c := newTestCustomer(t, server.URL, WithResponseHeaderOutput(&buf))

			test.call(t, c)

			out := buf.String()
			for _, expected := range test.expected {
				assert.Contains(t, out, expected)
			}

			// Header keys are sorted alphabetically
			assert.Less(t, strings.Index(out, "< Content-Type:"), strings.Index(out, "< X-Test-Header:"))

			// Every block ends with a blank line
			assert.True(t, strings.HasSuffix(out, "\n\n"))
		})
	}
}

func TestWithResponseHeaderOutputDefaults(t *testing.T) {
	server := newTestServer(t)

	t.Run("client without the option does not wrap the transport and still works", func(t *testing.T) {
		c := newTestCustomer(t, server.URL)

		_, wrapped := c.resty.GetClient().Transport.(*responseHeaderPrinter)
		assert.False(t, wrapped, "transport should not be wrapped without the option")

		accessToken, err := c.GetDataAccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "test-access-token", accessToken)
	})

	t.Run("a nil writer does not wrap the transport", func(t *testing.T) {
		c := newTestCustomer(t, server.URL, WithResponseHeaderOutput(nil))

		_, wrapped := c.resty.GetClient().Transport.(*responseHeaderPrinter)
		assert.False(t, wrapped, "transport should not be wrapped for a nil writer")
	})

	t.Run("write errors do not break the API call", func(t *testing.T) {
		c := newTestCustomer(t, server.URL, WithResponseHeaderOutput(failingWriter{}))

		accessToken, err := c.GetDataAccessToken()
		assert.NoError(t, err)
		assert.Equal(t, "test-access-token", accessToken)
	})
}

// failingWriter always fails, to verify that write errors are ignored.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

// testRetryWait keeps the retry tests quick. It caps both the backoff and any wait asked
// for by a Retry-After header, so no test ever sleeps for the real, CLI-sized defaults.
const testRetryWait = 10 * time.Millisecond

// newMockedCustomer creates a Customer client whose requests are served by httpmock.
func newMockedCustomer(t *testing.T, opts ...Option) *client {
	t.Helper()

	c, ok := NewCustomer("test-refresh-token", opts...).(*client)
	assert.True(t, ok, "client should be of internal type *client")

	httpmock.ActivateNonDefault(c.resty.GetClient())
	t.Cleanup(httpmock.DeactivateAndReset)

	return c
}

// tokenURL is the absolute URL of the token endpoint, as httpmock sees it once the
// client has resolved it against its base URL.
func tokenURL(c *client) string {
	return c.resty.BaseURL + "/token"
}

// tokenResponder answers the token endpoint with the given statuses in order, repeating
// the last one once they run out. A 200 carries a valid token.
func tokenResponder(statuses ...int) httpmock.Responder {
	var calls int

	return func(req *http.Request) (*http.Response, error) {
		status := statuses[len(statuses)-1]
		if calls < len(statuses) {
			status = statuses[calls]
		}
		calls++

		body := ""
		if status == http.StatusOK {
			body = `{"result": "fake-access-token"}`
		}

		res := httpmock.NewStringResponse(status, body)
		res.Header.Set("Content-Type", "application/json")
		return res, nil
	}
}

func TestRetryPolicyDefaults(t *testing.T) {
	clients := map[string]*client{
		"customer":   NewCustomer("test-refresh-token").(*client),
		"thirdparty": NewThirdParty("test-refresh-token").(*client),
	}

	for name, c := range clients {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, DefaultRetryCount, c.resty.RetryCount)
			assert.Equal(t, DefaultRetryWait, c.resty.RetryWaitTime)
			assert.Equal(t, DefaultRetryMaxWait, c.resty.RetryMaxWaitTime)
			assert.NotNil(t, c.resty.RetryAfter, "the Retry-After header must be honoured")
			assert.Len(t, c.resty.RetryConditions, 1, "exactly one retry condition")

			// A CLI must not appear to hang: the retries stay well inside a minute when
			// the API does not ask for a longer wait
			assert.LessOrEqual(t, c.resty.RetryCount, 3)
			assert.LessOrEqual(t, c.resty.RetryMaxWaitTime, time.Minute)
		})
	}
}

// TestRetryOnlyTransientStatuses is the regression test for the missing retry: the API
// answers 429 when a rate limit is exceeded and 503 when DataHub is overloaded, and both
// used to be handed straight to the caller. Permanent failures must still not be retried.
func TestRetryOnlyTransientStatuses(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		options       []Option
		expectedCalls int
		expectedError error
	}{
		{
			name:          "429 is retried",
			status:        http.StatusTooManyRequests,
			options:       []Option{WithRetry(DefaultRetryCount, testRetryWait)},
			expectedCalls: 3,
			expectedError: ErrorTooManyRequests,
		},
		{
			name:          "503 is retried",
			status:        http.StatusServiceUnavailable,
			options:       []Option{WithRetry(DefaultRetryCount, testRetryWait)},
			expectedCalls: 3,
			expectedError: ErrorClientConnection(http.StatusServiceUnavailable),
		},
		{
			name:          "401 is not retried",
			status:        http.StatusUnauthorized,
			options:       []Option{WithRetry(DefaultRetryCount, testRetryWait)},
			expectedCalls: 1,
			expectedError: ErrorUnauthorized,
		},
		{
			name:          "400 is not retried",
			status:        http.StatusBadRequest,
			options:       []Option{WithRetry(DefaultRetryCount, testRetryWait)},
			expectedCalls: 1,
			expectedError: ErrorClientConnection(http.StatusBadRequest),
		},
		{
			name:          "500 is not retried",
			status:        http.StatusInternalServerError,
			options:       []Option{WithRetry(DefaultRetryCount, testRetryWait)},
			expectedCalls: 1,
			expectedError: ErrorClientConnection(http.StatusInternalServerError),
		},
		{
			name:          "WithoutRetry gives up on the first 429",
			status:        http.StatusTooManyRequests,
			options:       []Option{WithoutRetry()},
			expectedCalls: 1,
			expectedError: ErrorTooManyRequests,
		},
		{
			name:          "WithRetry can ask for more attempts",
			status:        http.StatusTooManyRequests,
			options:       []Option{WithRetry(4, testRetryWait)},
			expectedCalls: 5,
			expectedError: ErrorTooManyRequests,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newMockedCustomer(t, test.options...)
			httpmock.RegisterResponder("GET", tokenURL(c), tokenResponder(test.status))

			start := time.Now()
			_, err := c.GetDataAccessToken()
			elapsed := time.Since(start)

			assert.Error(t, err)
			assert.EqualError(t, err, test.expectedError.Error())
			assert.Equal(t, test.expectedCalls, httpmock.GetTotalCallCount())
			assert.Less(t, elapsed, time.Second, "the test retry policy must not sleep for the real defaults")
		})
	}
}

// TestRetryRecovers covers the point of retrying: a rate limit that clears is invisible
// to the caller.
func TestRetryRecovers(t *testing.T) {
	c := newMockedCustomer(t, WithRetry(DefaultRetryCount, testRetryWait))
	httpmock.RegisterResponder("GET", tokenURL(c),
		tokenResponder(http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusOK))

	token, err := c.GetDataAccessToken()

	assert.NoError(t, err)
	assert.Equal(t, "fake-access-token", token)
	assert.Equal(t, 3, httpmock.GetTotalCallCount())
}

func TestWithRetryClampsItsArguments(t *testing.T) {
	t.Run("a negative count disables retrying", func(t *testing.T) {
		c := newMockedCustomer(t, WithRetry(-1, testRetryWait))
		assert.Equal(t, 0, c.resty.RetryCount)
	})

	t.Run("a zero max wait falls back to the default", func(t *testing.T) {
		c := newMockedCustomer(t, WithRetry(DefaultRetryCount, 0))
		assert.Equal(t, DefaultRetryMaxWait, c.resty.RetryMaxWaitTime)
		assert.Equal(t, DefaultRetryWait, c.resty.RetryWaitTime)
	})

	t.Run("the base backoff never exceeds the cap", func(t *testing.T) {
		c := newMockedCustomer(t, WithRetry(DefaultRetryCount, testRetryWait))
		assert.Equal(t, testRetryWait, c.resty.RetryMaxWaitTime)
		assert.Equal(t, testRetryWait, c.resty.RetryWaitTime)
	})

	t.Run("re-applying the policy does not stack retry conditions", func(t *testing.T) {
		c := newMockedCustomer(t, WithRetry(1, testRetryWait), WithoutRetry())
		assert.Len(t, c.resty.RetryConditions, 1)
		assert.Equal(t, 0, c.resty.RetryCount)
	})
}

func TestRetryAfterHeader(t *testing.T) {
	// resty caps whatever this returns at the client's max wait time, so the tests can
	// assert the raw value the header asks for
	response := func(header string) *resty.Response {
		raw := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
		if header != "" {
			raw.Header.Set("Retry-After", header)
		}
		return &resty.Response{RawResponse: raw}
	}

	t.Run("honours a delay in seconds", func(t *testing.T) {
		wait, err := retryAfter(nil, response("60"))
		assert.NoError(t, err)
		assert.Equal(t, time.Minute, wait)
	})

	t.Run("honours an HTTP-date", func(t *testing.T) {
		wait, err := retryAfter(nil, response(time.Now().Add(30*time.Second).UTC().Format(http.TimeFormat)))
		assert.NoError(t, err)
		assert.Greater(t, wait, 25*time.Second)
		assert.LessOrEqual(t, wait, 30*time.Second)
	})

	t.Run("falls back to the backoff when the header is unusable", func(t *testing.T) {
		for _, header := range []string{"", "soon", "0", "-5", "Mon, 02 Jan 2006 15:04:05 GMT"} {
			wait, err := retryAfter(nil, response(header))
			assert.NoError(t, err)
			assert.Zero(t, wait, "an unusable Retry-After (%q) means: use the default backoff", header)
		}
	})

	t.Run("is capped by the max wait time", func(t *testing.T) {
		c := newMockedCustomer(t, WithRetry(1, testRetryWait))
		httpmock.RegisterResponder("GET", tokenURL(c),
			func(req *http.Request) (*http.Response, error) {
				res := httpmock.NewStringResponse(http.StatusTooManyRequests, "")
				// Far longer than the client is willing to wait
				res.Header.Set("Retry-After", "3600")
				return res, nil
			})

		start := time.Now()
		_, err := c.GetDataAccessToken()
		elapsed := time.Since(start)

		assert.Equal(t, ErrorTooManyRequests, err)
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
		assert.Less(t, elapsed, time.Second, "a Retry-After beyond the cap must not be waited out")
	})
}

func TestRetryCondition(t *testing.T) {
	t.Run("no response is not retried", func(t *testing.T) {
		assert.False(t, retryCondition(nil, nil))
	})

	t.Run("retries the transient statuses only", func(t *testing.T) {
		for status, expected := range map[int]bool{
			http.StatusOK:                  false,
			http.StatusBadRequest:          false,
			http.StatusUnauthorized:        false,
			http.StatusNotFound:            false,
			http.StatusInternalServerError: false,
			http.StatusTooManyRequests:     true,
			http.StatusServiceUnavailable:  true,
		} {
			res := &resty.Response{RawResponse: &http.Response{StatusCode: status, Header: http.Header{}}}
			assert.Equal(t, expected, retryCondition(res, nil), "status %d", status)
		}
	})
}
