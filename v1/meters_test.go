package eloverblik

import (
	"net/http"
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
