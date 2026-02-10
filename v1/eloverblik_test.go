package eloverblik

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	refreshToken := "test-refresh-token"

	t.Run("creates a Customer client with NewCustomer", func(t *testing.T) {
		c := NewCustomer(refreshToken)
		assert.NotNil(t, c)

		// Use type assertion to access the internal client struct for testing
		internalClient, ok := c.(*client)
		assert.True(t, ok, "client should be of internal type *client")

		expectedURL := "https://" + prodModeHost + "/customerapi/api"
		assert.Equal(t, refreshToken, internalClient.refreshToken)
		assert.Equal(t, CustomerApi, internalClient.apiType)
		assert.Equal(t, expectedURL, internalClient.resty.BaseURL)
	})

	t.Run("creates a ThirdParty client with NewThirdParty", func(t *testing.T) {
		c := NewThirdParty(refreshToken)
		assert.NotNil(t, c)

		// Use type assertion to access the internal client struct for testing
		internalClient, ok := c.(*client)
		assert.True(t, ok, "client should be of internal type *client")
		expectedURL := "https://" + prodModeHost + "/thirdpartyapi/api"
		assert.Equal(t, refreshToken, internalClient.refreshToken)
		assert.Equal(t, ThirdPartyApi, internalClient.apiType)
		assert.Equal(t, expectedURL, internalClient.resty.BaseURL)
	})
}
