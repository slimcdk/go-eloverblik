package eloverblik

import (
	"github.com/go-resty/resty/v2"
)

// client is the internal implementation that satisfies the Customer and ThirdParty interfaces.
type client struct {
	refreshToken string
	accessToken  string
	resty        *resty.Client
	apiType      apiType
}

type apiType int

const (
	CustomerApi apiType = iota
	ThirdPartyApi
)

const (
	// apiVersionHeader pins the API contract. Every operation in both OpenAPI specs
	// declares an "api-version" header with a default of "1.0". Sending it explicitly
	// keeps the response shapes stable, should the server-side default ever move.
	apiVersionHeader = "api-version"
	apiVersion       = "1.0"
)

// NewCustomer creates and returns a new Eloverblik Customer client.
// Zero or more options can be passed to configure the client.
//
// Example:
//
//	customerClient := eloverblik.NewCustomer(refreshToken)
func NewCustomer(refreshToken string, opts ...Option) Customer {
	c := &client{
		refreshToken: refreshToken,
		resty:        newRestyClient("https://" + prodModeHost + "/customerapi/api"),
		apiType:      CustomerApi,
	}
	applyOptions(c, opts)
	return c
}

// NewThirdParty creates and returns a new Eloverblik ThirdParty client.
// Zero or more options can be passed to configure the client.
//
// Example:
//
//	thirdPartyClient := eloverblik.NewThirdParty(refreshToken)
func NewThirdParty(refreshToken string, opts ...Option) ThirdParty {
	c := &client{
		refreshToken: refreshToken,
		resty:        newRestyClient("https://" + prodModeHost + "/thirdpartyapi/api"),
		apiType:      ThirdPartyApi,
	}
	applyOptions(c, opts)
	return c
}

// newRestyClient creates the HTTP client shared by both APIs: the base URL, the pinned
// api-version header and the default retry policy for the documented rate limits.
func newRestyClient(baseURL string) *resty.Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader(apiVersionHeader, apiVersion)

	return setRetryPolicy(client, DefaultRetryCount, DefaultRetryMaxWait)
}

// applyOptions applies the options to the client. It runs after the resty client has
// been created, so an option can configure it.
func applyOptions(c *client, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
}
