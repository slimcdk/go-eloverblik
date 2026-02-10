package eloverblik

import (
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetCharges(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	meteringPointIDs := []string{"571313180100000001"}

	t.Run("successfully gets customer charges", func(t *testing.T) {
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
						"meteringPointId": "571313180100000001",
						"fees": [
							{
								"name": "Some Fee",
								"description": "Test fee",
								"owner": "Test Owner",
								"validFromDate": "2024-01-01T00:00:00Z",
								"validToDate": "2024-12-31T23:59:59Z",
								"periodType": "P1M",
								"price": 25.0,
								"quantity": 1
							}
						]
					}
				}
			]
		}`
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetCustomerCharges(meteringPointIDs)
		assert.NoError(t, err)

		assert.IsType(t, []CustomerChargeResponse{}, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "571313180100000001", result[0].Result.MeteringPointID)
		assert.Len(t, result[0].Result.Fees, 1)
		assert.Equal(t, "Some Fee", result[0].Result.Fees[0].Name)
	})

	t.Run("successfully gets third party charges", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     ThirdPartyApi,
		}
		httpmock.Reset()

		mockResponse := `{
			"result": [
				{
					"result": {
						"meteringPointId": "571313180100000001",
						"tariffs": [
							{
								"name": "Some Tariff",
								"description": "Test tariff",
								"owner": "Test Owner",
								"validFromDate": "2024-01-01T00:00:00Z",
								"validToDate": "2024-12-31T23:59:59Z",
								"periodType": "P1M",
								"prices": []
							}
						]
					}
				}
			]
		}`
		httpmock.RegisterResponder("POST", "/meteringpoint/getcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetThirdPartyCharges(meteringPointIDs)
		assert.NoError(t, err)

		assert.IsType(t, []ThirdPartyChargeResponse{}, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "571313180100000001", result[0].Result.MeteringPointID)
		assert.Len(t, result[0].Result.Tariffs, 1)
		assert.Equal(t, "Some Tariff", result[0].Result.Tariffs[0].Name)
	})
}

func TestExportCharges(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	meteringPointIDs := []string{"571313180100000001"}

	t.Run("successfully exports charges as CSV", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}

		mockCSV := "meteringPointId,name,price\n571313180100000001,Nettarif C,0.12"
		httpmock.RegisterResponder("POST", "/meteringpoints/charges/export",
			httpmock.NewStringResponder(200, mockCSV))

		stream, err := c.ExportCharges(meteringPointIDs)
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

		_, err := c.ExportCharges(meteringPointIDs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only available for Customer API")
	})
}
