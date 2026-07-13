package eloverblik

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetMeteringPoints(t *testing.T) {
	// Setup mock client
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token", // Pre-set token to skip auth mock
		resty:       mockResty,
	}

	t.Run("successfully gets metering points", func(t *testing.T) {
		httpmock.Reset()
		// Mock the API response
		mockResponse := `{
			"result": [
				{
					"meteringPointId": "571313180100000001",
					"typeOfMP": "E17",
					"streetName": "Testvej",
					"buildingNumber": "1",
					"postcode": "8000",
					"cityName": "Aarhus C",
					"consumerStartDate": "",
					"hasRelation": true
				}
			]
		}`
		httpmock.RegisterResponder("GET", "/MeteringPoints/MeteringPoints",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			},
		)

		// Call the function
		meteringPoints, err := c.GetMeteringPoints(true)

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, meteringPoints, 1)
		assert.Equal(t, "571313180100000001", meteringPoints[0].MeteringPointID)
		assert.Equal(t, "Testvej", meteringPoints[0].StreetName)
		assert.True(t, meteringPoints[0].HasRelation)
	})
}

// TestGetMeteringPointsIncludeAll guards the includeAll query parameter. It used to be
// passed as a path parameter to a path with no placeholder, so resty dropped it and the
// API silently applied its default of false.
func TestGetMeteringPointsIncludeAll(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
	}

	for _, includeAll := range []bool{true, false} {
		t.Run(fmt.Sprintf("includeAll=%t reaches the wire", includeAll), func(t *testing.T) {
			httpmock.Reset()

			var query string
			httpmock.RegisterResponder("GET", "/MeteringPoints/MeteringPoints",
				func(req *http.Request) (*http.Response, error) {
					query = req.URL.Query().Get("includeAll")
					resp := httpmock.NewStringResponse(200, `{"result": []}`)
					resp.Header.Set("Content-Type", "application/json")
					return resp, nil
				},
			)

			_, err := c.GetMeteringPoints(includeAll)

			assert.NoError(t, err)
			assert.Equal(t, strconv.FormatBool(includeAll), query, "includeAll must be sent as a query parameter")
		})
	}
}

// TestGetMeteringPointsFailure guards the metering point list. Every non-200 used to be
// returned as (nil, nil), so an expired token or a rate limit looked exactly like an
// account without metering points.
func TestGetMeteringPointsFailure(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	tests := []struct {
		name        string
		status      int
		body        string
		contentType string
		expected    error
	}{
		{
			name:        "expired access token",
			status:      http.StatusUnauthorized,
			body:        `"[20012] Unauthorized"`,
			contentType: "application/json",
			expected:    ErrorUnauthorized,
		},
		{
			// includeAll=true asks for CPR data, which requires consent
			name:        "missing CPR consent",
			status:      http.StatusForbidden,
			body:        `"[10007] Missing consent for CPR lookup"`,
			contentType: "application/json",
			expected:    ErrorNoCprConsent,
		},
		{
			name:     "rate limited without an API error message",
			status:   http.StatusTooManyRequests,
			expected: ErrorTooManyRequests,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpmock.Reset()
			httpmock.RegisterResponder("GET", "/MeteringPoints/MeteringPoints",
				func(req *http.Request) (*http.Response, error) {
					resp := httpmock.NewStringResponse(test.status, test.body)
					if test.contentType != "" {
						resp.Header.Set("Content-Type", test.contentType)
					}
					return resp, nil
				})

			meteringPoints, err := c.GetMeteringPoints(true)

			assert.Error(t, err)
			assert.EqualError(t, err, test.expected.Error())
			assert.Nil(t, meteringPoints)
		})
	}
}

func TestGetMeteringPointDetails(t *testing.T) {
	// Setup mock client
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token", // Pre-set token to skip auth mock
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	meteringPointIDs := []string{"571313180100000001"}

	t.Run("successfully gets metering point details", func(t *testing.T) {
		httpmock.Reset()
		// Mock the API response
		mockResponse := `{
			"result": [
				{
					"result": {
						"meteringPointId": "571313180100000001",
						"streetName": "Testvej",
						"gridOperatorName": "Test Grid",
						"consumerStartDate": "",
						"balanceSupplierStartDate": "",
						"taxSettlementDate": ""
					},
					"success": true
				}
			]
		}`
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getdetails",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			},
		)

		// Call the function
		details, err := c.GetMeteringPointDetails(meteringPointIDs)

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, details, 1)
		assert.True(t, details[0].Success)
		assert.Equal(t, "571313180100000001", details[0].Result.MeteringPointID)
		assert.Equal(t, "Test Grid", details[0].Result.GridOperatorName)
	})

	t.Run("handles API error response", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getdetails",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(401, `"[30006] Access denied"`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		_, err := c.GetMeteringPointDetails(meteringPointIDs)

		assert.Error(t, err)
		assert.Equal(t, ErrorAccessToMeteringPointDenied, err)
	})
}
func TestExportMasterdata(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	meteringPointIDs := []string{"571313180100000001"}

	t.Run("successfully exports masterdata as CSV", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}

		mockCSV := "meteringPointId,streetName,postcode\n571313180100000001,Testvej,8000"
		httpmock.RegisterResponder("POST", "/meteringpoints/masterdata/export",
			httpmock.NewStringResponder(200, mockCSV))

		stream, err := c.ExportMasterdata(meteringPointIDs)
		assert.NoError(t, err)
		assert.NotNil(t, stream)
		defer stream.Close()

		// Read and verify CSV content
		buf := make([]byte, len(mockCSV))
		n, _ := stream.Read(buf)
		assert.Equal(t, mockCSV, string(buf[:n]))
	})

	t.Run("fails for non-customer API", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     ThirdPartyApi,
		}

		_, err := c.ExportMasterdata(meteringPointIDs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available for Customer API")
	})
}
