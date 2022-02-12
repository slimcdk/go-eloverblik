package eloverblik

import (
	"fmt"
	"net/http"
	"net/url"
)

type client struct {
	refreshToken string
	accessToken  string
	hostUrl      url.URL
	client       *http.Client
}

type CustomerClient struct{ *client }
type ThirdPartyClient struct{ *client }

// Base struct to authenticate with fetch access tokens
func newClient(refreshToken string, hostUrl url.URL) (*client, error) {
	c := client{
		refreshToken: refreshToken,
		hostUrl:      hostUrl,
		client:       http.DefaultClient,
	}
	return &c, c.authenticate()
}

// NewCustomerClient wraps NewCustomerApi calls.
func NewCustomerClient(refreshToken string) (CustomerAPI, error) {

	hostUrl := url.URL{
		Scheme: HTTP_SCHEME,
		Host:   PROD_HOST,
		Path:   fmt.Sprintf("%s/%s", CUSTOMER_ENDPOINTS, API_VERSION_1),
	}

	c, err := newClient(refreshToken, hostUrl)
	return &CustomerClient{client: c}, err
}

// NewThirdPartyClient wraps ThirdPartyApi calls.
func NewThirdPartyClient(refreshToken string) (ThirdPartyAPI, error) {
	hostUrl := url.URL{
		Scheme: HTTP_SCHEME,
		Host:   PROD_HOST,
		Path:   fmt.Sprintf("%s/%s", THIRDPART_ENDPOINTS, API_VERSION_1),
	}

	c, err := newClient(refreshToken, hostUrl)
	return &ThirdPartyClient{client: c}, err
}
