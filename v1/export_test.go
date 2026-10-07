package eloverblik

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// closeRecorder is a response body that records whether it was closed.
type closeRecorder struct {
	io.Reader
	closed bool
}

func (b *closeRecorder) Close() error {
	b.closed = true
	return nil
}

// exportCall is one of the export methods, bound to its arguments.
type exportCall struct {
	name   string
	export func() (io.ReadCloser, error)
}

// exportCalls binds every export method of c to a metering point and a period.
func exportCalls(c *client) []exportCall {
	ids := []string{"571313180100000001"}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, cph)
	to := time.Date(2026, 9, 2, 0, 0, 0, 0, cph)

	return []exportCall{
		{"ExportMasterdata", func() (io.ReadCloser, error) { return c.ExportMasterdata(ids) }},
		{"ExportCharges", func() (io.ReadCloser, error) { return c.ExportCharges(ids) }},
		{"ExportTimeSeries", func() (io.ReadCloser, error) { return c.ExportTimeSeries(ids, from, to, Hour) }},
	}
}

// TestFailedExport covers an export the API refuses. The body of an export is streamed to
// the caller, so the client never reads it; when the export fails nobody else will either,
// so the client must close it, and report the failure through apiErrorFromBody, wrapped as
// "failed to export <what>: ...", so errors.Is still finds its sentinel.
func TestFailedExport(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	for _, export := range exportCalls(c) {
		t.Run(export.name, func(t *testing.T) {
			body := &closeRecorder{Reader: strings.NewReader(`"[20012] Unauthorized access"`)}
			httpmock.Reset()
			httpmock.RegisterNoResponder(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Status:     "401 Unauthorized",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       body,
				}, nil
			})

			stream, err := export.export()

			assert.Nil(t, stream)
			require.ErrorIs(t, err, ErrorUnauthorized)
			assert.True(t, body.closed, "the body of a failed export must be closed")
		})
	}
}

// trackedBody is a response body that records whether it was read to the end and whether
// it was closed. Like a body from net/http, it cannot be read once it is closed.
type trackedBody struct {
	data   *strings.Reader
	closed bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	if b.closed {
		return 0, http.ErrBodyReadAfterClose
	}
	return b.data.Read(p)
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

// drained reports whether the body was read to the end.
func (b *trackedBody) drained() bool {
	return b.data.Len() == 0
}

// attempt is the response the API answers one attempt of a request with.
type attempt struct {
	status int
	body   string
}

// rateLimited is a 429 carrying the problem document the API sends with it. The traceId
// tells the attempts apart.
func rateLimited(traceID string) attempt {
	return attempt{
		status: http.StatusTooManyRequests,
		body:   fmt.Sprintf(`{"title":"Too Many Requests","status":429,"traceId":%q}`, traceID),
	}
}

// exported is a successful export.
func exported(body string) attempt {
	return attempt{status: http.StatusOK, body: body}
}

// attemptRecorder answers the attempts of a request in order, and keeps the body of every
// response it hands out so a test can check what became of it.
type attemptRecorder struct {
	attempts []attempt
	bodies   []*trackedBody
}

func (r *attemptRecorder) respond(*http.Request) (*http.Response, error) {
	if len(r.bodies) == len(r.attempts) {
		return nil, fmt.Errorf("unexpected attempt %d", len(r.bodies)+1)
	}

	next := r.attempts[len(r.bodies)]
	body := &trackedBody{data: strings.NewReader(next.body)}
	r.bodies = append(r.bodies, body)

	return &http.Response{
		StatusCode: next.status,
		Status:     fmt.Sprintf("%d %s", next.status, http.StatusText(next.status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       body,
	}, nil
}

// TestExportRetryReleasesDiscardedBodies is the regression test for the bodies of retried
// export attempts. An export is sent with SetDoNotParseResponse, so resty leaves every
// body it receives to the client; when it retries a 429 or 503 it drops the response of
// that attempt, and its body used to be left open, leaking the connection. Each discarded
// body must be read to the end, so the connection can be reused, and closed. The body of
// the last attempt must be left alone: the caller streams it on success, and on failure
// the client reads the API's error message from it before closing it.
func TestExportRetryReleasesDiscardedBodies(t *testing.T) {
	tests := []struct {
		name     string
		attempts []attempt
		// export is the body the caller must get to stream; empty when the export fails
		export string
		// traceID is the traceId of the error the export fails with
		traceID string
	}{
		{
			name:     "429, 429, 200 hands the caller the final body",
			attempts: []attempt{rateLimited("attempt-1"), rateLimited("attempt-2"), exported(exportedCSV)},
			export:   exportedCSV,
		},
		{
			name:     "429, 429, 429 exhausts the retries",
			attempts: []attempt{rateLimited("attempt-1"), rateLimited("attempt-2"), rateLimited("attempt-3")},
			traceID:  "attempt-3",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newMockedCustomer(t, WithRetry(len(test.attempts)-1, testRetryWait))
			c.accessToken = "test-access-token"

			for _, export := range exportCalls(c) {
				t.Run(export.name, func(t *testing.T) {
					recorder := &attemptRecorder{attempts: test.attempts}
					httpmock.Reset()
					httpmock.RegisterNoResponder(recorder.respond)

					stream, err := export.export()

					require.Len(t, recorder.bodies, len(test.attempts), "every attempt must be made")
					discarded, last := recorder.bodies[:len(recorder.bodies)-1], recorder.bodies[len(recorder.bodies)-1]
					for i, body := range discarded {
						assert.True(t, body.drained(), "the body of discarded attempt %d must be read to the end", i+1)
						assert.True(t, body.closed, "the body of discarded attempt %d must be closed", i+1)
					}

					if test.export != "" {
						require.NoError(t, err)
						require.NotNil(t, stream)
						assert.False(t, last.closed, "the caller must get the body of the last attempt open")

						data, err := io.ReadAll(stream)
						require.NoError(t, err)
						assert.Equal(t, test.export, string(data))

						require.NoError(t, stream.Close())
						assert.True(t, last.closed, "closing the stream must close the body of the last attempt")
						return
					}

					assert.Nil(t, stream)
					require.ErrorIs(t, err, ErrorTooManyRequests)
					var apiErr *APIError
					require.ErrorAs(t, err, &apiErr)
					assert.Equal(t, test.traceID, apiErr.TraceID, "the error must be read from the body of the last attempt")
					assert.True(t, last.closed, "the body of the last attempt must be closed when the export fails")
				})
			}
		})
	}
}

// TestExportRetryDrainIsBounded covers a misbehaving server that answers a retried attempt
// with a huge body. Draining it would cost more than the connection it saves, so the
// client must close it without reading it to the end.
func TestExportRetryDrainIsBounded(t *testing.T) {
	c := newMockedCustomer(t, WithRetry(1, testRetryWait))
	c.accessToken = "test-access-token"

	recorder := &attemptRecorder{attempts: []attempt{
		{status: http.StatusServiceUnavailable, body: strings.Repeat("x", 8<<20)},
		exported(exportedCSV),
	}}
	httpmock.RegisterNoResponder(recorder.respond)

	stream, err := c.ExportMasterdata([]string{"571313180100000001"})
	require.NoError(t, err)
	defer func() { _ = stream.Close() }()

	require.Len(t, recorder.bodies, 2)
	assert.False(t, recorder.bodies[0].drained(), "a huge discarded body must not be read to the end")
	assert.True(t, recorder.bodies[0].closed, "a huge discarded body must still be closed")
}
