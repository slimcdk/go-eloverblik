package eloverblik

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
