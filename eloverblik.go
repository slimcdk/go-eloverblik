package eloverblik

import (
	"fmt"
	"net/http"
	"net/url"
)

// Base struct to authenticate with fetch access tokens
func newClient(refreshToken string, hostUrl url.URL) (*client, error) {
	c := &client{
		refreshToken: refreshToken,
		hostUrl:      hostUrl,
		client:       http.DefaultClient,
	}
	return c, c.authenticate()
}

// NewCustomerClient wraps NewCustomerApi calls.
func NewCustomerClient(refreshToken string) (CustomerAPI, error) {
	hostUrl := url.URL{
		Scheme: HTTP_SCHEME,
		Host:   PROD_HOST,
		Path:   fmt.Sprintf("%s/%s", CUSTOMER_ENDPOINTS, API_VERSION_1),
	}
	return newClient(refreshToken, hostUrl)
}

// NewThirdPartyClient wraps ThirdPartyApi calls.
func NewThirdPartyClient(refreshToken string) (ThirdPartyAPI, error) {
	hostUrl := url.URL{
		Scheme: HTTP_SCHEME,
		Host:   PROD_HOST,
		Path:   fmt.Sprintf("%s/%s", THIRDPART_ENDPOINTS, API_VERSION_1),
	}
	return newClient(refreshToken, hostUrl)
}
