package eloverblik

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type client struct {
	refreshToken string
	accessToken  string

	apiRoot string
	client  *http.Client
}

func (c *client) authenticate() error {

	// Fetch access token
	_url := fmt.Sprintf("%s/%s/Token", c.apiRoot, API_VERSION_1)
	req, err := http.NewRequest("GET", _url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.refreshToken))
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return ErrorClientConnection(res.Status)
	}

	// Save access token
	tokenResponse := getTokenResponse{}
	json.NewDecoder(res.Body).Decode(&tokenResponse)
	c.accessToken = tokenResponse.AccessToken
	return err
}

func newClient(refreshToken, apiRoot string) (*client, error) {
	c := &client{
		refreshToken: refreshToken,
		apiRoot:      apiRoot,
		client:       http.DefaultClient,
	}
	return c, c.authenticate()
}

func NewCustomerClient(refreshToken string) (CustomerAPI, error) {
	apiRoot := fmt.Sprintf("%s/%s", API_PROD_ROOT, API_CUSTOMER_ENDPOINTS)
	return newClient(refreshToken, apiRoot)
}

func NewThirdPartyClient(refreshToken string) (ThirdPartyAPI, error) {
	apiRoot := fmt.Sprintf("%s/%s", API_PROD_ROOT, API_THIRDPART_ENDPOINTS)
	return newClient(refreshToken, apiRoot)
}

func (c *client) GetDataAccessToken() (string, error) {
	if c.accessToken == "" {
		if err := c.authenticate(); err != nil {
			return "", err
		}
	}
	return c.accessToken, nil
}

func (c *client) GetAuthorizations() error       { return nil }
func (c *client) AddRelationOnID() error         { return nil }
func (c *client) AddRelationOnAccessCode() error { return nil }
func (c *client) DeleteRelation() error          { return nil }
func (c *client) GetMeteringPoints() error       { return nil }
func (c *client) GetMeteringPointDetails() error { return nil }
func (c *client) GetCharges() error              { return nil }
func (c *client) GetTimeSeries() error           { return nil }
func (c *client) GetMeterReadings() error        { return nil }
