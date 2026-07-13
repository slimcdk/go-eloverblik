# Go Client for Eloverblik.dk

[![Tests](https://github.com/slimcdk/go-eloverblik/workflows/Tests/badge.svg)](https://github.com/slimcdk/go-eloverblik/actions?query=workflow%3ATests)
[![Go Report Card](https://goreportcard.com/badge/github.com/slimcdk/go-eloverblik)](https://goreportcard.com/report/github.com/slimcdk/go-eloverblik)
[![Go Reference](https://pkg.go.dev/badge/github.com/slimcdk/go-eloverblik.svg)](https://pkg.go.dev/github.com/slimcdk/go-eloverblik)
[![License](https://img.shields.io/github/license/slimcdk/go-eloverblik)](LICENSE)

A comprehensive Go client library and CLI tool for the Danish energy data platform [Eloverblik](https://eloverblik.dk/). Access electricity consumption data, metering points, charges, and more through both the Customer API and Third-Party API.

## Features

- **Complete API Coverage**: Every endpoint both OpenAPI documents declare, for the Customer and the Third-Party API alike (including `getchargelinkswithcharges`, which Energinet has not deployed yet — see [the note](#note-on-charge-links))
- **Data Export**: Export timeseries, masterdata, and charges in CSV or JSON format
- **Rate Limit Aware**: Retries the documented 429 and 503 responses, honouring `Retry-After`
- **Token Introspection**: Read a token's API, roles and expiry without spending a call
- **Debuggable**: `--print-response-headers` shows what the API actually answered
- **Well-Tested**: 87% statement coverage of the library, verified against the live API
- **Multi-Platform**: Cross-compiled binaries for Linux, macOS, and Windows

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
  - [CLI Usage](#cli-usage)
  - [Library Usage](#library-usage)
- [API Coverage](#api-coverage)
- [CLI Reference](#cli-reference)
- [Library Reference](#library-reference)
  - [Dates Are Half-Open](#dates-are-half-open)
  - [Rate Limits and Retries](#rate-limits-and-retries)
  - [Reading Token Claims](#reading-token-claims)
- [Examples](#examples)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

> **Access tokens are not refreshed automatically.** The client exchanges the refresh token
> for a data access token on the first call that needs one and caches it for its own
> lifetime. A data access token lasts about 24 hours, so a long-running process should
> check [the expiry](#reading-token-claims) and build a new client, rather than assume the
> old one keeps working.

## Installation

### Using Go Install

```bash
go install github.com/slimcdk/go-eloverblik@latest
```

### Download Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/slimcdk/go-eloverblik/releases).

### Build from Source

```bash
git clone https://github.com/slimcdk/go-eloverblik.git
cd go-eloverblik
go build .
```

## Quick Start

### Getting Your API Token

1. Visit [Eloverblik.dk](https://eloverblik.dk/)
2. Log in with MitID
3. Go to "Data" → "Datadeling" → "Tredjepartsadgang"
4. Create a new access token (refresh token)

### CLI Usage

```bash
# Set your token as an environment variable
export ELO_TOKEN="your-refresh-token-here"

# See what the token is: which API, which roles, when it expires. No API call.
go-eloverblik token --token=$ELO_TOKEN

# Get your metering points
go-eloverblik --token=$ELO_TOKEN customer installations

# Get time series data. --to is EXCLUSIVE, so this is the whole of January
go-eloverblik --token=$ELO_TOKEN customer timeseries 571313155411053087 \
  --from=2024-01-01 --to=2024-02-01

# Or use a named period, which gets the boundaries right for you
go-eloverblik --token=$ELO_TOKEN customer timeseries 571313155411053087 \
  --period=last_month --aggregation=Day --flatten

# Export data as JSON
go-eloverblik --token=$ELO_TOKEN customer export-charges 571313155411053087 \
  --format=json

# Get charges information
go-eloverblik --token=$ELO_TOKEN customer charges 571313155411053087
```

### Library Usage

```go
package main

import (
    "fmt"
    "log"
    "time"

    eloverblik "github.com/slimcdk/go-eloverblik/v1"
)

func main() {
    // Create a customer client
    client := eloverblik.NewCustomer("your-refresh-token")

    // Get metering points
    meteringPoints, err := client.GetMeteringPoints(true)
    if err != nil {
        log.Fatal(err)
    }

    for _, mp := range meteringPoints {
        fmt.Printf("Metering Point: %s\n", mp.MeteringPointID)
        fmt.Printf("Address: %s %s, %s %s\n",
            mp.StreetName, mp.BuildingNumber, mp.Postcode, mp.CityName)
    }

    // Get time series data. The range is half-open, so this is the whole of January
    from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    to := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

    timeseries, err := client.GetTimeSeries(
        []string{"571313155411053087"},
        from,
        to,
        eloverblik.Hour,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Process the data. Every point carries the interval it covers, not just a timestamp
    for _, ts := range timeseries {
        for _, point := range ts.Flatten() {
            fmt.Printf("%s → %s: %.3f %s\n",
                point.From.Format(time.RFC3339),
                point.To.Format(time.RFC3339),
                point.Measurement,
                point.Unit,
            )
        }
    }
}
```

## API Coverage

### Customer API

| Endpoint | CLI Command | Library Method | Description |
|----------|-------------|----------------|-------------|
| `/api/token` | - | `GetDataAccessToken()` | Get access token |
| `/meteringpoints/meteringpoints` | `customer installations` | `GetMeteringPoints()` | List metering points |
| `/meteringpoints/meteringpoint/getdetails` | `customer details` | `GetMeteringPointDetails()` | Get detailed info |
| `/meterdata/gettimeseries/{from}/{to}/{aggregation}` | `customer timeseries` | `GetTimeSeries()` | Get consumption data |
| `/meteringpoints/meteringpoint/getcharges` | `customer charges` | `GetCustomerCharges()` | Get charges/tariffs |
| `/meteringpoints/meteringpoint/getchargelinkswithcharges` | `customer charge-links` | `GetChargeLinksWithCharges()` | Get charge links with dated prices — **not deployed by Energinet, answers 404** ([note](#note-on-charge-links)) |
| `/meteringpoints/meteringpoint/relation/add` | `customer add-relation` | `AddRelationByID()` | Link metering point |
| `/meteringpoints/meteringpoint/relation/add/{id}/{code}` | `customer add-relation-by-code` | `AddRelationByWebAccessCode()` | Link via code |
| `/meteringpoints/meteringpoint/relation/{id}` | `customer delete-relation` | `DeleteRelation()` | Unlink metering point |
| `/meterdata/export` | `customer export-timeseries` | `ExportTimeSeries()` | Export timeseries |
| `/meteringpoints/masterdata/export` | `customer export-masterdata` | `ExportMasterdata()` | Export masterdata |
| `/meteringpoints/charges/export` | `customer export-charges` | `ExportCharges()` | Export charges |
| `/api/isalive` | `customer alive` | `IsAlive()` | Health check |

### Third-Party API

| Endpoint | CLI Command | Library Method | Description |
|----------|-------------|----------------|-------------|
| `/api/token` | - | `GetDataAccessToken()` | Get access token |
| `/api/authorization/authorization` | `thirdparty authorizations` | `GetAuthorizations()` | List authorizations |
| `/api/meteringpoints/{scope}/{identifier}` | `thirdparty metering-points` | `GetMeteringPointsForScope()` | Get metering points |
| `/api/meteringpoints/meteringpointid/{scope}/{identifier}` | `thirdparty metering-point-ids` | `GetMeteringPointIDsForScope()` | Get IDs only |
| `/meteringpoints/meteringpoint/getdetails` | `thirdparty details` | `GetMeteringPointDetails()` | Get detailed info |
| `/meterdata/gettimeseries/{from}/{to}/{aggregation}` | `thirdparty timeseries` | `GetTimeSeries()` | Get consumption data |
| `/meteringpoints/meteringpoint/getcharges` | `thirdparty charges` | `GetThirdPartyCharges()` | Get charges |
| `/meteringpoint/getchargelinkswithcharges` | `thirdparty charge-links` | `GetChargeLinksWithCharges()` | Get charge links with dated prices — **not deployed by Energinet, answers 404** ([note](#note-on-charge-links)) |
| `/api/isalive` | `thirdparty alive` | `IsAlive()` | Health check |

<a id="note-on-charge-links"></a>

> **Note on `charge-links` / `getchargelinkswithcharges`: Energinet has not deployed this
> endpoint.** Both OpenAPI documents declare it, but the live API answers `404 Not Found`
> for it on **both the Customer API and the Third-Party API**. Checked on **2026-07-13**
> with a valid Customer token and a valid Third-Party token, on every documented path:
>
> ```
> POST /customerapi/api/meteringpoints/meteringpoint/getchargelinkswithcharges  -> 404
> POST /thirdpartyapi/api/meteringpoint/getchargelinkswithcharges               -> 404
> ```
>
> The same tokens got `200 OK` from `getcharges` and `getdetails` in the same session, so
> this is not an authentication or authorization problem — the endpoint simply is not
> served. This client implements it exactly as both specifications describe it and is ready
> for the day Energinet deploys it; until then every call returns a 404 error.
>
> **What to use instead today:** `customer charges` / `thirdparty charges`
> (`GetCustomerCharges` / `GetThirdPartyCharges`). They return the subscriptions, fees and
> tariffs of a metering point — but **only those that are currently valid or take effect in
> the future**, so they cannot price consumption that already happened. That gap is exactly
> what `charge-links` is meant to close, and there is no other endpoint that closes it.

## CLI Reference

### Help Output

Running `go-eloverblik --help` shows a grouped command tree:

```
A CLI for the Danish Eloverblik platform

Usage:
  go-eloverblik [command]

Available Commands:

  customer
    add-relation             Link one or more metering points to the authenticated user by ID
    add-relation-by-code     Link a metering point to the authenticated user via a web access code
    alive                    Check if the API is operational
    charge-links             Get charge links with dated charge prices (Eloverblik has not deployed this endpoint: it answers 404)
    charges                  Get charges (subscriptions, fees, tariffs) for one or more metering points
    delete-relation          Unlink a metering point from the authenticated user
    details                  Get metering point details
    export-charges           Export charges (customer API only)
    export-masterdata        Export metering point masterdata (customer API only)
    export-timeseries        Export time series as a raw stream (customer API only)
    installations            Get metering points (installations)
    timeseries               Get time series for one or more metering points

  thirdparty
    alive                    Check if the API is operational
    authorizations           Get authorizations (powers of attorney) granted by customers
    charge-links             Get charge links with dated charge prices (Eloverblik has not deployed this endpoint: it answers 404)
    charges                  Get charges (subscriptions, tariffs) for one or more metering points
    details                  Get metering point details
    metering-point-ids       Get metering point IDs accessible under a specific authorization scope
    metering-points          Get metering points accessible under a specific authorization scope
    timeseries               Get time series for one or more metering points

  token                      Show what the Eloverblik token says about itself

Flags:
  -h, --help                     help for go-eloverblik
      --print-response-headers   Print HTTP response headers from the Eloverblik API to stderr
      --token string             Eloverblik refresh token (required)

Use "go-eloverblik [command] --help" for more information about a command.
```

`charge-links` is registered on both `customer` and `thirdparty`, and both currently fail:
Energinet has not deployed `getchargelinkswithcharges` on either API. See
[the note above](#note-on-charge-links).

### Global Flags

```
--token string               Eloverblik API refresh token (required)
--print-response-headers     Print HTTP response headers from the Eloverblik API to stderr
```

`--print-response-headers` is a debugging aid. The headers of every API call, including
the token call, are written to stderr, so stdout stays clean, parseable output:

```bash
go-eloverblik customer details <metering-id> --token=$TOKEN --print-response-headers 2>headers.txt
```

```
< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Mon, 01 Jan 2024 00:00:00 GMT
```

### Inspecting a Token

`token` decodes the claims of the token in `--token` and makes no request, which answers
the questions that otherwise cost a failed call: which API is this token for, which roles
does it carry, and has it expired?

```bash
go-eloverblik token --token=$TOKEN
```

```json
{
  "tokenType": "THIRDPARTYAPI_Refresh",
  "tokenName": "christian local testing",
  "name": "Christian Silas Skjerning",
  "company": "Styr paa ApS",
  "cvr": "44341603",
  "roles": ["ReadPrivate", "ReadBusiness"],
  "expiresAt": "2027-02-10T13:33:10+01:00"
}
```

Add `--data-access` to exchange the refresh token for a short lived data access token and
decode that one instead. That does make a request, and the client to use is taken from the
token itself:

```bash
go-eloverblik token --data-access --token=$TOKEN
```

### Customer Commands

```bash
# Installation Management
go-eloverblik customer installations                    # List all metering points
go-eloverblik customer details <metering-id>...         # Get detailed information

# Relations
go-eloverblik customer add-relation <metering-id>...    # Add relation by ID
go-eloverblik customer add-relation-by-code <id> <code> # Add relation by web code
go-eloverblik customer delete-relation <metering-id>    # Remove relation

# Data Retrieval
go-eloverblik customer timeseries <metering-id>... --period=last_month
go-eloverblik customer timeseries <metering-id>... --from=now-30d --to=now

# Use --from/--to for specific ranges:
#   YYYY-MM-DD
#   now
#   now-30d (days), now-4w (weeks), now-2m (months), now-1y (years)
#
# Use --period for common ranges (cannot be used with --from/--to):
#   yesterday, this_week, last_week, this_month, last_month,
#   this_year, last_year

go-eloverblik customer timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --aggregation=Hour \                         # Actual, Quarter, Hour, Day, Month, Year
  --flatten                                    # Simplify output

go-eloverblik customer charges <metering-id>...         # Get charges and tariffs
# NOTE: 'charges' only returns charges that are currently valid or take effect in the
# future. It cannot price consumption that already happened.

# Charge links with the dated price series of every linked charge.
# NOT AVAILABLE: Energinet has not deployed getchargelinkswithcharges. Checked 2026-07-13
# with a valid customer token, the Customer API answered 404 while 'charges' answered 200.
# The command implements the endpoint as specified and is ready for the day it is deployed;
# today it returns a 404 error. Until then, 'charges' above is the closest data available.
go-eloverblik customer charge-links <metering-id>... --period=last_month
go-eloverblik customer charge-links <metering-id>... --from=YYYY-MM-DD --to=YYYY-MM-DD

# Data Export (CSV or JSON)
go-eloverblik customer export-timeseries <metering-id>... --period=last_year

go-eloverblik customer export-timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --format=json                                # csv (default) or json

go-eloverblik customer export-masterdata <metering-id>... \
  --format=json

go-eloverblik customer export-charges <metering-id>... \
  --format=json

# Health Check
go-eloverblik customer alive                            # Check API status
```

### Third-Party Commands

```bash
# Authorization Management
go-eloverblik thirdparty authorizations                 # List all authorizations

# Metering Points
go-eloverblik thirdparty metering-points <scope> <identifier>
  # Scope: authorizationid, customercvr, customerkey

go-eloverblik thirdparty metering-point-ids <scope> <identifier>
  # Get IDs only (faster)

# Data Retrieval
go-eloverblik thirdparty details <metering-id>...
go-eloverblik thirdparty timeseries <metering-id>... --period=last_week
go-eloverblik thirdparty timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --aggregation=Hour

go-eloverblik thirdparty charges <metering-id>...
# NOTE: 'charges' only returns charges that are currently valid or take effect in the
# future. It cannot price consumption that already happened.

# Charge links with the dated price series of every linked charge.
# NOT AVAILABLE: Energinet has not deployed getchargelinkswithcharges. Checked 2026-07-13
# with a valid third-party token, the Third-Party API answered 404 while 'charges' answered
# 200 — exactly as the Customer API did. The command implements the endpoint as specified
# and is ready for the day it is deployed; today it returns a 404 error.
go-eloverblik thirdparty charge-links <metering-id>... --period=last_month
go-eloverblik thirdparty charge-links <metering-id>... --from=YYYY-MM-DD --to=YYYY-MM-DD

# Health Check
go-eloverblik thirdparty alive
```

## Library Reference

### Creating Clients

```go
import eloverblik "github.com/slimcdk/go-eloverblik/v1"

// Customer API client
customer := eloverblik.NewCustomer("refresh-token")

// Third-Party API client
thirdparty := eloverblik.NewThirdParty("refresh-token")
```

Both constructors accept optional options.

### Debugging Response Headers

`WithResponseHeaderOutput` writes the HTTP response headers of every API call, including
the token call and the streamed exports, to the given `io.Writer`:

```go
customer := eloverblik.NewCustomer("refresh-token", eloverblik.WithResponseHeaderOutput(os.Stderr))
```

```
< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Mon, 01 Jan 2024 00:00:00 GMT
```

### Reading Token Claims

Both Eloverblik tokens are JWTs. `ParseToken` decodes the claims of any of them, and the
client can read its own:

```go
claims, err := eloverblik.ParseToken(refreshToken)

claims.TokenName    // the name given to the token in the portal
claims.Roles        // []string{"ReadPrivate", "ReadBusiness"}
claims.Company      // "Styr paa ApS"
claims.ExpiresAt    // time.Time, in Copenhagen time
claims.IsExpired()  // no request needed to find out
claims.APIType()    // eloverblik.ThirdPartyApi, taken from the token itself

customer := eloverblik.NewCustomer(refreshToken)
claims, err = customer.RefreshTokenClaims()     // no request
claims, err = customer.DataAccessTokenClaims()  // fetches a data access token first
```

The claims are decoded, not verified: only Energinet holds the signing key, so a token can
still only be authenticated by using it. Read the claims to tell tokens apart, to check an
expiry before a batch job, or to see which roles a token was granted — not as a security
check.

### Pricing Historic Consumption

`GetCustomerCharges` and `GetThirdPartyCharges` only return charges that are currently
valid or take effect in the future, so they cannot price consumption that already
happened. `GetChargeLinksWithCharges` is the endpoint that returns the missing half: the
dated price series of every charge a metering point is linked to, along with the charge
link periods and their factors, the VAT classification and the tax indicator.

> **Energinet has not deployed it — on either API.** Both OpenAPI documents declare
> `getchargelinkswithcharges`, but the live API answers `404 Not Found` for it on the
> **Customer API and the Third-Party API alike**. Checked on **2026-07-13** with a valid
> Customer token and a valid Third-Party token, on every documented path, in a session
> where `getcharges` returned `200 OK` for the same tokens — so it is not an auth problem,
> the route is simply not served. This client implements the endpoint exactly as both
> specifications describe it and is ready for the day Energinet deploys it. Until then,
> `GetChargeLinksWithCharges` returns a 404 error on both clients, and **there is no way to
> price historic consumption through this API**: `GetCustomerCharges` /
> `GetThirdPartyCharges` (the `charges` commands) are the closest available data, and they
> only carry present and future prices.

The code below is what the endpoint will return once it is deployed — it is included so
you can see the shape of the data, not because it works today.

```go
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    log.Fatal(err)
}

// client is either a Customer or a ThirdParty client
links, err := client.GetChargeLinksWithCharges([]string{"571313180400000001"}, from, to)
if err != nil {
    log.Fatal(err)
}

// The charges are returned once, next to the links, keyed by their charge identifier
charges := make(map[eloverblik.ChargeIdentifier]eloverblik.ChargeInformation, len(links.ChargeInformations))
for _, info := range links.ChargeInformations {
    charges[info.ChargeIdentifier] = info
}

for _, result := range links.Results {
    if result.Error != "" { // errors are reported per metering point
        log.Printf("%s: %s", result.MeteringPointID, result.Error)
        continue
    }
    for _, link := range result.ChargeLinks {
        info := charges[link.ChargeIdentifier]
        for _, point := range info.ChargeSeriesPoints {
            fmt.Printf("%s  %s  %.4f DKK  tax=%t\n",
                link.ChargeIdentifier.Code, point.From.Format(time.RFC3339), point.Price, info.TaxIndicator)
        }
    }
}
```

Multiply a price point by the consumption in the same interval and by the `Factor` of the
charge link period covering it to get the amount charged.

### Aggregation Levels

The aggregation you ask for:

```go
eloverblik.Actual   // Raw meter readings
eloverblik.Quarter  // 15-minute aggregation
eloverblik.Hour     // Hourly aggregation
eloverblik.Day      // Daily aggregation
eloverblik.Month    // Monthly aggregation
eloverblik.Year     // Yearly aggregation
```

The resolution the API answers with is a different vocabulary, and it is worth knowing
before you parse a response yourself:

| Aggregation | `resolution` on the wire | Shape of the response |
|---|---|---|
| `Quarter` | `PT15M` | one period per day, 96 points |
| `Hour` | `PT1H` | one period per day, 24 points |
| `Day` | `PT1D` | one period per day, a single point |
| `Month` | `P1M` | one period per month, a single point |
| `Year` | `PT1Y` | one period per year, a single point, and it may be **partial** |

Two traps here. Eloverblik's OpenAPI document says the day and year resolutions are `P1D`
and `P1Y`; the live API sends `PT1D` and `PT1Y`. This client accepts both. And a `Year`
period can cover only part of a year — 27 April to 31 December, say — so `Flatten()` takes
the interval the API states for single-point periods instead of assuming a full calendar
year. A metering point read hourly answers `PT1H` even when you ask for `Quarter`.

### Authorization Scopes (Third-Party API)

```go
eloverblik.AuthScopeID           // Scope by authorization ID
eloverblik.AuthScopeCustomerCVR  // Scope by customer CVR number
eloverblik.AuthScopeCustomerKey  // Scope by customer key
```

### Dates Are Half-Open

The API reads a requested range as `[dateFrom, dateTo)`, at the granularity of a date:
`dateFrom` is included, `dateTo` is not, and a request where the two are equal is rejected
outright with error 30002.

```go
// Asking for 1 July through 4 July returns 1, 2 and 3 July — three days, not four.
from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
to := time.Date(2026, 7, 4, 0, 0, 0, 0, time.Local)
timeseries, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)
```

So to get a single day, ask for that day and the next one. This is the off-by-one that
makes `--period yesterday` sound like it should send the same date twice; it must not.

### Period Constants

The `Period` type and `GetDatesFromPeriod` cover the common ranges and already return an
exclusive `to`, so a period never drops its own last day.

```go
import eloverblik "github.com/slimcdk/go-eloverblik/v1"

// Last month, in full: from is the 1st of last month, to is the 1st of this month
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    // handle error
}
timeseries, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)

// Available Period constants:
// Yesterday, ThisWeek, LastWeek, ThisMonth, LastMonth,
// ThisYear, LastYear
```

The API only holds time series for the previous five years plus the current one, and
refuses a `to` in the future (error 30003), so today's consumption is never available.

### Rate Limits and Retries

Eloverblik limits a single IP to **2 token calls per minute** and **120 calls per minute**
in total, and answers `429` when you exceed it. It answers `503` when DataHub itself is
overloaded. Both are transient, and the client retries them by default — twice, honouring
the `Retry-After` header when the API sends one, waiting at most 60 seconds. Nothing else
is retried: a `401` or a `404` is a real answer and is returned to you immediately.

```go
// The defaults, spelled out
client := eloverblik.NewCustomer(refreshToken,
    eloverblik.WithRetry(eloverblik.DefaultRetryCount, eloverblik.DefaultRetryMaxWait))

// Fail fast instead — useful in a request handler that cannot afford to block
client = eloverblik.NewCustomer(refreshToken, eloverblik.WithoutRetry())
```

The API's own advice is to ask for **at most 10 metering points per request**, which is
also what the CLI enforces.

## Examples

### Fetch Hourly Data for the Past 7 Days

```go
package main

import (
    "fmt"
    "log"
    "os"
    "time"

    eloverblik "github.com/slimcdk/go-eloverblik/v1"
)

func main() {
    client := eloverblik.NewCustomer(os.Getenv("ELO_TOKEN"))

    from := time.Now().AddDate(0, 0, -7)
    to := time.Now()

    ts, err := client.GetTimeSeries(
        []string{"571313155411053087"},
        from, to,
        eloverblik.Hour,
    )
    if err != nil {
        log.Fatal(err)
    }

    for _, series := range ts {
        for _, point := range series.Flatten() {
            fmt.Printf("%s  %.3f %s\n",
                point.From.Format("2006-01-02 15:04"),
                point.Measurement,
                point.Unit,
            )
        }
    }
}
```

Or using a predefined period:

```go
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastWeek)
if err != nil {
    log.Fatal(err)
}
ts, err := client.GetTimeSeries(ids, from, to, eloverblik.Hour)
```

### Export Data to CSV File

```bash
go-eloverblik --token=$ELO_TOKEN customer export-timeseries 571313155411053087 \
  --from=2024-01-01 --to=2024-12-31 > consumption_2024.csv
```

### Export Data to JSON File

```bash
go-eloverblik --token=$ELO_TOKEN customer export-charges 571313155411053087 \
  --format=json > charges.json
```

### Process Multiple Metering Points

```go
meteringPoints := []string{
    "571313155411053087",
    "571313155411782079",
}

details, err := client.GetMeteringPointDetails(meteringPoints)
if err != nil {
    log.Fatal(err)
}

for _, detail := range details {
    if !detail.Success {
        log.Printf("Error for %s: %s", detail.ID, detail.ErrorText)
        continue
    }
    fmt.Printf("Grid Operator: %s\n", detail.Result.GridOperatorName)
}
```

### Flatten Time Series Data

```go
timeseries, err := client.GetTimeSeries(
    meteringPoints,
    from, to,
    eloverblik.Hour,
)

for _, ts := range timeseries {
    // Flatten the nested market document into one record per measured interval
    points := ts.Flatten()

    for _, point := range points {
        fmt.Printf("%s → %s: %.3f %s (Quality: %s, Resolution: %s)\n",
            point.From.Format(time.RFC3339),
            point.To.Format(time.RFC3339),
            point.Measurement,
            point.Unit,
            point.Quality,
            point.Resolution,
        )
    }
}
```

### Third-Party Authorization Flow

```go
client := eloverblik.NewThirdParty("refresh-token")

// Get all authorizations
auths, err := client.GetAuthorizations()
if err != nil {
    log.Fatal(err)
}

for _, auth := range auths {
    fmt.Printf("Customer: %s (Key: %s)\n",
        auth.CustomerName,
        auth.CustomerKey,
    )

    // Get metering points for this authorization
    meteringPoints, err := client.GetMeteringPointsForScope(
        eloverblik.AuthScopeCustomerKey,
        auth.CustomerKey,
    )
    if err != nil {
        log.Printf("Error: %v", err)
        continue
    }

    for _, mp := range meteringPoints {
        fmt.Printf("  - %s\n", mp.MeteringPointID)
    }
}
```

### Check a Token Before a Batch Job

Nothing is more annoying than a nightly job that dies at 03:00 because a refresh token
expired. The claims answer that without spending an API call:

```go
claims, err := eloverblik.ParseToken(refreshToken)
if err != nil {
    log.Fatalf("that is not an Eloverblik token: %v", err)
}

if claims.IsExpired() {
    log.Fatalf("token %q expired at %s — generate a new one in the portal",
        claims.TokenName, claims.ExpiresAt.Format(time.RFC1123))
}
if claims.ExpiresIn() < 7*24*time.Hour {
    log.Printf("warning: token %q expires in %s", claims.TokenName, claims.ExpiresIn())
}

// The token knows which API it belongs to, so the caller does not have to say it twice
apiType, err := claims.APIType()
if err != nil {
    log.Fatal(err)
}

var client eloverblik.Client
if apiType == eloverblik.ThirdPartyApi {
    client = eloverblik.NewThirdParty(refreshToken)
} else {
    client = eloverblik.NewCustomer(refreshToken)
}
```

From the CLI, the same thing:

```bash
go-eloverblik token --token=$TOKEN | jq '{tokenName, roles, expiresAt}'
```

### Third-Party: From Authorization to Consumption

The full path a third party walks: list the powers of attorney customers have granted, take
the metering points of one of them, and read yesterday's consumption.

```go
client := eloverblik.NewThirdParty(refreshToken)

authorizations, err := client.GetAuthorizations()
if err != nil {
    log.Fatal(err)
}

for _, auth := range authorizations {
    log.Printf("%s (CVR %s), valid until %s", auth.CustomerName, auth.CustomerCVR, auth.ValidTo)

    ids, err := client.GetMeteringPointIDsForScope(eloverblik.AuthScopeID, auth.ID)
    if err != nil {
        log.Printf("  %v", err)
        continue
    }

    // The API asks for at most 10 metering points per request
    for _, batch := range chunk(ids, 10) {
        from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.Yesterday)
        if err != nil {
            log.Fatal(err)
        }

        timeseries, err := client.GetTimeSeries(batch, from, to, eloverblik.Hour)
        if err != nil {
            log.Printf("  %v", err)
            continue
        }

        for _, ts := range timeseries {
            for _, point := range ts.Flatten() {
                fmt.Printf("  %s  %s → %s  %.2f %s\n",
                    ts.MyEnergyDataMarketDocument.TimeSeries[0].MRID,
                    point.From.Format(time.RFC3339), point.To.Format(time.RFC3339),
                    point.Measurement, point.Unit)
            }
        }
    }
}
```

Note the batching. A third party with a few hundred metering points will otherwise walk
straight into the 120 calls per minute limit; the client retries the resulting 429, but
not making the call at all is faster.

### Debug a Call That Fails

When the API says no and you want to know what it really answered, print the response
headers. They go to stderr, so stdout stays a clean JSON stream you can still pipe:

```bash
go-eloverblik customer timeseries 571313180400000000 \
    --period=yesterday --aggregation=Hour \
    --token=$TOKEN --print-response-headers 2>headers.txt | jq .

cat headers.txt
```

```
< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
< Api-Supported-Versions: 1.0
< Content-Type: application/json; charset=utf-8
< Date: Mon, 13 Jul 2026 00:24:46 GMT
```

In the library, the same switch is an option, and it covers the token call and the
streamed exports too:

```go
client := eloverblik.NewCustomer(refreshToken,
    eloverblik.WithResponseHeaderOutput(os.Stderr))
```

### Error Handling

Failures arrive at two levels, and both matter.

**The call itself** fails with a sentinel error you can match on:

```go
timeseries, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)
switch {
case errors.Is(err, eloverblik.ErrorUnauthorized):
    // the refresh token is wrong or has expired — a new one must be generated
case errors.Is(err, eloverblik.ErrorTooManyRequests):
    // still rate limited after the retries; back off for a minute
case errors.Is(err, eloverblik.ErrorNoCprConsent):
    // GetMeteringPoints(true) needs CPR consent, granted once in the portal
case errors.Is(err, eloverblik.ErrorToDateCanNotBeEqualToFromDate):
    // the range is half-open: ask for the day AND the next one
case err != nil:
    log.Fatal(err)
}
```

**Individual metering points** can fail inside an otherwise successful response — one
missing relation does not fail the batch, it fails that item:

```go
details, err := client.GetMeteringPointDetails(meteringPoints)
if err != nil {
    log.Fatal(err)
}

for _, detail := range details {
    if !detail.Success {
        // e.g. 20010 RelationNotFound: this metering point is not linked to the token
        log.Printf("%s: error %d: %s", detail.ID, detail.ErrorCode, detail.ErrorText)
        continue
    }

    processDetail(detail.Result)
}
```

A 429 or a 503 is retried for you (twice, honouring `Retry-After`) before it ever becomes
an error. Everything else — a 401, a 404, a rejected date range — is returned straight
away, because retrying it would only waste a call against the rate limit.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./v1
go test ./cmd
```

### Running Linter

The config is in the golangci-lint **v2** format, so v1 will not read it. CI pins v2.12;
match it locally:

```bash
# Install golangci-lint v2
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# Run linter — the same command CI runs
golangci-lint run --timeout=5m
```

### Building

```bash
# Build for current platform
go build .

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o go-eloverblik-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o go-eloverblik-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o go-eloverblik-windows-amd64.exe .
```

## Project Structure

```
.
├── cmd/                    # CLI command implementations
│   ├── measurements.go     # Timeseries and export commands
│   ├── charges.go          # Charges commands
│   ├── customer.go         # Customer-specific commands
│   ├── thirdparty.go       # Third-party specific commands
│   └── root.go             # Root command and initialization
├── v1/                     # Library implementation
│   ├── auth.go             # Authentication
│   ├── charges.go          # Charges endpoints
│   ├── eloverblik.go       # Client initialization
│   ├── errors.go           # Error handling
│   ├── interfaces.go       # API interfaces
│   ├── meters.go           # Metering point endpoints
│   ├── models.go           # Data models
│   ├── relations.go        # Relations endpoints
│   ├── timeseries.go       # Timeseries endpoints
│   └── *_test.go           # Unit tests
├── .github/
│   └── workflows/
│       └── test.yml        # CI/CD pipeline
├── .golangci.yml           # Linter configuration
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── main.go                 # CLI entry point
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Run linter (`golangci-lint run`)
6. Commit your changes (`git commit -m 'Add some amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Guidelines

- Write tests for new features
- Maintain or improve test coverage
- Follow existing code style
- Update documentation for API changes
- Add examples for new functionality

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Eloverblik.dk](https://eloverblik.dk/) for providing the API
- [Energinet](https://energinet.dk/) for the energy data platform
- All contributors who have helped improve this project

## Support

- [API Documentation](https://api.eloverblik.dk/customerapi/index.html)
- [Report Issues](https://github.com/slimcdk/go-eloverblik/issues)
- [Discussions](https://github.com/slimcdk/go-eloverblik/discussions)

## Related Projects

- [Eloverblik API Documentation](https://api.eloverblik.dk/)
- [Eloverblik Portal](https://eloverblik.dk/)

---

Made for the Danish energy community
