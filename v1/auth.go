package eloverblik

import "net/http"

// Fetches and sets a access token on the base client
func (c *client) authenticate() error {

	// Response struct
	var result struct {
		AccessToken string `json:"result"`
	}

	// Request preflight
	req := c.resty.R().
		SetHeader("Accept", "application/json").
		SetAuthToken(c.refreshToken).
		SetResult(&result)

	// Execute request
	res, err := req.Get("/token")
	if err != nil || res.StatusCode() != http.StatusOK {
		return err
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

func (c *client) GetAuthorizations() error { return nil }
