package eloverblik

import (
	"fmt"
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

	// The API answers with a boolean envelope, so a business error such as
	// "[20010] Relation not found" has to be read out of the body: the HTTP status
	// alone does not tell the relation was actually deleted.
	var result struct {
		Result *bool `json:"result"`
	}
	var apiErrorMsg string

	path := fmt.Sprintf("/meteringpoints/meteringpoint/relation/%s", meteringPointID)

	res, err := c.resty.R().
		SetAuthToken(accessToken).
		SetResult(&result).
		SetError(&apiErrorMsg).
		Delete(path)

	if err != nil {
		return false, err
	}
	if err = apiError(apiErrorMsg, res.StatusCode()); err != nil {
		return false, err
	}
	if result.Result != nil {
		return *result.Result, nil
	}

	return res.IsSuccess(), nil
}
