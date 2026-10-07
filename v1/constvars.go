package eloverblik

import (
	"fmt"
	"time"

	// The client works in Copenhagen time on every platform. time.LoadLocation reads the
	// host's zone database first, but Windows has none of its own, and neither have
	// minimal container images: the embedded copy is the fallback that makes
	// Europe/Copenhagen load there too.
	_ "time/tzdata"
)

type Aggregation string
type Resolution string

// APIType is the type of ApiType.
//
// Deprecated: APIType serves only ApiType, which has no effect. It is unrelated to
// TokenClaims.APIType, which reports the API a token was issued for as CustomerApi or
// ThirdPartyApi. APIType will be removed in v2.
type APIType string

const (
	// prodModeHost serves both APIs. NewCustomer and NewThirdParty always call it.
	prodModeHost string = "api.eloverblik.dk"

	// customerApiAtype is the default of the deprecated ApiType, and goes with it in v2.
	customerApiAtype APIType = "customer"
)

var (
	// ReleaseMode is the value of Mode that pointed the clients at the production API
	// before v1.0.0.
	//
	// Deprecated: ReleaseMode has no effect, and neither has Mode: every client calls the
	// production API at api.eloverblik.dk. ReleaseMode will be removed in v2.
	ReleaseMode string = "prod"

	// TestMode is the value of Mode that pointed the clients at Energinet's pre-production
	// API before v1.0.0. It is still the default of Mode.
	//
	// Deprecated: TestMode has no effect, and neither has Mode: every client calls the
	// production API at api.eloverblik.dk, never the pre-production one. TestMode will be
	// removed in v2.
	TestMode string = "preprod"

	// Mode chose the environment the clients called, TestMode or ReleaseMode, before v1.0.0.
	//
	// Deprecated: Mode has no effect. Nothing reads it, so every client calls the production
	// API at api.eloverblik.dk whatever Mode holds, its default TestMode included. Mode will
	// be removed in v2.
	Mode string = TestMode

	// ApiType names an API, "customer" by default.
	//
	// Deprecated: ApiType has no effect. Nothing reads it: the constructor decides the API,
	// so NewCustomer returns a Customer API client and NewThirdParty a Third-Party API
	// client, whatever ApiType holds. ApiType will be removed in v2.
	ApiType APIType = customerApiAtype
)

var cph = mustLoadLocation("Europe/Copenhagen")

// mustLoadLocation loads a time zone from the host's database or, failing that, the
// embedded one. It cannot fail for a valid name; a panic at start-up still beats the nil
// *time.Location the error used to leave behind, which panicked later in every Time.In.
func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(fmt.Errorf("eloverblik: load time zone %s: %w", name, err))
	}
	return loc
}

const (
	Actual  Aggregation = "Actual"
	Quarter Aggregation = "Quarter"
	Hour    Aggregation = "Hour"
	Day     Aggregation = "Day"
	Month   Aggregation = "Month"
	Year    Aggregation = "Year"

	PT15M Resolution = "PT15M"
	PT1H  Resolution = "PT1H"
	PT1D  Resolution = "PT1D"
	P1M   Resolution = "P1M"
	PT1Y  Resolution = "PT1Y"

	// The OpenAPI description documents the day and year resolutions with their ISO 8601
	// spellings, P1D and P1Y, and adds PXD for profiled energy quantities covering a
	// variable number of days. The live API sends PT1D and PT1Y, so both spellings are
	// accepted when flattening a time series.
	P1D Resolution = "P1D"
	P1Y Resolution = "P1Y"
	PXD Resolution = "PXD"
)

const (
	MaximumDayRequestLeap  int           = 730
	MaximumRequestDuration time.Duration = time.Hour * 24 * 730
)
