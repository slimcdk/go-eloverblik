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

	t.Run("returns false when error is nil regardless of status", func(t *testing.T) {
		assert.False(t, isRetryableError(500, nil))
		assert.False(t, isRetryableError(503, nil))
	})

	t.Run("returns true for non-200 status with error", func(t *testing.T) {
		err := assert.AnError
		assert.True(t, isRetryableError(500, err))
		assert.True(t, isRetryableError(503, err))
		assert.True(t, isRetryableError(429, err))
		assert.True(t, isRetryableError(400, err))
	})
}
