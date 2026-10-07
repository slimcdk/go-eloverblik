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
type APIType string
type Resolution string

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

	cph = mustLoadLocation("Europe/Copenhagen")
)

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
