package eloverblik

import (
	"fmt"
	"net/http"
)

func (c *client) AddRelationByID(meteringPointIDs []string) ([]StringResponse, error) {
	if c.apiType != CustomerApi {
		return nil, fmt.Errorf("AddRelationByID is only available for Customer API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []StringResponse `json:"result"`
	}
	var apiErrorMsg string

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetBody(meteringPointIDsToRequestStruct(meteringPointIDs)).
		SetResult(&result).
		SetError(&apiErrorMsg).
		Post("/meteringpoints/meteringpoint/relation/add")

	if err != nil {
		return nil, err
	}
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return nil, err
	}

	return result.Result, nil
}

func (c *client) AddRelationByWebAccessCode(meteringPointID, webAccessCode string) (string, error) {
	if c.apiType != CustomerApi {
		return "", fmt.Errorf("AddRelationByWebAccessCode is only available for Customer API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return "", err
	}

	var result StringResponse
	var apiErrorMsg string

	path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/add/%s/%s", meteringPointID, webAccessCode)

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetResult(&result).
		SetError(&apiErrorMsg).
		Put(path)

	if err != nil {
		return "", err
	}
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return "", err
	}

	return result.Result, nil
}

func (c *client) DeleteRelation(meteringPointID string) (bool, error) {
	if c.apiType != CustomerApi {
		return false, fmt.Errorf("DeleteRelation is only available for Customer API")
	}

	accessToken, err := c.GetDataAccessToken()
	if err != nil {
		return false, err
	}

	path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/%s", meteringPointID)

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		Delete(path)

	if err != nil {
		return false, err
	}

	return res.StatusCode() == http.StatusOK, nil
}
