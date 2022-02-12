package eloverblik

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TimeSeries struct {
	MyEnergyDataMarketDocument MyEnergyDataMarketDocumentResponse `json:"MyEnergyData_MarketDocument"`
	StatusResponse
}

type MyEnergyDataMarketDocumentResponse struct {
	MRID                          string                         `json:"mRID"`
	CreatedDateTime               string                         `json:"createdDateTime"`
	Sender_MarketParticipant_name string                         `json:"sender_MarketParticipant.name"`
	Sender_MarketParticipant_mRID MRIDResponse                   `json:"sender_MarketParticipant.mRID"`
	Period_TimeInterval           TimeInterval                   `json:"period.timeInterval"`
	TimeSeries                    []TimeSeriesTimeSeriesResponse `json:"TimeSeries"`
}

type MRIDResponse struct {
	CodingScheme string `json:"codingScheme"`
	Name         string `json:"name"`
}

type TimeInterval struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type TimeSeriesTimeSeriesResponse struct {
	MRID                  string                        `json:"mRID"`
	BusinessType          string                        `json:"businessType"`
	CurveType             string                        `json:"curveType"`
	Measurement_Unit_name string                        `json:"measurement_Unit.name"`
	MarketEvaluationPoint MarketEvaluationPointResponse `json:"MarketEvaluationPoint"`
	Period                []PeriodResponse              `json:"Period"`
}

type MarketEvaluationPointResponse struct {
	MRID MRIDResponse `json:"mRID"`
}

type PeriodResponse struct {
	Resolution   string          `json:"resolution"`
	TimeInterval TimeInterval    `json:"timeInterval"`
	Point        []PointResponse `json:"point"`
}

type PointResponse struct {
	Position              string `json:"position"`
	Out_Quantity_quantity string `json:"out_Quantity.quantity"`
	Out_Quantity_quality  string `json:"out_Quantity.quality"`
}

type MeterReading struct {
	Result struct {
		MeteringPointID string             `json:"meteringPointId"`
		MeterReadings   []MeterReadingData `json:"readings"`
	} `json:"result"`
	StatusResponse
}

type MeterReadingData struct {
	ReadingDate      string `json:"readingDate"`
	RegistrationDate string `json:"registrationDate"`
	MeterNumber      string `json:"meterNumber"`
	MeterReading     string `json:"meterReading"`
	MeasurementUnit  string `json:"measurementUnit"`
}

func (c *client) GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error) {

	dateFrom, dateTo := from.Format(DateFormat), to.Format(DateFormat)
	if !validAggregation(aggregation) {
		return nil, ErrorAggrationNotValid
	}

	// Build URL
	_url := c.hostUrl
	_url.Path += fmt.Sprintf("/MeterData/GetTimeSeries/%s/%s/%s", dateFrom, dateTo, aggregation)

	// Construct body payload
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(meteringPointIDsToRequestStruct(meteringPointIDs))
	if err != nil {
		return nil, err
	}

	// Construct request and set authorization
	req, err := http.NewRequest(http.MethodPost, _url.String(), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Make request and parse response
	res, err := c.client.Do(req)
	if isRetryableError(res.StatusCode, err) {
		return c.GetTimeSeries(meteringPointIDs, from, to, aggregation)
	}

	// Decode response result
	var result struct {
		Result []TimeSeries `json:"result"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Result, err
}

func (c *client) GetMeterReadings(meteringPointIDs []string, from, to time.Time) ([]MeterReading, error) {

	dateFrom, dateTo := from.Format(DateFormat), to.Format(DateFormat)

	// Build URL
	_url := c.hostUrl
	_url.Path += fmt.Sprintf("/MeterData/GetMeterReadings/%s/%s", dateFrom, dateTo)

	// Construct body payload
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(meteringPointIDsToRequestStruct(meteringPointIDs))
	if err != nil {
		return nil, err
	}

	// Construct request and set authorization
	req, err := http.NewRequest(http.MethodPost, _url.String(), &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Make request and parse response
	res, err := c.client.Do(req)
	if isRetryableError(res.StatusCode, err) {
		return c.GetMeterReadings(meteringPointIDs, from, to)
	}

	// Decode response result
	var result struct {
		Result []MeterReading `json:"result"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Result, err

}
