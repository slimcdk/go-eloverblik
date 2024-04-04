package eloverblik

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type client struct {
	refreshToken string
	accessToken  string
	resty        *resty.Client
	log          *log.Logger
}

// Base struct
func buildClient(refreshToken string, apiType APIType) (client, error) {

	// Set resty client parameters
	r := resty.NewWithClient(http.DefaultClient)
	r.SetBaseURL(fmt.Sprintf("https://%s/%s/api/1", hostModeMap[Mode], apiType+"api"))
	c := client{refreshToken: refreshToken, resty: r}
	return c, c.authenticate()
}

// NewCustomerClient wraps NewCustomerApi calls.
func CustomerClient(refreshToken string) (Customer, error) {
	c, err := buildClient(refreshToken, customerApiAtype)
	return Customer(&c), err
}

// NewThirdPartyClient wraps ThirdPartyApi calls.
func ThirdPartyClient(refreshToken string) (ThirdParty, error) {
	c, err := buildClient(refreshToken, thirdPartyApiType)
	return ThirdParty(&c), err
}

// SetMode
func SetMode(value string) {
	if value == "" {
		value = TestMode
	}

	switch value {
	case TestMode:
		Mode = TestMode
	case ReleaseMode:
		Mode = ReleaseMode
	default:
		panic("gin mode unknown: " + value + " (available mode: debug release test)")
	}
}

func (c *client) SetLogger(log *log.Logger) {
	c.log = log
}
