package eloverblik

import "time"

type Aggregation string
type APIType string

const (
	testModeHost      string  = "apipreprod.eloverblik.dk"
	prodModeHost      string  = "api.eloverblik.dk"
	customerApiAtype  APIType = "customer"
	thirdPartyApiType APIType = "thirdparty"
)

var (
	ReleaseMode string = "prod"
	TestMode    string = "preprod"

	// Default settings
	Mode    string  = TestMode
	ApiType APIType = customerApiAtype

	// Lookup tables
	hostModeMap map[string]string = map[string]string{
		"prod":          prodModeHost, // Production purposes
		"production":    prodModeHost, // Production purposes
		"release":       prodModeHost, // Production purposes
		"test":          testModeHost, // Testing purposes
		"preprod":       testModeHost, // Testing purposes
		"preproduction": testModeHost, // Testing purposes
	}
)

const (
	Actual  Aggregation = "Actual"
	Quarter Aggregation = "Quarter"
	Hour    Aggregation = "Hour"
	Day     Aggregation = "Day"
	Month   Aggregation = "Month"
	Year    Aggregation = "Year"
)

const (
	requestDateFormat string = "2006-01-02"

	MaximumDayRequestLeap  int           = 730
	MaximumRequestDuration time.Duration = time.Hour * 24 * 730
)
