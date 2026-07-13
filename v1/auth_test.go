package eloverblik

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetDataAccessToken(t *testing.T) {
	// Create a new client with a mock resty client
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		refreshToken: "test-refresh-token",
		resty:        mockResty,
	}

	t.Run("successfully authenticates and retrieves token", func(t *testing.T) {
		// Mock the API response for the /token endpoint
		expectedToken := "fake-access-token"
		response := map[string]string{"result": expectedToken}
		httpmock.RegisterResponder("GET", "/token",
			func(req *http.Request) (*http.Response, error) {
				// Check if the refresh token is sent correctly
				assert.Equal(t, "Bearer "+c.refreshToken, req.Header.Get("Authorization"))
				resp, err := httpmock.NewJsonResponse(200, response)
				return resp, err
			},
		)

		// Call the function to test
		token, err := c.GetDataAccessToken()

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, expectedToken, token)
		assert.Equal(t, expectedToken, c.accessToken, "Access token should be stored in the client struct")
	})

	t.Run("returns cached token on subsequent calls", func(t *testing.T) {
		// Reset call count and set an existing access token
		httpmock.Reset()
		c.accessToken = "already-cached-token"

		// Call the function again
		token, err := c.GetDataAccessToken()

		// Assertions
		assert.NoError(t, err)
		assert.Equal(t, "already-cached-token", token)
		assert.Equal(t, 0, httpmock.GetTotalCallCount(), "authenticate() should not be called if token is cached")
	})
}

// TestAuthenticateFailure guards the token endpoint. Any non-200 used to be swallowed:
// authenticate() returned a nil error, the access token stayed empty and the caller went
// on to make unauthenticated requests.
func TestAuthenticateFailure(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name        string
		status      int
		body        string
		contentType string
		expected    error
	}{
		{
			name:        "expired or revoked refresh token",
			status:      http.StatusUnauthorized,
			body:        `"[50001] Token is invalid"`,
			contentType: "application/json",
			expected:    ErrorTokenNotValid,
		},
		{
			name:     "unauthorized without an API error message",
			status:   http.StatusUnauthorized,
			expected: ErrorUnauthorized,
		},
		{
			name:     "rate limited",
			status:   http.StatusTooManyRequests,
			expected: ErrorTooManyRequests,
		},
		{
			name:     "datahub unavailable",
			status:   http.StatusServiceUnavailable,
			expected: ErrorClientConnection(http.StatusServiceUnavailable),
		},
		{
			name:        "success without a token",
			status:      http.StatusOK,
			body:        `{"result": ""}`,
			contentType: "application/json",
			expected:    ErrorErrorCreatingToken,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("GET", "/token",
				func(req *http.Request) (*http.Response, error) {
					resp := httpmock.NewStringResponse(test.status, test.body)
					if test.contentType != "" {
						resp.Header.Set("Content-Type", test.contentType)
					}
					return resp, nil
				})

			// Retrying is disabled so the transient statuses do not slow the test down
			c := &client{refreshToken: "test-refresh-token", resty: mockResty}

			token, err := c.GetDataAccessToken()

			assert.Error(t, err)
			assert.EqualError(t, err, test.expected.Error())
			assert.Empty(t, token)
			assert.Empty(t, c.accessToken, "no access token may be stored when authentication fails")
		})
	}
}

func TestGetAuthorizations(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     ThirdPartyApi,
	}

	t.Run("successfully gets authorizations", func(t *testing.T) {
		httpmock.Reset()
		mockResponse := `{
			"result": [
				{
					"id": "auth-uuid-1",
					"thirdPartyName": "Test Corp",
					"validFrom": "2024-01-01",
					"validTo": "2024-12-31",
					"customerName": "Test Customer",
					"customerCVR": "12345678",
					"customerKey": "test-key",
					"includeFutureMeteringPoints": false,
					"timeStamp": "2024-01-01T10:00:00Z"
				}
			]
		}`
		httpmock.RegisterResponder("GET", "/authorization/authorizations",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		authorizations, err := c.GetAuthorizations()

		assert.NoError(t, err)
		if assert.Len(t, authorizations, 1) {
			assert.Equal(t, "auth-uuid-1", authorizations[0].ID)
			assert.Equal(t, "Test Corp", authorizations[0].ThirdPartyName)
		}
	})

	t.Run("returns error for customer API", func(t *testing.T) {
		customerClient := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}

		_, err := customerClient.GetAuthorizations()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available for ThirdParty API")
	})

	t.Run("handles API error response", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "/authorization/authorizations",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(401, `"[20012] Unauthorized"`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		_, err := c.GetAuthorizations()

		assert.Error(t, err)
		assert.Equal(t, ErrorUnauthorized, err)
	})
}

func TestGetMeteringPointsForScope(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     ThirdPartyApi,
	}

	t.Run("successfully gets metering points for scope", func(t *testing.T) {
		httpmock.Reset()
		mockResponse := `{
			"result": [
				{ "meteringPointId": "571313180100000001", "typeOfMP": "E17" }
			]
		}`
		path := "/authorization/authorization/meteringpoints/customerCVR/12345678"
		httpmock.RegisterResponder("GET", path,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		meteringPoints, err := c.GetMeteringPointsForScope(AuthScopeCustomerCVR, "12345678")

		assert.NoError(t, err)
		if assert.Len(t, meteringPoints, 1) {
			assert.Equal(t, "571313180100000001", meteringPoints[0].MeteringPointID)
		}
	})

	t.Run("returns error for customer API", func(t *testing.T) {
		customerClient := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}

		_, err := customerClient.GetMeteringPointsForScope(AuthScopeCustomerCVR, "12345678")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available for ThirdParty API")
	})
}

// TestGetMeteringPointsForScopeFullPayload decodes a payload shaped like a real
// /authorization/authorization/meteringpoints response. The struct used to model only the
// identity and address of a metering point, dropping the meter number, the CVR numbers, the
// consumer names, the settlement method, the reading occurrence, the location description
// and the child metering points the API actually returns.
func TestGetMeteringPointsForScopeFullPayload(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     ThirdPartyApi,
	}

	httpmock.Reset()
	mockResponse := `{
		"result": [
			{
				"meteringPointId": "571313113162842251",
				"typeOfMP": "E17",
				"accessFrom": "2024-10-31T23:00:00.000Z",
				"accessTo": "2026-07-31T22:00:00.000Z",
				"streetCode": "0116",
				"streetName": "Blichers Alle",
				"buildingNumber": "1",
				"floorId": "",
				"roomId": "",
				"postcode": "8830",
				"cityName": "Tjele",
				"citySubDivisionName": "Foulum",
				"municipalityCode": "791",
				"locationDescription": "Bag ved laden",
				"settlementMethod": "D01",
				"meterReadingOccurrence": "PT1H",
				"firstConsumerPartyName": "John Sisk & Son ApS",
				"secondConsumerPartyName": "",
				"consumerCVR": "42703087",
				"dataAccessCVR": "42703087",
				"meterNumber": "30203518",
				"consumerStartDate": "2025-04-27T22:00:00.000Z",
				"childMeteringPoints": [
					{
						"parentMeteringPointId": "571313113162842251",
						"meteringPointId": "571313113162842268",
						"typeOfMP": "D01",
						"meterReadingOccurrence": "PT1H",
						"meterNumber": "30203519"
					}
				]
			}
		]
	}`
	path := "/authorization/authorization/meteringpoints/authorizationId/725809"
	httpmock.RegisterResponder("GET", path,
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, mockResponse)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		})

	meteringPoints, err := c.GetMeteringPointsForScope(AuthScopeID, "725809")

	assert.NoError(t, err)
	if !assert.Len(t, meteringPoints, 1) {
		return
	}
	meteringPoint := meteringPoints[0]

	assert.Equal(t, "571313113162842251", meteringPoint.MeteringPointID)
	assert.Equal(t, "Foulum", meteringPoint.CitySubDivisionName)
	assert.Equal(t, "791", meteringPoint.MunicipalityCode)
	assert.Equal(t, "Bag ved laden", meteringPoint.LocationDescription)
	assert.Equal(t, "D01", meteringPoint.SettlementMethod)
	assert.Equal(t, "PT1H", meteringPoint.MeterReadingOccurrence)
	assert.Equal(t, "John Sisk & Son ApS", meteringPoint.FirstConsumerPartyName)
	assert.Equal(t, "", meteringPoint.SecondConsumerPartyName)
	assert.Equal(t, "42703087", meteringPoint.ConsumerCVR)
	assert.Equal(t, "42703087", meteringPoint.DataAccessCVR)
	assert.Equal(t, "30203518", meteringPoint.MeterNumber)
	assert.Equal(t, "2025-04-27T22:00:00Z", meteringPoint.ConsumerStartDate.UTC().Format(time.RFC3339))

	if assert.Len(t, meteringPoint.ChildMeteringPoints, 1) {
		child := meteringPoint.ChildMeteringPoints[0]
		assert.Equal(t, "571313113162842268", child.MeteringPointID)
		assert.Equal(t, "571313113162842251", child.ParentMeteringPointID)
		assert.Equal(t, "D01", child.TypeOfMP)
		assert.Equal(t, "30203519", child.MeterNumber)
	}
}

func TestGetMeteringPointIDsForScope(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     ThirdPartyApi,
	}

	t.Run("successfully gets metering point IDs for scope", func(t *testing.T) {
		httpmock.Reset()
		mockResponse := `{
			"result": [
				"571313180100000001",
				"571313180100000002"
			]
		}`
		path := "/authorization/authorization/meteringpointids/customerCVR/12345678"
		httpmock.RegisterResponder("GET", path,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		ids, err := c.GetMeteringPointIDsForScope(AuthScopeCustomerCVR, "12345678")

		assert.NoError(t, err)
		if assert.Len(t, ids, 2) {
			assert.Equal(t, "571313180100000001", ids[0])
			assert.Equal(t, "571313180100000002", ids[1])
		}
	})

	t.Run("returns error for customer API", func(t *testing.T) {
		customerClient := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}

		_, err := customerClient.GetMeteringPointIDsForScope(AuthScopeCustomerCVR, "12345678")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available for ThirdParty API")
	})
}

func TestIsAlive(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{resty: mockResty}

	t.Run("returns true on 200 OK", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "/isalive", httpmock.NewStringResponder(200, "true"))
		alive, err := c.IsAlive()
		assert.NoError(t, err)
		assert.True(t, alive)
	})

	t.Run("returns false on 503 Service Unavailable", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "/isalive", httpmock.NewStringResponder(503, ""))
		alive, err := c.IsAlive()
		assert.NoError(t, err)
		assert.False(t, alive)
	})
}
