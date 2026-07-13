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

// TestGetChargesPriceID decodes a payload shaped like a real getcharges response. Both
// subscriptions and tariffs carry a priceId on the wire (it is the only stable identifier
// of a charge across renames), and the structs used to drop it.
func TestGetChargesPriceID(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	meteringPointIDs := []string{"571313113162842251"}

	// Trimmed from a live /meteringpoint/getcharges response
	mockResult := `{
		"meteringPointId": "571313113162842251",
		"subscriptions": [
			{
				"price": 28.122861,
				"quantity": 1,
				"priceId": "B2D",
				"name": "Net abo B lav forbrug",
				"description": "Abonnement for timeaflæst måler",
				"owner": "5790001089030",
				"validFromDate": "2020-12-31T23:00:00.000Z",
				"validToDate": null,
				"periodType": "P1M"
			}
		],
		"fees": [
			{
				"price": 10.5,
				"quantity": 2,
				"priceId": "FEE-01",
				"name": "Some Fee",
				"owner": "5790001089030",
				"validFromDate": "2020-12-31T23:00:00.000Z",
				"validToDate": null,
				"periodType": "P1M"
			}
		],
		"tariffs": [
			{
				"prices": [
					{ "position": "1", "price": 0.054689 },
					{ "position": "2", "price": 0.164068 }
				],
				"priceId": "B2D",
				"name": "Nettarif B lav",
				"description": "Tarif for timeaflæst måler",
				"owner": "5790001089030",
				"validFromDate": "2020-12-31T23:00:00.000Z",
				"validToDate": null,
				"periodType": "PT1H"
			},
			{
				"prices": [
					{ "position": "1", "price": 0.008 }
				],
				"priceId": "EA-001",
				"name": "Elafgift",
				"owner": "5790000432752",
				"validFromDate": "2020-07-09T22:00:00.000Z",
				"validToDate": null,
				"periodType": "P1D"
			}
		]
	}`

	t.Run("customer charges carry priceId", func(t *testing.T) {
		httpmock.Reset()
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}

		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, `{"result":[{"result":`+mockResult+`,"success":true}]}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetCustomerCharges(meteringPointIDs)
		assert.NoError(t, err)
		if !assert.Len(t, result, 1) {
			return
		}
		charges := result[0].Result

		if assert.Len(t, charges.Subscriptions, 1) {
			assert.Equal(t, "B2D", charges.Subscriptions[0].PriceID)
			assert.Equal(t, "P1M", charges.Subscriptions[0].PeriodType)
			assert.InDelta(t, 28.122861, charges.Subscriptions[0].Price, 0.000001)
		}
		if assert.Len(t, charges.Fees, 1) {
			assert.Equal(t, "FEE-01", charges.Fees[0].PriceID)
			assert.Equal(t, 2, charges.Fees[0].Quantity)
		}
		if assert.Len(t, charges.Tariffs, 2) {
			assert.Equal(t, "B2D", charges.Tariffs[0].PriceID)
			assert.Equal(t, "EA-001", charges.Tariffs[1].PriceID)
		}
	})

	t.Run("third party charges carry priceId", func(t *testing.T) {
		httpmock.Reset()
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     ThirdPartyApi,
		}

		httpmock.RegisterResponder("POST", "/meteringpoint/getcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, `{"result":[{"result":`+mockResult+`,"success":true}]}`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetThirdPartyCharges(meteringPointIDs)
		assert.NoError(t, err)
		if !assert.Len(t, result, 1) {
			return
		}
		charges := result[0].Result

		if assert.Len(t, charges.Subscriptions, 1) {
			assert.Equal(t, "B2D", charges.Subscriptions[0].PriceID)
		}
		if assert.Len(t, charges.Tariffs, 2) {
			// The tariff resolution is reported as PT1H, not the P1H the spec documents
			assert.Equal(t, "PT1H", charges.Tariffs[0].PeriodType)
			assert.Equal(t, "B2D", charges.Tariffs[0].PriceID)
			if assert.Len(t, charges.Tariffs[0].Prices, 2) {
				assert.Equal(t, "2", charges.Tariffs[0].Prices[1].Position)
				assert.InDelta(t, 0.164068, charges.Tariffs[0].Prices[1].Price, 0.000001)
			}
			assert.Equal(t, "EA-001", charges.Tariffs[1].PriceID)
			// validToDate is null on open-ended charges
			assert.True(t, charges.Tariffs[1].ValidToDate.IsZero())
		}
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
