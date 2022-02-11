package eloverblik

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Get tokens
type getTokenResponse struct {
	AccessToken string `json:"result"`
}

// Fetches and sets a access token on the base client
func (c *client) authenticate() error {

	_url := c.hostUrl
	_url.Path += "/Token"

	req, err := http.NewRequest("GET", _url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.refreshToken))
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return ErrorClientConnection(res.StatusCode)
	}

	// Save access token
	tokenResponse := getTokenResponse{}
	json.NewDecoder(res.Body).Decode(&tokenResponse)
	c.accessToken = tokenResponse.AccessToken
	return err
}

func (c *client) GetDataAccessToken() (string, error) {
	if c.accessToken == "" {
		if err := c.authenticate(); err != nil {
			return "", err
		}
	}
	return c.accessToken, nil
}

func (c *client) GetAuthorizations() error { return nil }
