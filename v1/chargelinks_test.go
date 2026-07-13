package eloverblik

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetChargeLinksWithCharges(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	meteringPointIDs := []string{"571313180100000001"}
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, cph)
	to := time.Date(2024, 2, 1, 0, 0, 0, 0, cph)

	t.Run("successfully gets charge links with charges from the customer API", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}
		httpmock.Reset()

		mockResponse := `{
			"result": {
				"results": [
					{
						"meteringPointId": "571313180100000001",
						"error": null,
						"chargeLinks": [
							{
								"meteringPointId": "571313180100000001",
								"chargeIdentifier": {
									"code": "DT_C_01",
									"owner": "5790000705689",
									"type": "D03"
								},
								"chargeLinkPeriods": [
									{
										"factor": 1,
										"from": "2024-01-01T00:00:00Z",
										"to": null
									}
								]
							}
						]
					}
				],
				"chargeInformations": [
					{
						"chargeIdentifier": {
							"code": "DT_C_01",
							"owner": "5790000705689",
							"type": "D03"
						},
						"taxIndicator": true,
						"resolution": "PT1H",
						"pricingCategory": "Flex",
						"chargeInformationPeriods": [
							{
								"name": "Nettarif C time",
								"description": "Nettarif for C-kunder",
								"transparentInvoicing": true,
								"from": "2024-01-01T00:00:00Z",
								"to": null,
								"vatClassification": "D02"
							}
						],
						"chargeSeriesPoints": [
							{
								"from": "2024-01-01T00:00:00Z",
								"to": "2024-01-01T01:00:00Z",
								"price": 0.2331
							},
							{
								"from": "2024-01-01T01:00:00Z",
								"to": "2024-01-01T02:00:00Z",
								"price": 0.6993
							}
						]
					}
				]
			}
		}`

		var requestBody []byte
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getchargelinkswithcharges",
			func(req *http.Request) (*http.Response, error) {
				requestBody, _ = io.ReadAll(req.Body)
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetChargeLinksWithCharges(meteringPointIDs, from, to)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// The request envelope is a flat query array with a date interval per metering point,
		// not the {"meteringPoints":{"meteringPoint":[...]}} envelope of the other endpoints
		assert.JSONEq(t, `{"query":[{"meteringPointId":"571313180100000001","from":"2024-01-01T00:00:00+01:00","to":"2024-02-01T00:00:00+01:00"}]}`, string(requestBody))

		var sent ChargeLinksWithChargesRequest
		assert.NoError(t, json.Unmarshal(requestBody, &sent))
		assert.Len(t, sent.Query, 1)
		assert.Equal(t, "571313180100000001", sent.Query[0].MeteringPointID)
		assert.True(t, sent.Query[0].From.Equal(from))
		assert.True(t, sent.Query[0].To.Equal(to))

		// Charge links
		assert.Len(t, result.Results, 1)
		assert.Equal(t, "571313180100000001", result.Results[0].MeteringPointID)
		assert.Empty(t, result.Results[0].Error)
		assert.Len(t, result.Results[0].ChargeLinks, 1)

		link := result.Results[0].ChargeLinks[0]
		assert.Equal(t, ChargeIdentifier{Code: "DT_C_01", Owner: "5790000705689", Type: "D03"}, link.ChargeIdentifier)
		assert.Len(t, link.ChargeLinkPeriods, 1)
		assert.Equal(t, 1, link.ChargeLinkPeriods[0].Factor)
		assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), link.ChargeLinkPeriods[0].From.Time)
		assert.True(t, link.ChargeLinkPeriods[0].To.IsZero(), "an open ended link period has no to date")

		// Charge information, including the dated price series
		assert.Len(t, result.ChargeInformations, 1)
		info := result.ChargeInformations[0]
		assert.Equal(t, link.ChargeIdentifier, info.ChargeIdentifier)
		assert.True(t, info.TaxIndicator)
		assert.Equal(t, "PT1H", info.Resolution)
		assert.Equal(t, "Flex", info.PricingCategory)

		assert.Len(t, info.ChargeInformationPeriods, 1)
		assert.Equal(t, "Nettarif C time", info.ChargeInformationPeriods[0].Name)
		assert.True(t, info.ChargeInformationPeriods[0].TransparentInvoicing)
		assert.Equal(t, "D02", info.ChargeInformationPeriods[0].VATClassification)

		assert.Len(t, info.ChargeSeriesPoints, 2)
		assert.Equal(t, ChargeSeriesPoint{
			From:  FlexibleTime{Time: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)},
			To:    FlexibleTime{Time: time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC)},
			Price: 0.6993,
		}, info.ChargeSeriesPoints[1])
	})

	t.Run("successfully gets charge links with charges from the third party API", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     ThirdPartyApi,
		}
		httpmock.Reset()

		mockResponse := `{
			"result": {
				"results": [
					{
						"meteringPointId": "571313180100000001",
						"chargeLinks": []
					}
				],
				"chargeInformations": []
			}
		}`
		httpmock.RegisterResponder("POST", "/meteringpoint/getchargelinkswithcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetChargeLinksWithCharges(meteringPointIDs, from, to)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 1)
		assert.Empty(t, result.Results[0].ChargeLinks)

		// The customer path must not be called by a third party client
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})

	t.Run("reports a per metering point error", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}
		httpmock.Reset()

		mockResponse := `{
			"result": {
				"results": [
					{
						"meteringPointId": "571313180100000002",
						"error": "No access to metering point",
						"chargeLinks": null
					}
				],
				"chargeInformations": null
			}
		}`
		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getchargelinkswithcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetChargeLinksWithCharges([]string{"571313180100000002"}, from, to)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 1)
		assert.Equal(t, "No access to metering point", result.Results[0].Error)
		assert.Empty(t, result.Results[0].ChargeLinks)
		assert.Empty(t, result.ChargeInformations)
	})

	t.Run("returns an API error", func(t *testing.T) {
		c := &client{
			accessToken: "test-access-token",
			resty:       mockResty,
			apiType:     CustomerApi,
		}
		httpmock.Reset()

		httpmock.RegisterResponder("POST", "/meteringpoints/meteringpoint/getchargelinkswithcharges",
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(400, `"[30004] Invalid date format in request"`)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			})

		result, err := c.GetChargeLinksWithCharges(meteringPointIDs, from, to)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrorInvalidDateFormat)
	})
}
