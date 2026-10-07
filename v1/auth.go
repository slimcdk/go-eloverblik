package eloverblik

import (
	"fmt"
	"time"
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

// authenticate exchanges the refresh token for a data access token and caches it on the
// client. The caller must hold c.tokenMu.
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

// dataAccessTokenRenewalMargin is how long before its expiry the cached data access token
// is replaced, so a request does not set out with a token that expires on the way.
const dataAccessTokenRenewalMargin = 5 * time.Minute

// GetDataAccessToken implements Client.GetDataAccessToken, which documents its contract.
// c.tokenMu is held across the /token request, so the goroutines that need a token while
// one is being fetched wait for that request instead of sending their own.
func (c *client) GetDataAccessToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken == "" || expiresWithin(c.accessToken, dataAccessTokenRenewalMargin) {
		if err := c.authenticate(); err != nil {
			return c.cachedTokenOr(err)
		}
	}
	return c.accessToken, nil
}

// cachedTokenOr is the outcome of a /token request that failed with err. A cached token
// inside the renewal margin still works for a few minutes, so it is returned in place of
// the error until it has actually expired. The caller must hold c.tokenMu.
func (c *client) cachedTokenOr(err error) (string, error) {
	if c.accessToken == "" || expiresWithin(c.accessToken, 0) {
		return "", err
	}
	return c.accessToken, nil
}

// expiresWithin reports whether the token's exp claim is past or less than margin away. A
// token without a readable expiry, including one whose exp is null, zero or negative, is
// treated as not expiring, so the client keeps it and leaves it to the API to reject it.
func expiresWithin(token string, margin time.Duration) bool {
	claims, err := ParseToken(token)
	if err != nil || claims.ExpiresAt.IsZero() {
		return false
	}
	return time.Until(claims.ExpiresAt) < margin
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
