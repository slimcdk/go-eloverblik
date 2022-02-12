package eloverblik

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type MeteringPointPrices struct {
	MeteringPointPrice MeteringPointPrice `json:"result"`
	StatusResponse
}

type MeteringPointPrice struct {
	MeteringPointID string         `json:"meteringPointId"`
	Subscriptions   []Subscription `json:"subscriptions"`
	Fees            []Fee          `json:"fee"`
	Tariffs         []Tariff       `json:"tariffs"`
}

type Subscription struct {
	SubscriptionId string  `json:"subscriptionId"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Owner          string  `json:"owner"`
	ValidFromDate  string  `json:"validFromDate"` // TODO: Parse time
	ValidToDate    string  `json:"validToDate"`   // TODO: Parse time
	Price          float32 `json:"price"`         // TODO: Correct int type?
	Quantity       int     `json:"quantity"`      // TODO: Correct int type?
}

type Fee struct {
	FeeId         string `json:"feeId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Owner         string `json:"owner"`
	ValidFromDate string `json:"validFromDate"` // TODO: Parse time
	ValidToDate   string `json:"validToDate"`   // TODO: Parse time
	Price         int    `json:"price"`         // TODO: Correct int type?
	Quantity      int    `json:"quantity"`      // TODO: Correct int type?
}

type Tariff struct {
	TariffId      string  `json:"tariffId"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Owner         string  `json:"owner"`
	PeriodType    string  `json:"periodType"`
	ValidFromDate string  `json:"validFromDate"` // TODO: Parse time
	ValidToDate   string  `json:"validToDate"`   // TODO: Parse time
	Prices        []Price `json:"prices"`
}

type Price struct {
	Position string  `json:"position"`
	Price    float32 `json:"price"` // TODO: Correct int type?
}

func (c *client) GetCharges(meteringPointIDs []string) ([]MeteringPointPrices, error) {

	// Build URL
	_url := c.hostUrl
	_url.Path += "/MeteringPoints/MeteringPoint/GetCharges"

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
	if err != nil {
		return nil, err
	}

	// Retry if possible
	for isRetryableError(err) {
		return c.GetCharges(meteringPointIDs)
	}

	// Decode response result
	var result struct {
		Result []MeteringPointPrices `json:"result"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Result, err
}
