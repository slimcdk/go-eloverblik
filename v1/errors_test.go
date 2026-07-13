package eloverblik

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// problemDocument404 is the body the live API answers a request to an endpoint it does not
// deploy with, verbatim. getchargelinkswithcharges is such an endpoint: both OpenAPI specs
// declare it, both APIs answer 404 for it, and this is what comes back.
const problemDocument404 = `{"type":"https://tools.ietf.org/html/rfc9110#section-15.5.5","title":"Not Found","status":404,` +
	`"traceId":"00-9c485a3a3ed458eab22cab724111db63-ed7aa1e057161e52-01"}`

// TestApiErrorBodyUnmarshal covers the two shapes an error body arrives in. The problem
// document used to make resty log "Cannot unmarshal response body" and leave the message
// empty, which turned every 404 into a bare "could't connect to eloverblik: 404".
func TestApiErrorBodyUnmarshal(t *testing.T) {
	t.Run("reads the API error message out of a bare JSON string", func(t *testing.T) {
		var body apiErrorBody
		require.NoError(t, json.Unmarshal([]byte(`"[20010] Relation not found"`), &body))
		assert.Equal(t, "[20010] Relation not found", body.Message)
		assert.Nil(t, body.Problem)
	})

	t.Run("reads the problem document the API answers a 404 with", func(t *testing.T) {
		var body apiErrorBody
		require.NoError(t, json.Unmarshal([]byte(problemDocument404), &body))
		assert.Empty(t, body.Message)

		require.NotNil(t, body.Problem)
		assert.Equal(t, "https://tools.ietf.org/html/rfc9110#section-15.5.5", body.Problem.Type)
		assert.Equal(t, "Not Found", body.Problem.Title)
		assert.Equal(t, 404, body.Problem.Status)
		assert.Equal(t, "00-9c485a3a3ed458eab22cab724111db63-ed7aa1e057161e52-01", body.Problem.TraceID)
	})

	t.Run("reads the detail and the validation errors of a 400", func(t *testing.T) {
		var body apiErrorBody
		require.NoError(t, json.Unmarshal([]byte(`{"title":"One or more validation errors occurred.","status":400,
			"detail":"The dateFrom field is invalid.","instance":"/customerapi/api/meterdata/gettimeseries",
			"traceId":"00-abc-def-01","errors":{"dateFrom":["Not a date."]}}`), &body))

		require.NotNil(t, body.Problem)
		assert.Equal(t, "The dateFrom field is invalid.", body.Problem.Detail)
		assert.Equal(t, "/customerapi/api/meterdata/gettimeseries", body.Problem.Instance)
		assert.Equal(t, map[string][]string{"dateFrom": {"Not a date."}}, body.Problem.Errors)
	})

	t.Run("never fails, so a body it cannot read makes no resty warning", func(t *testing.T) {
		for _, raw := range []string{`null`, `[]`, `1`, `{`, `"`, ``} {
			var body apiErrorBody
			assert.NoError(t, body.UnmarshalJSON([]byte(raw)), "body %q", raw)
			assert.Empty(t, body.Message, "body %q", raw)
			assert.Nil(t, body.Problem, "body %q", raw)
		}
	})
}

// TestApiErrorFromBody is the entry point every request uses. It has to keep answering
// exactly as before for the string form, and report the problem document form in a way that
// keeps the status, the title and the trace ID, which is what Energinet support asks for.
func TestApiErrorFromBody(t *testing.T) {
	t.Run("reports the problem document of a 404 with everything worth keeping", func(t *testing.T) {
		var body apiErrorBody
		require.NoError(t, json.Unmarshal([]byte(problemDocument404), &body))

		err := apiErrorFromBody(body, 404)
		require.Error(t, err)

		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr), "a problem document is reported as an *APIError")
		assert.Equal(t, 404, apiErr.StatusCode)
		assert.Equal(t, "Not Found", apiErr.Title)
		assert.Equal(t, "https://tools.ietf.org/html/rfc9110#section-15.5.5", apiErr.Type)
		assert.Equal(t, "00-9c485a3a3ed458eab22cab724111db63-ed7aa1e057161e52-01", apiErr.TraceID)
		assert.Zero(t, apiErr.Code, "the document carries no API error code")

		// The message a caller who only prints the error gets to see
		assert.EqualError(t, err, "eloverblik: 404 Not Found "+
			"(traceId 00-9c485a3a3ed458eab22cab724111db63-ed7aa1e057161e52-01)")
	})

	t.Run("keeps mapping a business error string to its sentinel", func(t *testing.T) {
		var body apiErrorBody
		require.NoError(t, json.Unmarshal([]byte(`"[20010] Relation not found"`), &body))

		err := apiErrorFromBody(body, 404)
		assert.Equal(t, ErrorRelationNotFound, err)
		assert.ErrorIs(t, err, ErrorRelationNotFound)
	})

	t.Run("judges an empty body by its status", func(t *testing.T) {
		assert.Equal(t, ErrorTooManyRequests, apiErrorFromBody(apiErrorBody{}, 429))
		assert.Equal(t, ErrorUnauthorized, apiErrorFromBody(apiErrorBody{}, 401))
		assert.EqualError(t, apiErrorFromBody(apiErrorBody{}, 503), "could't connect to eloverblik: 503")
		assert.NoError(t, apiErrorFromBody(apiErrorBody{}, 200))
	})

	t.Run("unwraps a problem document to the sentinel of its status", func(t *testing.T) {
		unauthorized := apiErrorFromBody(apiErrorBody{Problem: &problemDetails{
			Title: "Unauthorized", Status: 401, TraceID: "00-abc-def-01",
		}}, 401)
		assert.ErrorIs(t, unauthorized, ErrorUnauthorized, "errors.Is keeps working on both shapes")

		var apiErr *APIError
		require.True(t, errors.As(unauthorized, &apiErr))
		assert.Equal(t, "00-abc-def-01", apiErr.TraceID, "the trace ID survives the wrapping")

		rateLimited := apiErrorFromBody(apiErrorBody{Problem: &problemDetails{Title: "Too Many Requests"}}, 429)
		assert.ErrorIs(t, rateLimited, ErrorTooManyRequests)
		assert.EqualError(t, rateLimited, "eloverblik: 429 Too Many Requests")
	})

	t.Run("unwraps a problem document to the sentinel of a code in its detail", func(t *testing.T) {
		err := apiErrorFromBody(apiErrorBody{Problem: &problemDetails{
			Title: "Bad Request", Status: 400, Detail: "[30004] Invalid date format in request",
		}}, 400)
		assert.ErrorIs(t, err, ErrorInvalidDateFormat)

		var apiErr *APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, uint64(30004), apiErr.Code)
		assert.EqualError(t, err, "eloverblik: 400 Bad Request: [30004] Invalid date format in request")
	})

	t.Run("does not report a problem document on a successful status", func(t *testing.T) {
		assert.NoError(t, apiErrorFromBody(apiErrorBody{Problem: &problemDetails{Title: "Not Found"}}, 200))
	})
}

func TestApiErrorCode(t *testing.T) {
	t.Run("reads the code out of an API error message", func(t *testing.T) {
		code, ok := apiErrorCode("[20010] Relation not found")
		assert.True(t, ok)
		assert.Equal(t, uint64(20010), code)
	})

	t.Run("reports no code for a message that carries none", func(t *testing.T) {
		for _, msg := range []string{"", "no", "Not Found", "Request 20010 failed", "[abcde] nonsense"} {
			_, ok := apiErrorCode(msg)
			assert.False(t, ok, "message %q", msg)
		}
	})
}

func TestApiError(t *testing.T) {
	t.Run("returns nil for success code", func(t *testing.T) {
		// The API returns a string like "[10000] No error" on success
		err := apiError("[10000] No error", 200)
		assert.NoError(t, err)
	})

	t.Run("returns correct error for known error code", func(t *testing.T) {
		// Example error for an invalid metering point ID
		err := apiError("[20003] Metering point ID must be 18 characters long", 400)
		assert.Error(t, err)
		assert.Equal(t, ErrorMeteringPointIdNot18CharsLong, err)
	})

	t.Run("returns specific error for unauthorized", func(t *testing.T) {
		err := apiError("[20012] Unauthorized access", 401)
		assert.Error(t, err)
		assert.Equal(t, ErrorUnauthorized, err)
	})

	t.Run("returns a formatted error for unknown codes", func(t *testing.T) {
		err := apiError("[99999] A completely new and unknown error", 400)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unhandled error: '[99999] A completely new and unknown error'")
	})
}

func TestErrorClientConnection(t *testing.T) {
	t.Run("returns formatted connection error", func(t *testing.T) {
		err := ErrorClientConnection(503)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could't connect to eloverblik: 503")
	})

	t.Run("handles different status codes", func(t *testing.T) {
		err := ErrorClientConnection(404)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func TestIsRetryableError(t *testing.T) {
	t.Run("returns false when no error and status OK", func(t *testing.T) {
		assert.False(t, isRetryableError(200, nil))
	})

	t.Run("returns false on 200 OK even with error object", func(t *testing.T) {
		err := assert.AnError
		assert.False(t, isRetryableError(200, err))
	})

	t.Run("returns true for the transient statuses documented by the API", func(t *testing.T) {
		assert.True(t, isRetryableError(429, nil), "rate limit exceeded")
		assert.True(t, isRetryableError(503, nil), "DataHub unavailable")
	})

	t.Run("returns false for permanent failures", func(t *testing.T) {
		assert.False(t, isRetryableError(400, nil))
		assert.False(t, isRetryableError(401, nil))
		assert.False(t, isRetryableError(404, nil))
		assert.False(t, isRetryableError(500, nil))
	})

	t.Run("returns false for transport errors, so no request is repeated by accident", func(t *testing.T) {
		err := assert.AnError
		assert.False(t, isRetryableError(429, err))
		assert.False(t, isRetryableError(503, err))
	})
}

// TestApiErrorWithoutMessage guards responses that carry no parseable API error message.
// They used to be reported as success, which made an expired token look like an empty
// result and a rate limit look like "no metering points".
func TestApiErrorWithoutMessage(t *testing.T) {
	t.Run("returns nil on a successful status", func(t *testing.T) {
		assert.NoError(t, apiError("", 200))
		assert.NoError(t, apiError("", 204))
	})

	t.Run("returns an error on a failed status", func(t *testing.T) {
		assert.Equal(t, ErrorTooManyRequests, apiError("", 429))
		assert.Equal(t, ErrorUnauthorized, apiError("", 401))
		assert.EqualError(t, apiError("", 503), "could't connect to eloverblik: 503")
	})

	t.Run("returns an error for an unknown code on any status", func(t *testing.T) {
		assert.Error(t, apiError("[99999] Unknown", 429))
	})

	t.Run("does not panic on a message too short to hold a code", func(t *testing.T) {
		assert.Error(t, apiError("no", 400))
	})
}

// TestApiErrorNoCprConsent covers error code 10007, returned when the token owner has
// not granted consent for CPR lookup. It is what the includeAll=true path runs into.
func TestApiErrorNoCprConsent(t *testing.T) {
	err := apiError("[10007] Missing consent for CPR lookup", 403)
	assert.Error(t, err)
	assert.Equal(t, ErrorNoCprConsent, err)
}
