package eloverblik

import (
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

// TestFailedExport covers an export the API refuses. The body of an export is streamed to
// the caller, so the client never reads it; when the export fails nobody else will either,
// so the client must close it, and report the failure the way every other call does.
func TestFailedExport(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	ids := []string{"571313180100000001"}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, cph)
	to := time.Date(2026, 9, 2, 0, 0, 0, 0, cph)

	exports := []struct {
		name   string
		export func() (io.ReadCloser, error)
	}{
		{"ExportMasterdata", func() (io.ReadCloser, error) { return c.ExportMasterdata(ids) }},
		{"ExportCharges", func() (io.ReadCloser, error) { return c.ExportCharges(ids) }},
		{"ExportTimeSeries", func() (io.ReadCloser, error) { return c.ExportTimeSeries(ids, from, to, Hour) }},
	}

	for _, export := range exports {
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
