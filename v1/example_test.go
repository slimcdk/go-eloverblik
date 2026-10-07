package eloverblik_test

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"time"

	eloverblik "github.com/slimcdk/go-eloverblik/v1"
)

// The examples that call the API have no Output comment, so go test compiles them but
// never runs them: nothing here reaches Eloverblik.

// This example fetches the hourly time series of every metering point linked to the
// user for September 2026, and flattens each into one point per hour.
func ExampleNewCustomer() {
	// One client per refresh token, shared by the whole program: it fetches, caches and
	// renews the data access token itself.
	client := eloverblik.NewCustomer(os.Getenv("ELO_TOKEN"))

	meteringPoints, err := client.GetMeteringPoints(false)
	if err != nil {
		log.Fatal(err)
	}
	ids := make([]string, 0, len(meteringPoints))
	for _, meteringPoint := range meteringPoints {
		ids = append(ids, meteringPoint.MeteringPointID)
	}

	// The API reads the bounds as Copenhagen dates, and to is excluded: this is the
	// whole of September.
	cph, err := time.LoadLocation("Europe/Copenhagen")
	if err != nil {
		log.Fatal(err)
	}
	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, cph)
	to := time.Date(2026, time.October, 1, 0, 0, 0, 0, cph)

	// At most 10 metering points per request
	for batch := range slices.Chunk(ids, 10) {
		series, err := client.GetTimeSeries(batch, from, to, eloverblik.Hour)
		if err != nil {
			log.Fatal(err) // the request failed as a whole
		}

		for _, ts := range series {
			// A metering point can fail on its own while the others succeed
			if err := ts.Err(); err != nil {
				log.Printf("skipping: %v", err)
				continue
			}
			for _, point := range ts.Flatten() {
				fmt.Printf("%s %s %.3f %s %s\n",
					ts.ID, point.From.Format(time.RFC3339), point.Measurement, point.Unit, point.Quality)
			}
		}
	}
}

// This example lists the metering points of every customer who has given the third
// party a power of attorney, with their master data.
func ExampleNewThirdParty() {
	client := eloverblik.NewThirdParty(os.Getenv("ELO_TOKEN"))

	authorizations, err := client.GetAuthorizations()
	if err != nil {
		log.Fatal(err)
	}

	for _, authorization := range authorizations {
		ids, err := client.GetMeteringPointIDsForScope(eloverblik.AuthScopeID, authorization.ID)
		if err != nil {
			log.Fatal(err)
		}

		// At most 10 metering points per request
		for batch := range slices.Chunk(ids, 10) {
			details, err := client.GetMeteringPointDetails(batch)
			if err != nil {
				log.Fatal(err)
			}

			for _, detail := range details {
				if err := detail.Err(); err != nil {
					log.Printf("skipping: %v", err)
					continue
				}
				mp := detail.Result
				fmt.Println(authorization.CustomerName, mp.MeteringPointID, mp.TypeOfMP,
					mp.StreetName, mp.BuildingNumber, mp.Postcode, mp.CityName)
			}
		}
	}
}

// This example asks for this month so far, and falls back to last month on the 1st,
// when this month has no complete day yet. Its output depends on the day it runs.
func ExampleGetDatesFromPeriod() {
	from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.ThisMonth)
	if errors.Is(err, eloverblik.ErrorPeriodHasNoCompleteDay) {
		from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
	}
	if err != nil {
		log.Fatal(err)
	}
	// to is excluded: now for this_month, so a call such as GetTimeSeries stops before
	// today, and the 1st of this month for last_month.
	fmt.Printf("[%s, %s)\n", from.Format(time.DateOnly), to.Format(time.DateOnly))

	// Weeks run from Monday to Sunday: last_week is [Monday, the following Monday).
	from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.LastWeek)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(from.Weekday(), to.Weekday()) // Monday Monday
}

// This example reads the claims of a refresh token, without a request. The token is
// made up for the example: its payload is {"tokenType":"CUSTOMERAPI_Refresh",
// "tokenName":"example","roles":"ReadPrivate, ReadBusiness","exp":1830000000}.
func ExampleParseToken() {
	const refreshToken = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
		"eyJ0b2tlblR5cGUiOiJDVVNUT01FUkFQSV9SZWZyZXNoIiwidG9rZW5OYW1lIjoiZXhhbXBsZSIsInJvbGVzIjoiUmVhZFByaXZhdGUsIFJlYWRCdXNpbmVzcyIsImV4cCI6MTgzMDAwMDAwMH0." +
		"c2lnbmF0dXJl"

	claims, err := eloverblik.ParseToken(refreshToken)
	if err != nil {
		log.Fatal(err)
	}

	apiType, err := claims.APIType()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(claims.TokenType, claims.TokenName)
	fmt.Println("refresh token:", claims.IsRefreshToken())
	fmt.Println("customer API:", apiType == eloverblik.CustomerApi)
	fmt.Println("roles:", claims.Roles)
	fmt.Println("expires:", claims.ExpiresAt.Format(time.RFC3339)) // in Copenhagen time
	// Output:
	// CUSTOMERAPI_Refresh example
	// refresh token: true
	// customer API: true
	// roles: [ReadPrivate ReadBusiness]
	// expires: 2027-12-28T14:20:00+01:00
}

// This example reads the outcome of one metering point in a batch response. The result
// is built here the way GetTimeSeries returns it for a metering point whose data does
// not cover the requested period, inside an otherwise successful response.
func ExampleStatusResponse_Err() {
	ts := eloverblik.TimeSeries{
		Success:   false,
		ErrorCode: 30018,
		ErrorText: "MeteringPointDataNotAvailableForTheRequestedPeriod",
		ID:        "571313180100000002",
	}

	err := ts.Err()
	fmt.Println(err)
	fmt.Println(errors.Is(err, eloverblik.ErrorMeteringPointDataNotAvailableForTheRequestedPeriod))
	fmt.Println("points:", len(ts.Flatten())) // a failed result holds no data
	// Output:
	// eloverblik: metering point 571313180100000002: 30018 MeteringPointDataNotAvailableForTheRequestedPeriod
	// true
	// points: 0
}

// This example tells the errors of a call apart. A sentinel matches with errors.Is
// whichever shape the API answered with, wrapped or not; a problem document is an
// *APIError, which errors.AsType (or errors.As) gets the status and trace ID out of.
func ExampleAPIError() {
	client := eloverblik.NewCustomer(os.Getenv("ELO_TOKEN"))

	details, err := client.GetMeteringPointDetails([]string{"571313180100000002"})
	if err != nil {
		// The trace ID is what Energinet support asks for
		if apiErr, ok := errors.AsType[*eloverblik.APIError](err); ok {
			log.Printf("eloverblik answered %d %s, traceId %s", apiErr.StatusCode, apiErr.Title, apiErr.TraceID)
		}

		switch {
		case errors.Is(err, eloverblik.ErrorTokenNotValid), errors.Is(err, eloverblik.ErrorUnauthorized):
			log.Fatal("token rejected: an expired or revoked refresh token must be replaced")
		case errors.Is(err, eloverblik.ErrorTooManyRequests):
			log.Fatal("still rate limited after the retries; try again in a minute")
		default:
			log.Fatal(err)
		}
	}

	for _, detail := range details {
		err := detail.Err()
		switch {
		case errors.Is(err, eloverblik.ErrorRelationNotFound):
			log.Printf("%s is not linked to the user", detail.ID)
		case err != nil:
			log.Printf("skipping: %v", err)
		default:
			fmt.Println(detail.Result.MeteringPointID, detail.Result.TypeOfMP)
		}
	}
}

// This example exports the daily time series of last month as CSV. The export is
// streamed, and the caller must close it. Unlike GetTimeSeries, the export includes to,
// so it ends on the last day of the month instead of the 1st of the next.
func ExampleCustomer_exportTimeSeries() {
	client := eloverblik.NewCustomer(os.Getenv("ELO_TOKEN"))

	from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
	if err != nil {
		log.Fatal(err)
	}
	lastDay := to.AddDate(0, 0, -1)

	csv, err := client.ExportTimeSeries([]string{"571313180100000002"}, from, lastDay, eloverblik.Day)
	if err != nil {
		log.Fatal(err)
	}

	_, copyErr := io.Copy(os.Stdout, csv)
	closeErr := csv.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		log.Fatal(err)
	}
}
