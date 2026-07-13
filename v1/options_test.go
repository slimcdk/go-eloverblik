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
