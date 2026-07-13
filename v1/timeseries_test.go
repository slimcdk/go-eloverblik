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

// TestFlattenResolutions covers every resolution. The live API sends PT1D and PT1Y where
// the OpenAPI description documents P1D and P1Y, so both spellings must flatten the same
// way; an unknown resolution must not collapse a point to a zero-width interval.
func TestFlattenResolutions(t *testing.T) {
	// The API expresses period boundaries in UTC. 2023-12-31T23:00Z is 2024-01-01 00:00
	// in Copenhagen, and 2024-03-30T23:00Z is 2024-03-31 00:00 — a 23 hour day, because
	// daylight saving time starts that night.
	newYear := time.Date(2023, 12, 31, 23, 0, 0, 0, time.UTC)
	dstDay := time.Date(2024, 3, 30, 23, 0, 0, 0, time.UTC)

	local := func(year int, month time.Month, day, hour int) time.Time {
		return time.Date(year, month, day, hour, 0, 0, 0, cph)
	}

	tests := []struct {
		name       string
		resolution string
		interval   TimeInterval
		points     int
		wantFrom   []time.Time
		wantTo     []time.Time
	}{
		{
			name:       "quarter of an hour",
			resolution: "PT15M",
			interval:   TimeInterval{Start: newYear, End: newYear.Add(30 * time.Minute)},
			points:     2,
			wantFrom:   []time.Time{local(2024, 1, 1, 0), local(2024, 1, 1, 0).Add(15 * time.Minute)},
			wantTo:     []time.Time{local(2024, 1, 1, 0).Add(15 * time.Minute), local(2024, 1, 1, 0).Add(30 * time.Minute)},
		},
		{
			// PT1D is what the live API sends for Day.
			name:       "day",
			resolution: "PT1D",
			interval:   TimeInterval{Start: newYear, End: newYear.Add(48 * time.Hour)},
			points:     2,
			wantFrom:   []time.Time{local(2024, 1, 1, 0), local(2024, 1, 2, 0)},
			wantTo:     []time.Time{local(2024, 1, 2, 0), local(2024, 1, 3, 0)},
		},
		{
			// P1D is the spelling the OpenAPI description documents.
			name:       "day, spec spelling",
			resolution: "P1D",
			interval:   TimeInterval{Start: newYear, End: newYear.Add(48 * time.Hour)},
			points:     2,
			wantFrom:   []time.Time{local(2024, 1, 1, 0), local(2024, 1, 2, 0)},
			wantTo:     []time.Time{local(2024, 1, 2, 0), local(2024, 1, 3, 0)},
		},
		{
			name:       "day across the daylight saving transition",
			resolution: "PT1D",
			interval:   TimeInterval{Start: dstDay, End: dstDay.Add(47 * time.Hour)},
			points:     2,
			// The second day starts 23 hours after the first, not 24.
			wantFrom: []time.Time{local(2024, 3, 31, 0), local(2024, 4, 1, 0)},
			wantTo:   []time.Time{local(2024, 4, 1, 0), local(2024, 4, 2, 0)},
		},
		{
			name:       "month",
			resolution: "P1M",
			interval:   TimeInterval{Start: newYear, End: time.Date(2024, 2, 29, 23, 0, 0, 0, time.UTC)},
			points:     2,
			wantFrom:   []time.Time{local(2024, 1, 1, 0), local(2024, 2, 1, 0)},
			wantTo:     []time.Time{local(2024, 2, 1, 0), local(2024, 3, 1, 0)},
		},
		{
			name:       "year",
			resolution: "PT1Y",
			interval:   TimeInterval{Start: newYear, End: time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC)},
			points:     2,
			wantFrom:   []time.Time{local(2024, 1, 1, 0), local(2025, 1, 1, 0)},
			wantTo:     []time.Time{local(2025, 1, 1, 0), local(2026, 1, 1, 0)},
		},
		{
			// The API sends one point per period for Day, Month and Year, and that period
			// can be partial: a Year period may start in April and end on 31 December.
			// A single point must take the interval the API states, not a calendar year.
			name:       "single point takes the period interval verbatim",
			resolution: "PT1Y",
			interval: TimeInterval{
				Start: time.Date(2025, 4, 26, 22, 0, 0, 0, time.UTC),
				End:   time.Date(2025, 12, 30, 23, 0, 0, 0, time.UTC),
			},
			points:   1,
			wantFrom: []time.Time{local(2025, 4, 27, 0)},
			wantTo:   []time.Time{local(2025, 12, 31, 0)},
		},
		{
			name:       "variable number of days is spread evenly",
			resolution: "PXD",
			interval:   TimeInterval{Start: newYear, End: newYear.Add(96 * time.Hour)},
			points:     2,
			wantFrom:   []time.Time{local(2024, 1, 1, 0), local(2024, 1, 1, 0).Add(48 * time.Hour)},
			wantTo:     []time.Time{local(2024, 1, 1, 0).Add(48 * time.Hour), local(2024, 1, 1, 0).Add(96 * time.Hour)},
		},
		{
			name:       "unknown resolution falls back to the whole period",
			resolution: "P1W",
			interval:   TimeInterval{Start: newYear, End: newYear.Add(168 * time.Hour)},
			points:     1,
			wantFrom:   []time.Time{local(2024, 1, 1, 0)},
			wantTo:     []time.Time{local(2024, 1, 1, 0).Add(168 * time.Hour)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			points := make([]PointResponse, 0, test.points)
			for i := 1; i <= test.points; i++ {
				points = append(points, PointResponse{Position: i, OutQuantityQuantity: float64(i)})
			}

			ts := TimeSeries{
				MyEnergyDataMarketDocument: MyEnergyDataMarketDocumentResponse{
					TimeSeries: []TimeSeriesTimeSeriesResponse{{
						Periods: []PeriodResponse{{
							Resolution:   test.resolution,
							TimeInterval: test.interval,
							Points:       points,
						}},
					}},
				},
			}

			flattened := ts.Flatten()
			assert.Len(t, flattened, test.points)

			for i, point := range flattened {
				assert.True(t, point.From.Equal(test.wantFrom[i]), "point %d from: got %s, want %s", i+1, point.From, test.wantFrom[i])
				assert.True(t, point.To.Equal(test.wantTo[i]), "point %d to: got %s, want %s", i+1, point.To, test.wantTo[i])
				assert.True(t, point.To.After(point.From), "point %d must cover a non-zero interval", i+1)
			}
		})
	}
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
