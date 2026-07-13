package eloverblik

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingLogger records what resty logs, so a test can prove resty had nothing to
// complain about. resty logs a warning of its own when it cannot unmarshal a response body,
// and swallows the body, which is exactly the failure this guards against.
type capturingLogger struct {
	warnings []string
	errs     []string
}

func (l *capturingLogger) Errorf(format string, v ...any) {
	l.errs = append(l.errs, fmt.Sprintf(format, v...))
}
func (l *capturingLogger) Warnf(format string, v ...any) {
	l.warnings = append(l.warnings, fmt.Sprintf(format, v...))
}
func (l *capturingLogger) Debugf(string, ...any) {}

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

// TestGetChargeLinksWithChargesNotDeployed covers the answer the live API actually gives
// this endpoint today: a 404 with an RFC 7807 problem document rather than the usual
// "[code] message" string. It used to make resty warn "Cannot unmarshal response body" and
// drop the body, leaving the caller with a bare "could't connect to eloverblik: 404".
func TestGetChargeLinksWithChargesNotDeployed(t *testing.T) {
	logger := &capturingLogger{}

	mockResty := resty.New().SetLogger(logger)
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     ThirdPartyApi,
	}

	httpmock.RegisterResponder("POST", "/meteringpoint/getchargelinkswithcharges",
		func(_ *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(404, problemDocument404)
			resp.Header.Set("Content-Type", "application/problem+json; charset=utf-8")
			return resp, nil
		})

	result, err := c.GetChargeLinksWithCharges(
		[]string{"571313180100000001"},
		time.Date(2024, 1, 1, 0, 0, 0, 0, cph),
		time.Date(2024, 2, 1, 0, 0, 0, 0, cph),
	)
	assert.Nil(t, result)
	require.Error(t, err)

	// The problem document reaches the caller whole, trace ID included: it is what Energinet
	// support asks for when an endpoint they declare answers 404
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "Not Found", apiErr.Title)
	assert.Equal(t, "00-9c485a3a3ed458eab22cab724111db63-ed7aa1e057161e52-01", apiErr.TraceID)

	// resty had nothing to complain about: the body was read, not swallowed
	assert.Empty(t, logger.warnings)
	assert.Empty(t, logger.errs)
}
