package eloverblik

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)
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
		require.NoError(t, err)
		assert.Equal(t, "already-cached-token", token)
		assert.Equal(t, 0, httpmock.GetTotalCallCount(), "authenticate() should not be called if token is cached")
	})
}

// TestGetDataAccessTokenConcurrent guards the token cache against concurrent use. Every
// goroutine used to find the cache empty and fetch a token of its own, which raced on the
// cached token and spent the 2 calls a minute the API allows on /token many times over.
func TestGetDataAccessTokenConcurrent(t *testing.T) {
	c := NewThirdParty("test-refresh-token", WithoutRetry()).(*client)
	httpmock.ActivateNonDefault(c.resty.GetClient())
	defer httpmock.DeactivateAndReset()

	token := dataAccessToken(t, time.Now().Add(24*time.Hour))
	var tokenCalls atomic.Int32
	httpmock.RegisterResponder(http.MethodGet, c.resty.BaseURL+"/token",
		func(*http.Request) (*http.Response, error) {
			tokenCalls.Add(1)
			// A slow answer keeps the window open in which an unguarded cache is still empty
			time.Sleep(20 * time.Millisecond)
			return httpmock.NewJsonResponse(http.StatusOK, map[string]string{"result": token})
		})
	httpmock.RegisterResponder(http.MethodGet, c.resty.BaseURL+"/authorization/authorizations",
		func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "Bearer "+token {
				return httpmock.NewStringResponse(http.StatusUnauthorized, ""), nil
			}
			return httpmock.NewJsonResponse(http.StatusOK, map[string]any{"result": []any{}})
		})

	const goroutines = 32
	start := make(chan struct{})
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			<-start
			if i%2 == 0 {
				_, err := c.GetAuthorizations()
				errs <- err
				return
			}
			got, err := c.GetDataAccessToken()
			if err == nil && got != token {
				err = fmt.Errorf("got data access token %q, want the one /token issued", got)
			}
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), tokenCalls.Load(), "one client must fetch its data access token once")
}

// TestGetDataAccessTokenRenewal guards the renewal of the cached data access token. It
// lasts about 24 hours, and the client used to keep it for its own lifetime, so a long
// running process saw every call fail with 401 once the token had expired.
func TestGetDataAccessTokenRenewal(t *testing.T) {
	fresh := dataAccessToken(t, time.Now().Add(24*time.Hour))

	// A data access token that carries a token type but no expiry
	withoutExpiry := testToken(t, map[string]any{"tokenType": "ThirdPartyApiDataAccess"})

	tests := []struct {
		name    string
		cached  string
		renewed bool
	}{
		{
			name:    "expired token is renewed",
			cached:  dataAccessToken(t, time.Now().Add(-time.Hour)),
			renewed: true,
		},
		{
			name:    "token expiring within five minutes is renewed",
			cached:  dataAccessToken(t, time.Now().Add(2*time.Minute)),
			renewed: true,
		},
		{
			name:    "token valid for more than five minutes is kept",
			cached:  dataAccessToken(t, time.Now().Add(10*time.Minute)),
			renewed: false,
		},
		{
			name:    "token that is not a JWT is kept",
			cached:  "opaque-access-token",
			renewed: false,
		},
		{
			name:    "token without an expiry is kept",
			cached:  withoutExpiry,
			renewed: false,
		},
		// An exp that names no point in time used to read as 1 January 1970, so the token
		// counted as expired and every call spent one of the 2 /token calls a minute on it
		{
			name:    "token whose exp is null is kept",
			cached:  testToken(t, map[string]any{"tokenType": "ThirdPartyApiDataAccess", "exp": nil}),
			renewed: false,
		},
		{
			name:    "token whose exp is zero is kept",
			cached:  testToken(t, map[string]any{"tokenType": "ThirdPartyApiDataAccess", "exp": 0}),
			renewed: false,
		},
		{
			name:    "token whose exp is negative is kept",
			cached:  testToken(t, map[string]any{"tokenType": "ThirdPartyApiDataAccess", "exp": -1}),
			renewed: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewThirdParty("test-refresh-token", WithoutRetry()).(*client)
			httpmock.ActivateNonDefault(c.resty.GetClient())
			defer httpmock.DeactivateAndReset()

			// The first /token call hands out the token under test, any later one a fresh token
			tokenCalls := 0
			httpmock.RegisterResponder(http.MethodGet, c.resty.BaseURL+"/token",
				func(*http.Request) (*http.Response, error) {
					tokenCalls++
					token := fresh
					if tokenCalls == 1 {
						token = test.cached
					}
					return httpmock.NewJsonResponse(http.StatusOK, map[string]string{"result": token})
				})

			first, err := c.GetDataAccessToken()
			require.NoError(t, err)
			require.Equal(t, test.cached, first)

			second, err := c.GetDataAccessToken()
			require.NoError(t, err)

			if test.renewed {
				assert.Equal(t, fresh, second, "the cached token must be replaced")
				assert.Equal(t, 2, tokenCalls, "the cached token must be renewed with a new /token call")
			} else {
				assert.Equal(t, test.cached, second, "the cached token must be kept")
				assert.Equal(t, 1, tokenCalls, "a usable token must not cost another /token call")
			}
		})
	}

	// A token inside the renewal margin still works for a few minutes. Its renewal failing
	// used to fail the call as well, although the token in hand would have done
	failedRenewals := []struct {
		name   string
		cached string
		err    error
	}{
		{
			name:   "failed renewal of a token that has not expired returns that token",
			cached: dataAccessToken(t, time.Now().Add(2*time.Minute)),
		},
		{
			name:   "failed renewal of an expired token reports the error",
			cached: dataAccessToken(t, time.Now().Add(-time.Hour)),
			err:    ErrorTooManyRequests,
		},
	}

	for _, test := range failedRenewals {
		t.Run(test.name, func(t *testing.T) {
			c := NewThirdParty("test-refresh-token", WithoutRetry()).(*client)
			httpmock.ActivateNonDefault(c.resty.GetClient())
			defer httpmock.DeactivateAndReset()

			// /token hands out the token under test, then fails the renewal, then succeeds
			tokenCalls := 0
			httpmock.RegisterResponder(http.MethodGet, c.resty.BaseURL+"/token",
				func(*http.Request) (*http.Response, error) {
					tokenCalls++
					switch tokenCalls {
					case 1:
						return httpmock.NewJsonResponse(http.StatusOK, map[string]string{"result": test.cached})
					case 2:
						return httpmock.NewStringResponse(http.StatusTooManyRequests, ""), nil
					default:
						return httpmock.NewJsonResponse(http.StatusOK, map[string]string{"result": fresh})
					}
				})

			_, err := c.GetDataAccessToken()
			require.NoError(t, err)

			token, err := c.GetDataAccessToken()
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				assert.Empty(t, token, "an expired token must not be handed out")
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.cached, token, "the token that still works must be handed out")
			}
			assert.Equal(t, 2, tokenCalls, "the token must have been due for renewal")

			// The failure is not remembered: the next call tries the renewal again
			token, err = c.GetDataAccessToken()
			require.NoError(t, err)
			assert.Equal(t, fresh, token)
			assert.Equal(t, 3, tokenCalls)
		})
	}
}

// dataAccessToken builds a data access token that expires at exp.
func dataAccessToken(t *testing.T, exp time.Time) string {
	t.Helper()
	return testToken(t, map[string]any{"tokenType": "ThirdPartyApiDataAccess", "exp": exp.Unix()})
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

			require.Error(t, err)
			require.EqualError(t, err, test.expected.Error())
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

		require.NoError(t, err)
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

		require.Error(t, err)
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

		require.Error(t, err)
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

		require.NoError(t, err)
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

		require.Error(t, err)
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

	require.NoError(t, err)
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
	assert.Empty(t, meteringPoint.SecondConsumerPartyName)
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

		require.NoError(t, err)
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

		require.Error(t, err)
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
		require.NoError(t, err)
		assert.True(t, alive)
	})

	t.Run("returns false on 503 Service Unavailable", func(t *testing.T) {
		httpmock.RegisterResponder("GET", "/isalive", httpmock.NewStringResponder(503, ""))
		alive, err := c.IsAlive()
		require.NoError(t, err)
		assert.False(t, alive)
	})
}
