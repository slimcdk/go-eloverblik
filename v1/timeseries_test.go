package eloverblik

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"io"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetTimeSeries(t *testing.T) {
	// Setup mock client
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token", // Pre-set token to skip auth mock
		resty:       mockResty,
	}

	meteringPointIDs := []string{"571313180100000001"}
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	aggregation := Hour

	t.Run("successfully gets time series data", func(t *testing.T) {
		httpmock.Reset()
		// Mock the API response
		mockResponse := `{
			"result": [
				{
					"MyEnergyData_MarketDocument": {
						"mRID": "2b43a48a-ae74-4059-a72c-325515b6279a",
						"TimeSeries": [
							{
								"mRID": "571313180100000001",
								"measurement_Unit.name": "KWH",
								"Period": [
									{
										"resolution": "PT1H",
										"timeInterval": {
											"start": "2024-01-01T00:00:00Z",
											"end": "2024-01-01T01:00:00Z"
										},
										"Point": [
											{
												"position": "1",
												"out_Quantity.quantity": "0.123",
												"out_Quantity.quality": "A04"
											}
										]
									}
								]
							}
						]
					}
				}
			]
		}`

		url := fmt.Sprintf("/meterdata/gettimeseries/%s/%s/%s", from.In(cph).Format(time.DateOnly), to.In(cph).Format(time.DateOnly), aggregation)
		httpmock.RegisterResponder("POST", url,
			func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, mockResponse)
				resp.Header.Set("Content-Type", "application/json")
				return resp, nil
			},
		)

		// Call the function
		timeSeries, err := c.GetTimeSeries(meteringPointIDs, from, to, aggregation)

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, timeSeries, 1)
		assert.Len(t, timeSeries[0].MyEnergyDataMarketDocument.TimeSeries, 1)
		assert.Equal(t, "571313180100000001", timeSeries[0].MyEnergyDataMarketDocument.TimeSeries[0].MRID)
		assert.Len(t, timeSeries[0].MyEnergyDataMarketDocument.TimeSeries[0].Periods[0].Points, 1)
		assert.Equal(t, 1, timeSeries[0].MyEnergyDataMarketDocument.TimeSeries[0].Periods[0].Points[0].Position)
		assert.Equal(t, 0.123, timeSeries[0].MyEnergyDataMarketDocument.TimeSeries[0].Periods[0].Points[0].OutQuantityQuantity)
	})
}

func TestFlatten(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

	ts := TimeSeries{
		MyEnergyDataMarketDocument: MyEnergyDataMarketDocumentResponse{
			TimeSeries: []TimeSeriesTimeSeriesResponse{
				{
					MeasurementUnitName: "KWH",
					CurveType:           "A01",
					BusinessType:        "A04",
					Periods: []PeriodResponse{
						{
							Resolution: "PT1H",
							TimeInterval: TimeInterval{
								Start: start,
								End:   start.Add(2 * time.Hour),
							},
							Points: []PointResponse{
								{Position: 1, OutQuantityQuantity: 1.1, OutQuantityQuality: "A04"},
								{Position: 2, OutQuantityQuantity: 2.2, OutQuantityQuality: "A03"},
							},
						},
					},
				},
			},
		},
	}

	t.Run("correctly flattens nested time series data", func(t *testing.T) {
		flattened := ts.Flatten()

		assert.Len(t, flattened, 2)

		// Check first point
		assert.Equal(t, start.In(cph), flattened[0].From)
		assert.Equal(t, start.In(cph).Add(1*time.Hour), flattened[0].To)
		assert.Equal(t, 1.1, flattened[0].Measurement)
		assert.Equal(t, "A04", flattened[0].Quality)
		assert.Equal(t, "KWH", flattened[0].Unit)
		assert.Equal(t, Resolution("PT1H"), flattened[0].Resolution)

		// Check second point
		assert.Equal(t, start.In(cph).Add(1*time.Hour), flattened[1].From)
		assert.Equal(t, start.In(cph).Add(2*time.Hour), flattened[1].To)
		assert.Equal(t, 2.2, flattened[1].Measurement)
		assert.Equal(t, "A03", flattened[1].Quality)
	})
}

func TestExportTimeSeries(t *testing.T) {
	mockResty := resty.New()
	httpmock.ActivateNonDefault(mockResty.GetClient())
	defer httpmock.DeactivateAndReset()

	c := &client{
		accessToken: "test-access-token",
		resty:       mockResty,
		apiType:     CustomerApi,
	}

	meteringPointIDs := []string{"571313180100000001"}
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	aggregation := Hour

	t.Run("successfully exports time series data", func(t *testing.T) {
		mockResponse := "header1,header2\nvalue1,value2"
		path := fmt.Sprintf("/meterdata/timeseries/export/%s/%s/%s", from.In(cph).Format(time.DateOnly), to.In(cph).Format(time.DateOnly), aggregation)
		httpmock.RegisterResponder("POST", path, httpmock.NewStringResponder(200, mockResponse))

		body, err := c.ExportTimeSeries(meteringPointIDs, from, to, aggregation)
		assert.NoError(t, err)
		assert.NotNil(t, body)
		defer body.Close()

		content, err := io.ReadAll(body)
		assert.NoError(t, err)
		assert.Equal(t, mockResponse, string(content))
	})
}
