package eloverblik

import (
	"fmt"
	"io"

	"github.com/go-resty/resty/v2"
)

// maxExportErrorBody caps how much of a failed export's body is read for the API's error
// message. Error bodies are small; the cap only guards against a misbehaving server.
const maxExportErrorBody = 64 << 10

// exportBody hands the body of an export to the caller, who streams and closes it. The
// export requests are sent with SetDoNotParseResponse, so resty never touches the body: on
// a failure nobody else would read or close it. It is read here for the API's error
// message, closed, and the failure reported like that of any other call.
func exportBody(res *resty.Response, err error, what string) (io.ReadCloser, error) {
	if err != nil {
		return nil, fmt.Errorf("failed to export %s: %w", what, err)
	}
	if res.IsSuccess() {
		return res.RawBody(), nil
	}

	var errBody apiErrorBody
	if body := res.RawBody(); body != nil {
		defer func() { _ = body.Close() }()
		data, _ := io.ReadAll(io.LimitReader(body, maxExportErrorBody))
		_ = errBody.UnmarshalJSON(data) // never fails, see apiErrorBody
	}

	return nil, fmt.Errorf("failed to export %s: %w", what, apiErrorFromBody(errBody, res.StatusCode()))
}
