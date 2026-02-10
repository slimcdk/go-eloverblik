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

			var resolution time.Duration
			switch Resolution(period.Resolution) {
			case PT15M:
				resolution = time.Minute * 15
			case PT1H:
				resolution = time.Hour
			case PT1D:
				resolution = time.Hour * 24
			}

			for _, point := range period.Points {

				offset := time.Duration(point.Position-1) * resolution

				var to time.Time
				switch Resolution(period.Resolution) {
				case P1M:
					to = period.TimeInterval.End.In(cph)
				case PT1Y:
					to = period.TimeInterval.End.In(cph)
				default:
					to = period.TimeInterval.Start.In(cph).Add(offset).Add(resolution)
				}

				fts = append(fts, FlatTimeSeriesPoint{
					From:         period.TimeInterval.Start.In(cph).Add(offset),
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
