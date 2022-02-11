package eloverblik

import (
	"time"
)

type TimeseriesQuery struct {
	MeteringPointIDs []string
	DateFrom, DateTo time.Time
	Aggregation      Aggregation
}

type timeSeriesRequest struct {
	meteringPointIDs
}

type timeSeriesResult struct {
	Result []TimeSeriesResponse `json:"result"`
}

type TimeSeriesResponse struct {
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

func (c *client) GetTimeSeries(query TimeseriesQuery) ([]TimeSeriesResponse, error) {
	return nil, nil
	// requestPayload := MeteringPointIDsRequest{MeteringPointID: MeteringPointID{MeteringPointIDs: query.MeteringPointIDs}}
	// dateFrom := query.DateFrom.Format(DateFormat) // TODO: Verify time format
	// dateTo := query.DateTo.Format(DateFormat)     // TODO: Verify time format
	// aggregation, err := verifyAggregation(query.Aggregation)
	// if err != nil {
	// 	return nil, err
	// }

	// // Build URL
	// _url := c.hostUrl
	// _url.Path += fmt.Sprintf("/MeterData/GetTimeSeries/%s/%s/%s", dateFrom, dateTo, aggregation)

	// // Make request and check for errors
	// result := timeSeriesResult{}
	// status, err := c.makeRequest(http.MethodPost, _url, requestPayload, &result)
	// if status != http.StatusOK || err != nil {
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return nil, ErrorClientConnection(status)
	// }

	// return result.Result, err

}
