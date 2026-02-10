package eloverblik

import (
	"fmt"
	"io"
)

type CustomerChargeResponse struct {
	Result CustomerCharges `json:"result,omitempty"`
	StatusResponse
}

type ThirdPartyChargeResponse struct {
	Result ThirdPartyCharges `json:"result,omitempty"`
	StatusResponse
}

type CustomerCharges struct {
	MeteringPointID string         `json:"meteringPointId"`
	Subscriptions   []Charge       `json:"subscriptions"`
	Fees            []Charge       `json:"fees"`
	Tariffs         []TariffCharge `json:"tariffs"`
}

type ThirdPartyCharges struct {
	MeteringPointID string         `json:"meteringPointId"`
	Subscriptions   []Charge       `json:"subscriptions"`
	Tariffs         []TariffCharge `json:"tariffs"`
}

type Charge struct {
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Owner         string       `json:"owner"`
	ValidFromDate FlexibleTime `json:"validFromDate"`
	ValidToDate   FlexibleTime `json:"validToDate"`
	PeriodType    string       `json:"periodType"`
	Price         float64      `json:"price"`
	Quantity      int          `json:"quantity"`
}

type TariffCharge struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Owner         string        `json:"owner"`
	ValidFromDate FlexibleTime  `json:"validFromDate"`
	ValidToDate   FlexibleTime  `json:"validToDate"`
	PeriodType    string        `json:"periodType"`
	Prices        []TariffPrice `json:"prices"`
}

type TariffPrice struct {
	Position string  `json:"position"`
	Price    float64 `json:"price"`
}

func (c *client) GetCustomerCharges(meteringPointIDs []string) ([]CustomerChargeResponse, error) {
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var apiErrorMsg string
	var result struct {
		Result []CustomerChargeResponse `json:"result"`
	}

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetError(&apiErrorMsg).
		SetResult(&result).
		Post("/meteringpoints/meteringpoint/getcharges")

	if err != nil {
		return nil, err
	}
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return nil, err
	}
	return result.Result, nil
}

func (c *client) GetThirdPartyCharges(meteringPointIDs []string) ([]ThirdPartyChargeResponse, error) {
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var apiErrorMsg string
	var result struct {
		Result []ThirdPartyChargeResponse `json:"result"`
	}

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetError(&apiErrorMsg).
		SetResult(&result).
		Post("/meteringpoint/getcharges")

	if err != nil {
		return nil, err
	}
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return nil, err
	}
	return result.Result, nil
}

func (c *client) ExportCharges(meteringPointIDs []string) (io.ReadCloser, error) {
	if c.apiType != CustomerApi {
		return nil, fmt.Errorf("ExportCharges is only available for Customer API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetDoNotParseResponse(true).
		Post("/meteringpoints/charges/export")

	if err != nil || !res.IsSuccess() {
		return nil, fmt.Errorf("failed to export charges, status: %s, err: %v", res.Status(), err)
	}

	return res.RawBody(), nil
}
