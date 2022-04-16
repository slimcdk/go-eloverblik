package eloverblik

import (
	"fmt"
	"net/http"
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

type MeterReading struct {
	Result struct {
		MeteringPointID string             `json:"meteringPointId"`
		MeterReadings   []MeterReadingData `json:"readings"`
	} `json:"result"`
	StatusResponse
}

type MeterReadingData struct {
	ReadingDate      time.Time `json:"readingDate"`
	RegistrationDate time.Time `json:"registrationDate"`
	MeterNumber      string    `json:"meterNumber"`
	MeterReading     string    `json:"meterReading"`
	MeasurementUnit  string    `json:"measurementUnit"`
}

type FlatTimeSeriesPoint struct {
	From, To     time.Time
	Measurement  float64
	Quality      string
	Unit         string
	CurveType    string
	BusinessType string
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
	req := c.resty.R()
	req.SetHeader("Accept", "application/json")
	req.SetAuthToken(accessToken)
	req.SetResult(&result)
	req.SetError(&apiErrorMsg)
	req.SetBody(meteringPointIDsToRequestStruct(meteringPointIDs))

	// Execute request
	res, err := req.Post(fmt.Sprintf("/MeterData/GetTimeSeries/%s/%s/%s", from.Format(requestDateFormat), to.Format(requestDateFormat), aggregation))
	if err != nil {
		return nil, err
	}

	// Handle API errors
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, err
}

func (c *client) GetMeterReadings(meteringPointIDs []string, from, to time.Time) ([]MeterReading, error) {

	// Ensure access token is fresh
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	// Response structs
	var resError TimeseriesError
	var result struct {
		Result []MeterReading `json:"result"`
	}

	// Request preflight
	req := c.resty.R()
	req.SetHeader("Accept", "application/json")
	req.SetAuthToken(accessToken)
	req.SetResult(&result)
	req.SetError(&resError)
	req.SetBody(meteringPointIDsToRequestStruct(meteringPointIDs))

	// Execute request
	res, err := req.Post(fmt.Sprintf("/MeterData/GetMeterReadings/%s/%s", from.Format(requestDateFormat), to.Format(requestDateFormat)))
	if err != nil || res.StatusCode() != http.StatusOK {
		return nil, err
	}

	fmt.Println(res.Request.URL)

	return result.Result, err
}

// Flatten simplifies the structure received directly from the API
func (ts *TimeSeries) Flatten() []FlatTimeSeriesPoint {

	fts := make([]FlatTimeSeriesPoint, 0)

	for _, ts := range ts.MyEnergyDataMarketDocument.TimeSeries {
		for _, period := range ts.Periods {
			for _, point := range period.Points {

				offset := time.Hour * time.Duration(point.Position-1)

				fts = append(fts, FlatTimeSeriesPoint{
					From:         period.TimeInterval.Start.Add(offset),
					To:           period.TimeInterval.Start.Add(offset).Add(time.Hour),
					Measurement:  point.OutQuantityQuantity,
					Quality:      point.OutQuantityQuality,
					Unit:         ts.MeasurementUnitName,
					CurveType:    ts.CurveType,
					BusinessType: ts.BusinessType,
				})
			}
		}
	}

	return fts
}
