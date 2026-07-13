package eloverblik

import (
	"fmt"
)

type AuthorizationScope string

const (
	AuthScopeID          AuthorizationScope = "authorizationId"
	AuthScopeCustomerCVR AuthorizationScope = "customerCVR"
	AuthScopeCustomerKey AuthorizationScope = "customerKey"
)

type Authorization struct {
	ID                          string       `json:"id"`
	ThirdPartyName              string       `json:"thirdPartyName"`
	ValidFrom                   string       `json:"validFrom"`
	ValidTo                     string       `json:"validTo"`
	CustomerName                string       `json:"customerName"`
	CustomerCVR                 string       `json:"customerCVR"`
	CustomerKey                 string       `json:"customerKey"`
	IncludeFutureMeteringPoints bool         `json:"includeFutureMeteringPoints"`
	Timestamp                   FlexibleTime `json:"timeStamp"`
}

type ThirdPartyMeteringPoint struct {
	MeteringPointID         string               `json:"meteringPointId"`
	TypeOfMP                string               `json:"typeOfMP"`
	AccessFrom              string               `json:"accessFrom"`
	AccessTo                string               `json:"accessTo"`
	StreetCode              string               `json:"streetCode"`
	StreetName              string               `json:"streetName"`
	BuildingNumber          string               `json:"buildingNumber"`
	FloorID                 string               `json:"floorId"`
	RoomID                  string               `json:"roomId"`
	Postcode                string               `json:"postcode"`
	CityName                string               `json:"cityName"`
	CitySubDivisionName     string               `json:"citySubDivisionName"`
	MunicipalityCode        string               `json:"municipalityCode"`
	LocationDescription     string               `json:"locationDescription"`
	SettlementMethod        string               `json:"settlementMethod"`
	MeterReadingOccurrence  string               `json:"meterReadingOccurrence"`
	FirstConsumerPartyName  string               `json:"firstConsumerPartyName"`
	SecondConsumerPartyName string               `json:"secondConsumerPartyName"`
	ConsumerCVR             string               `json:"consumerCVR"`
	DataAccessCVR           string               `json:"dataAccessCVR"`
	MeterNumber             string               `json:"meterNumber"`
	ConsumerStartDate       FlexibleTime         `json:"consumerStartDate"`
	ChildMeteringPoints     []ChildMeteringPoint `json:"childMeteringPoints"`
}

// Fetches and sets a access token on the base client
func (c *client) authenticate() error {

	// Response struct
	var result struct {
		AccessToken string `json:"result"`
	}
	var apiErrBody apiErrorBody

	// Request preflight
	req := c.resty.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(c.refreshToken).
		SetResult(&result).
		SetError(&apiErrBody)

	// Execute request
	res, err := req.Get("/token")
	if err != nil {
		return err
	}
	if err = apiErrorFromBody(apiErrBody, res.StatusCode()); err != nil {
		return err
	}

	// A response without a token leaves the client unauthenticated, which would make
	// every following call fail with a confusing 401 instead of the real cause
	if result.AccessToken == "" {
		return ErrorErrorCreatingToken
	}

	// Set access token on client
	c.accessToken = result.AccessToken
	return nil
}

func (c *client) GetDataAccessToken() (string, error) {
	if c.accessToken == "" {
		if err := c.authenticate(); err != nil {
			return c.accessToken, err
		}
	}
	return c.accessToken, nil
}

func (c *client) GetAuthorizations() ([]Authorization, error) {
	if c.apiType != ThirdPartyApi {
		return nil, fmt.Errorf("GetAuthorizations is only available for ThirdParty API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []Authorization `json:"result"`
	}
	var apiErrBody apiErrorBody

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetResult(&result).
		SetError(&apiErrBody).
		Get("/authorization/authorizations")

	if err != nil {
		return nil, err
	}
	if err = apiErrorFromBody(apiErrBody, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, nil
}

func (c *client) GetMeteringPointsForScope(scope AuthorizationScope, identifier string) ([]ThirdPartyMeteringPoint, error) {
	if c.apiType != ThirdPartyApi {
		return nil, fmt.Errorf("GetMeteringPointsForScope is only available for ThirdParty API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []ThirdPartyMeteringPoint `json:"result"`
	}
	var apiErrBody apiErrorBody

	path := fmt.Sprintf("/authorization/authorization/meteringpoints/%s/%s", scope, identifier)

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetResult(&result).
		SetError(&apiErrBody).
		Get(path)

	if err != nil {
		return nil, err
	}
	if err = apiErrorFromBody(apiErrBody, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, nil
}

func (c *client) GetMeteringPointIDsForScope(scope AuthorizationScope, identifier string) ([]string, error) {
	if c.apiType != ThirdPartyApi {
		return nil, fmt.Errorf("GetMeteringPointIDsForScope is only available for ThirdParty API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []string `json:"result"`
	}
	var apiErrBody apiErrorBody

	path := fmt.Sprintf("/authorization/authorization/meteringpointids/%s/%s", scope, identifier)

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetResult(&result).
		SetError(&apiErrBody).
		Get(path)

	if err != nil {
		return nil, err
	}
	if err = apiErrorFromBody(apiErrBody, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, nil
}

func (c *client) IsAlive() (bool, error) {
	res, err := c.resty.R().Get("/isalive")
	if err != nil {
		return false, err
	}
	return res.IsSuccess(), nil
}
