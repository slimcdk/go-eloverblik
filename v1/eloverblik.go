package eloverblik

import (
	"sync"

	"github.com/go-resty/resty/v2"
)

// client is the internal implementation that satisfies the Customer and ThirdParty interfaces.
// It is safe for concurrent use by multiple goroutines, and renews its data access token
// before it expires.
type client struct {
	refreshToken string
	resty        *resty.Client
	apiType      apiType

	// tokenMu guards accessToken, the cached data access token. It is held across the
	// /token request, so goroutines that need a data access token at the same time, the
	// first one or the one that replaces an expiring token, share the one request.
	tokenMu     sync.Mutex
	accessToken string
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
// The client is safe for concurrent use by multiple goroutines. Create one and share it:
// it fetches a data access token from /token on the first call that needs one, and the
// API allows only 2 such calls a minute. The data access token lasts about 24 hours; the
// client fetches a new one when the cached one has expired or expires within five
// minutes, so a long running process can keep using the same client.
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
// The client is safe for concurrent use by multiple goroutines. Create one and share it:
// it fetches a data access token from /token on the first call that needs one, and the
// API allows only 2 such calls a minute. The data access token lasts about 24 hours; the
// client fetches a new one when the cached one has expired or expires within five
// minutes, so a long running process can keep using the same client.
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
