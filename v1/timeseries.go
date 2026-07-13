package eloverblik

import (
	"fmt"
	"io"
	"time"
)

type TimeSeries struct {
	MyEnergyDataMarketDocument MyEnergyDataMarketDocumentResponse `json:"MyEnergyData_MarketDocument"`
	StatusResponse
}

type MyEnergyDataMarketDocumentResponse struct {
	MRID                        string                         `json:"mRID"`
	CreatedDateTime             time.Time                      `json:"createdDateTime"`
	SenderMarketParticipantName string                         `json:"sender_MarketParticipant.name"`
	SenderMarketParticipantMRID MRIDResponse                   `json:"sender_MarketParticipant.mRID"`
	PeriodTimeInterval          TimeInterval                   `json:"period.timeInterval"`
	TimeSeries                  []TimeSeriesTimeSeriesResponse `json:"TimeSeries"`
}

type MRIDResponse struct {
	CodingScheme string `json:"codingScheme"`
	Name         string `json:"name"`
}

type TimeInterval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type TimeSeriesTimeSeriesResponse struct {
	MRID                  string                        `json:"mRID"`
	BusinessType          string                        `json:"businessType"`
	CurveType             string                        `json:"curveType"`
	MeasurementUnitName   string                        `json:"measurement_Unit.name"`
	MarketEvaluationPoint MarketEvaluationPointResponse `json:"MarketEvaluationPoint"`
	Periods               []PeriodResponse              `json:"Period"`
}

type MarketEvaluationPointResponse struct {
	MRID MRIDResponse `json:"mRID"`
}

type PeriodResponse struct {
	Resolution   string          `json:"resolution"`
	TimeInterval TimeInterval    `json:"timeInterval"`
	Points       []PointResponse `json:"point"`
}

type PointResponse struct {
	Position            int     `json:"position,string"`
	OutQuantityQuantity float64 `json:"out_Quantity.quantity,string"`
	OutQuantityQuality  string  `json:"out_Quantity.quality"`
}

type FlatTimeSeriesPoint struct {
	From         time.Time  `json:"from"`
	To           time.Time  `json:"to"`
	Measurement  float64    `json:"measurement"`
	Quality      string     `json:"quality"`
	Unit         string     `json:"unit"`
	CurveType    string     `json:"curvetype"`
	BusinessType string     `json:"businesstype"`
	Resolution   Resolution `json:"resolution"`
}

// GetTimeSeries fetches meter accumulated meter readings within the given aggregation
func (c *client) GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error) {

	// Ensure access token is fresh
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	// Response structs
	var apiErrorMsg string
	var result struct {
		Result []TimeSeries `json:"result"`
	}

	// Request preflight
	req := c.resty.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(accessToken).
		SetResult(&result).
		SetError(&apiErrorMsg).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs))

	// Both Customer and ThirdParty APIs use the same lowercase path
	path := fmt.Sprintf("/meterdata/gettimeseries/%s/%s/%s", from.In(cph).Format(time.DateOnly), to.In(cph).Format(time.DateOnly), aggregation)

	// Execute request
	res, err := req.Post(path)
	if err != nil {
		return nil, err
	}

	// Handle API errors
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, err
}

// Flatten simplifies the structure received directly from the API
func (ts *TimeSeries) Flatten() []FlatTimeSeriesPoint {

	fts := make([]FlatTimeSeriesPoint, 0)

	for _, ts := range ts.MyEnergyDataMarketDocument.TimeSeries {
		for _, period := range ts.Periods {
			for _, point := range period.Points {

				from, to := pointInterval(Resolution(period.Resolution), period.TimeInterval, point.Position, len(period.Points))

				fts = append(fts, FlatTimeSeriesPoint{
					From:         from,
					To:           to,
					Measurement:  point.OutQuantityQuantity,
					Quality:      point.OutQuantityQuality,
					Unit:         ts.MeasurementUnitName,
					CurveType:    ts.CurveType,
					BusinessType: ts.BusinessType,
					Resolution:   Resolution(period.Resolution),
				})
			}
		}
	}

	return fts
}

// pointInterval returns the half-open [from, to) interval covered by the point at the
// given 1-based position within a period, in Copenhagen local time.
//
// The API sends one period per day for Day, per month for Month and per year for Year,
// each holding a single point, and one period per day holding 24 points for Hour. A
// period with a single point therefore takes its interval verbatim, which also keeps a
// partial period correct — a Year period may cover, say, only April to December.
//
// Periods with several points step by calendar unit rather than by a fixed duration, so
// that a day containing a daylight saving transition (23 or 25 hours) and months of
// unequal length still yield the correct boundaries.
func pointInterval(resolution Resolution, interval TimeInterval, position, points int) (time.Time, time.Time) {

	start := interval.Start.In(cph)
	end := interval.End.In(cph)
	index := position - 1

	// The API already states the exact interval this point covers.
	if points == 1 {
		return start, end
	}

	switch resolution {
	case PT15M:
		from := start.Add(time.Duration(index) * 15 * time.Minute)
		return from, from.Add(15 * time.Minute)

	case PT1H:
		from := start.Add(time.Duration(index) * time.Hour)
		return from, from.Add(time.Hour)

	// The live API sends PT1D and PT1Y; the spec documents P1D and P1Y. Accept both.
	case PT1D, P1D:
		from := start.AddDate(0, 0, index)
		return from, from.AddDate(0, 0, 1)

	case P1M:
		from := start.AddDate(0, index, 0)
		return from, from.AddDate(0, 1, 0)

	case PT1Y, P1Y:
		from := start.AddDate(index, 0, 0)
		return from, from.AddDate(1, 0, 0)

	case PXD:
		// The period covers a variable number of days for profiled energy quantities.
		// Its width is not implied by the resolution, so spread the points evenly.
		if points > 0 {
			width := end.Sub(start) / time.Duration(points)
			from := start.Add(time.Duration(index) * width)
			return from, from.Add(width)
		}
	}

	// Unknown resolution: attribute the whole period to the point rather than
	// silently collapsing it to a zero-width interval.
	return start, end
}

func (c *client) ExportTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) (io.ReadCloser, error) {
	if c.apiType != CustomerApi {
		return nil, fmt.Errorf("ExportTimeSeries is only available for Customer API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/meterdata/timeseries/export/%s/%s/%s", from.In(cph).Format(time.DateOnly), to.In(cph).Format(time.DateOnly), aggregation)

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetDoNotParseResponse(true). // We want the raw response body
		Post(path)

	if err != nil || !res.IsSuccess() {
		return nil, fmt.Errorf("failed to export time series, status: %s, err: %v", res.Status(), err)
	}

	return res.RawBody(), nil
}
