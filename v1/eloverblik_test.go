package eloverblik

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
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

// TestApiVersionHeader guards the pinned api-version. Both specs declare the header on
// every operation with a server-side default of "1.0"; the library used to leave it out
// entirely, so a bump of that default could silently change the response shapes.
func TestApiVersionHeader(t *testing.T) {
	clients := map[string]*client{
		"customer":   NewCustomer("test-refresh-token").(*client),
		"thirdparty": NewThirdParty("test-refresh-token").(*client),
	}

	for name, c := range clients {
		t.Run(name+" sends api-version on every request", func(t *testing.T) {
			assert.Equal(t, apiVersion, c.resty.Header.Get(apiVersionHeader))

			httpmock.ActivateNonDefault(c.resty.GetClient())
			defer httpmock.DeactivateAndReset()

			var sent string
			httpmock.RegisterResponder("GET", c.resty.BaseURL+"/token",
				func(req *http.Request) (*http.Response, error) {
					sent = req.Header.Get(apiVersionHeader)
					res := httpmock.NewStringResponse(http.StatusOK, `{"result": "fake-access-token"}`)
					res.Header.Set("Content-Type", "application/json")
					return res, nil
				})

			_, err := c.GetDataAccessToken()

			assert.NoError(t, err)
			assert.Equal(t, "1.0", sent, "the api-version header must reach the wire")
		})
	}
}
