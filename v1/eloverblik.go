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

// NewCustomer creates and returns a new Eloverblik Customer client.
//
// Example:
//
//	customerClient := eloverblik.NewCustomer(refreshToken)
func NewCustomer(refreshToken string) Customer {
	return &client{
		refreshToken: refreshToken,
		resty:        resty.New().SetBaseURL("https://" + prodModeHost + "/customerapi/api"),
		apiType:      CustomerApi,
	}
}

// NewThirdParty creates and returns a new Eloverblik ThirdParty client.
//
// Example:
//
//	thirdPartyClient := eloverblik.NewThirdParty(refreshToken)
func NewThirdParty(refreshToken string) ThirdParty {
	return &client{
		refreshToken: refreshToken,
		resty:        resty.New().SetBaseURL("https://" + prodModeHost + "/thirdpartyapi/api"),
		apiType:      ThirdPartyApi,
	}
}
