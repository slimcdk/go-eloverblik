package eloverblik

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

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

// TestGetMeteringPointDetailsFullPayload decodes a payload shaped like a real
// /meteringpoint/getdetails response. The struct used to model only a subset of the
// masterdata the API returns, so fields like assetType, darReference, gridOperatorID,
// meteringPointAlias, mpAddressWashInstructions, occurrence, powerLimitKWDecimal and the
// customer-only balanceSupplier/protectedName fields were silently dropped.
func TestGetMeteringPointDetailsFullPayload(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	httpmock.Reset()
	mockResponse := `{
		"result": [
			{
				"result": {
					"meteringPointId": "571313113162842251",
					"parentMeteringPointId": "",
					"typeOfMP": "E17",
					"energyTimeSeriesMeasureUnit": "KWH",
					"settlementMethod": "D01",
					"meterNumber": "30203518",
					"gridOperatorName": "N1 A/S - 131",
					"gridOperatorID": "5790001089030",
					"gridOperatorID_SchemeAgencyIdentifier": "GLN",
					"meteringGridAreaIdentification": "131",
					"netSettlementGroup": "0",
					"physicalStatusOfMP": "E22",
					"powerLimitKW": "",
					"powerLimitKWDecimal": 25.5,
					"powerLimitA": "",
					"subTypeOfMP": "D01",
					"disconnectionType": "D02",
					"product": "Item8716867000030",
					"consumerCVR": "42703087",
					"dataAccessCVR": "42703087",
					"consumerStartDate": "2025-04-27T22:00:00.000Z",
					"meterReadingOccurrence": "PT1H",
					"meterCounterDigits": "7.0",
					"meterCounterMultiplyFactor": "1.0",
					"meterCounterUnit": "KWH",
					"meterCounterType": "D01",
					"balanceSupplierName": "Test Supplier",
					"balanceSupplierId": "5790000000001",
					"balanceSupplierId_SchemeAgencyIdentifier": "GLN",
					"balanceSupplierStartDate": "2024-01-01T00:00:00.000Z",
					"taxReduction": "False",
					"taxSettlementDate": "",
					"mpRelationType": "",
					"firstConsumerPartyName": "John Sisk & Son ApS",
					"secondConsumerPartyName": "",
					"protectedName": "False",
					"occurrence": "2026-07-12T22:00:00.000Z",
					"meteringPointAlias": "Main meter",
					"assetType": "D01",
					"mpAddressWashInstructions": "D01",
					"darReference": "0a3f5098-ac77-32b8-e044-0003ba298018",
					"streetCode": "0116",
					"streetName": "Blichers Alle",
					"buildingNumber": "1",
					"postcode": "8830",
					"cityName": "Tjele",
					"citySubDivisionName": "Foulum",
					"municipalityCode": "791",
					"contactAddresses": [
						{
							"contactName1": "John Sisk & Son ApS",
							"addressCode": "D01",
							"streetName": "Ørestads Boulevard",
							"buildingNumber": "73",
							"postcode": "2300",
							"cityName": "København S",
							"countryName": "DK",
							"contactPhoneNumber": "00353873349334",
							"contactEmailAddress": "T.Kelly@SISK.ie",
							"attention": "Accounts payable",
							"postBox": "1234",
							"protectedAddress": "False"
						}
					],
					"childMeteringPoints": [
						{
							"parentMeteringPointId": "571313113162842251",
							"meteringPointId": "571313113162842268",
							"typeOfMP": "D01",
							"meterReadingOccurrence": "PT1H",
							"meterNumber": "30203519"
						}
					]
				},
				"success": true,
				"errorCode": 10000,
				"errorText": "NoError",
				"id": "571313113162842251"
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

	details, err := c.GetMeteringPointDetails([]string{"571313113162842251"})

	assert.NoError(t, err)
	if !assert.Len(t, details, 1) {
		return
	}
	detail := details[0].Result

	// Fields the struct used to drop entirely
	assert.Equal(t, "5790001089030", detail.GridOperatorID)
	assert.Equal(t, "GLN", detail.GridOperatorIDSchemeAgencyID)
	assert.Equal(t, "D01", detail.AssetType)
	assert.Equal(t, "D01", detail.MpAddressWashInstructions)
	assert.Equal(t, "0a3f5098-ac77-32b8-e044-0003ba298018", detail.DarReference)
	assert.Equal(t, "Main meter", detail.MeteringPointAlias)
	assert.Equal(t, "False", detail.ProtectedName)
	assert.Equal(t, "5790000000001", detail.BalanceSupplierID)
	assert.Equal(t, "GLN", detail.BalanceSupplierIDSchemeAgencyID)

	// occurrence is a timestamp, not a plain string
	assert.Equal(t, "2026-07-12T22:00:00Z", detail.Occurrence.UTC().Format(time.RFC3339))

	// powerLimitKW stays a string, powerLimitKWDecimal is a number
	assert.Equal(t, "", detail.PowerLimitKW)
	if assert.NotNil(t, detail.PowerLimitKWDecimal) {
		assert.InDelta(t, 25.5, *detail.PowerLimitKWDecimal, 0.0001)
	}

	// Contact address fields the struct used to drop
	if assert.Len(t, detail.ContactAddresses, 1) {
		address := detail.ContactAddresses[0]
		assert.Equal(t, "Accounts payable", address.Attention)
		assert.Equal(t, "1234", address.PostBox)
		assert.Equal(t, "False", address.ProtectedAddress)
	}

	if assert.Len(t, detail.ChildMeteringPoints, 1) {
		assert.Equal(t, "571313113162842268", detail.ChildMeteringPoints[0].MeteringPointID)
		assert.Equal(t, "30203519", detail.ChildMeteringPoints[0].MeterNumber)
	}
}

// TestGetMeteringPointDetailsNullPowerLimit guards the nullable powerLimitKWDecimal. The
// API sends null for most metering points, which must not be confused with a limit of 0.
func TestGetMeteringPointDetailsNullPowerLimit(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     ThirdPartyApi,
	}

	httpmock.Reset()
	httpmock.RegisterResponder("POST", "/meteringpoint/getdetails",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{
				"result": [
					{
						"result": {
							"meteringPointId": "571313113162842251",
							"powerLimitKW": "",
							"powerLimitKWDecimal": null,
							"occurrence": "",
							"consumerStartDate": "",
							"taxSettlementDate": "",
							"balanceSupplierStartDate": ""
						},
						"success": true
					}
				]
			}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		})

	details, err := c.GetMeteringPointDetails([]string{"571313113162842251"})

	assert.NoError(t, err)
	if !assert.Len(t, details, 1) {
		return
	}
	assert.Nil(t, details[0].Result.PowerLimitKWDecimal, "a null power limit must stay unset")
	assert.True(t, details[0].Result.Occurrence.IsZero(), "an empty occurrence must decode to the zero time")
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
