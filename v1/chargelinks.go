package eloverblik

import (
	"fmt"
	"time"
)

// ChargeLinksWithChargesRequest is the request envelope for getchargelinkswithcharges.
// Unlike the other POST endpoints it does not take a list of metering point IDs, but a
// flat list of query items, each carrying its own date interval.
type ChargeLinksWithChargesRequest struct {
	Query []ChargeLinksWithChargesQueryItem `json:"query"`
}

// ChargeLinksWithChargesQueryItem is a single metering point and the date interval the
// charge links and charges are requested for.
type ChargeLinksWithChargesQueryItem struct {
	MeteringPointID string       `json:"meteringPointId"`
	From            FlexibleTime `json:"from"`
	To              FlexibleTime `json:"to"`
}

// ChargeLinksWithChargesResponse holds the charge links per metering point and the
// charge information they refer to. Charge information is deduplicated by the API and
// therefore returned once per charge, alongside the links rather than nested in them.
type ChargeLinksWithChargesResponse struct {
	Results            []ChargeLinksWithChargesResult `json:"results"`
	ChargeInformations []ChargeInformation            `json:"chargeInformations"`
}

// ChargeLinksWithChargesResult holds the charge links of a single metering point. Errors
// are reported per metering point in Error, the remaining metering points still resolve.
type ChargeLinksWithChargesResult struct {
	MeteringPointID string       `json:"meteringPointId"`
	Error           string       `json:"error,omitempty"`
	ChargeLinks     []ChargeLink `json:"chargeLinks"`
}

// ChargeLink links a metering point to a charge for one or more periods.
type ChargeLink struct {
	MeteringPointID   string             `json:"meteringPointId"`
	ChargeIdentifier  ChargeIdentifier   `json:"chargeIdentifier"`
	ChargeLinkPeriods []ChargeLinkPeriod `json:"chargeLinkPeriods"`
}

// ChargeIdentifier uniquely identifies a charge and is the key between a ChargeLink and
// the ChargeInformation carrying its prices.
type ChargeIdentifier struct {
	Code  string `json:"code"`
	Owner string `json:"owner"`
	Type  string `json:"type"`
}

// ChargeLinkPeriod is the interval a charge is linked to a metering point in. Factor is
// the quantity the charge applies with, e.g. the number of subscriptions. To is nil for
// an open ended link.
type ChargeLinkPeriod struct {
	Factor int          `json:"factor"`
	From   FlexibleTime `json:"from"`
	To     FlexibleTime `json:"to"`
}

// ChargeInformation describes a charge and the price series it is settled with.
// TaxIndicator reports whether the charge is a tax, Resolution is the period each
// ChargeSeriesPoint covers, e.g. PT1H for an hourly tariff.
type ChargeInformation struct {
	ChargeIdentifier         ChargeIdentifier          `json:"chargeIdentifier"`
	TaxIndicator             bool                      `json:"taxIndicator"`
	Resolution               string                    `json:"resolution"`
	PricingCategory          string                    `json:"pricingCategory"`
	ChargeInformationPeriods []ChargeInformationPeriod `json:"chargeInformationPeriods"`
	ChargeSeriesPoints       []ChargeSeriesPoint       `json:"chargeSeriesPoints"`
}

// ChargeInformationPeriod is the descriptive part of a charge in a given interval. To is
// nil for an open ended period.
type ChargeInformationPeriod struct {
	Name                 string       `json:"name"`
	Description          string       `json:"description"`
	TransparentInvoicing bool         `json:"transparentInvoicing"`
	From                 FlexibleTime `json:"from"`
	To                   FlexibleTime `json:"to"`
	VATClassification    string       `json:"vatClassification"`
}

// ChargeSeriesPoint is a dated price of a charge. The price is valid in [From, To).
type ChargeSeriesPoint struct {
	From  FlexibleTime `json:"from"`
	To    FlexibleTime `json:"to"`
	Price float64      `json:"price"`
}

// GetChargeLinksWithCharges fetches the charge links of the given metering points in the
// interval [from, to), together with the dated price series of every charge they link to.
//
// Where GetCustomerCharges and GetThirdPartyCharges only return charges that are
// currently valid or take effect in the future, this endpoint returns the historic price
// series as well, and is therefore the one to price past consumption with.
//
// Both OpenAPI documents declare the endpoint, but as of 2026-07-13 the live API answers
// 404 for it on BOTH the Customer and the Third-Party API, with valid tokens, on every
// documented path, while its getcharges sibling answers normally on the same tokens.
// Energinet has specified it but not deployed it: expect an error rather than data until
// they do.
func (c *client) GetChargeLinksWithCharges(meteringPointIDs []string, from, to time.Time) (*ChargeLinksWithChargesResponse, error) {

	// Ensure access token is fresh
	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var path string
	switch c.apiType {
	case CustomerApi:
		path = "/meteringpoints/meteringpoint/getchargelinkswithcharges"
	case ThirdPartyApi:
		path = "/meteringpoint/getchargelinkswithcharges"
	default:
		return nil, fmt.Errorf("unsupported API type for GetChargeLinksWithCharges")
	}

	// Response structs
	var apiErrBody apiErrorBody
	var result struct {
		Result ChargeLinksWithChargesResponse `json:"result"`
	}

	// Execute request
	res, err := c.resty.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(accessToken).
		SetBody(chargeLinksRequest(meteringPointIDs, from, to)).
		SetResult(&result).
		SetError(&apiErrBody).
		Post(path)

	if err != nil {
		return nil, err
	}

	// Handle API errors
	if err = apiErrorFromBody(apiErrBody, res.StatusCode()); err != nil {
		return nil, err
	}

	return &result.Result, nil
}

// chargeLinksRequest builds the query envelope of getchargelinkswithcharges. The interval
// is per metering point in the API, this client applies the same interval to all of them.
func chargeLinksRequest(meteringPointIDs []string, from, to time.Time) ChargeLinksWithChargesRequest {
	query := make([]ChargeLinksWithChargesQueryItem, 0, len(meteringPointIDs))
	for _, id := range meteringPointIDs {
		query = append(query, ChargeLinksWithChargesQueryItem{
			MeteringPointID: id,
			From:            FlexibleTime{Time: from.In(cph)},
			To:              FlexibleTime{Time: to.In(cph)},
		})
	}
	return ChargeLinksWithChargesRequest{Query: query}
}
