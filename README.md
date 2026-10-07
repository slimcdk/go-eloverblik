# Go Client for Eloverblik.dk

[![Tests](https://github.com/slimcdk/go-eloverblik/workflows/Tests/badge.svg)](https://github.com/slimcdk/go-eloverblik/actions?query=workflow%3ATests)
[![Go Report Card](https://goreportcard.com/badge/github.com/slimcdk/go-eloverblik)](https://goreportcard.com/report/github.com/slimcdk/go-eloverblik)
[![Go Reference](https://pkg.go.dev/badge/github.com/slimcdk/go-eloverblik/v1.svg)](https://pkg.go.dev/github.com/slimcdk/go-eloverblik/v1)
[![License](https://img.shields.io/github/license/slimcdk/go-eloverblik)](LICENSE)

A comprehensive Go client library and CLI tool for the Danish energy data platform [Eloverblik](https://eloverblik.dk/). Access electricity consumption data, metering points, charges, and more through both the Customer API and Third-Party API.

## Features

- **Complete API Coverage**: Every endpoint both OpenAPI documents declare, for the Customer and the Third-Party API alike (including `getchargelinkswithcharges`, which Energinet has not switched on yet — see [the note](#note-on-charge-links))
- **Data Export**: Export timeseries, masterdata, and charges as the CSV files the API generates; the CLI can convert them to JSON
- **Rate Limit Aware**: Retries the documented 429 and 503 responses, honouring `Retry-After`
- **Token Renewal**: Fetches the data access token on first use, caches it and renews it before it expires; one client is safe to share between goroutines
- **Token Introspection**: Read a token's API, roles and expiry without spending a call
- **Debuggable**: `--print-response-headers` shows what the API actually answered
- **Well-Tested**: 93% statement coverage of the library, with test fixtures taken from live API responses
- **Multi-Platform**: Binaries for Linux (x86-64, ARM64 and 32-bit ARM, Raspberry Pi included), macOS and Windows (x86-64 and ARM64), tested on all three operating systems and as 32-bit code

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
  - [CLI Usage](#cli-usage)
  - [Library Usage](#library-usage)
- [API Coverage](#api-coverage)
  - [DataHub 3.0](#datahub-3)
- [CLI Reference](#cli-reference)
- [Library Reference](#library-reference)
  - [Dates Are Half-Open](#dates-are-half-open)
  - [Rate Limits and Retries](#rate-limits-and-retries)
  - [Reading Token Claims](#reading-token-claims)
- [Examples](#examples)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

> **For integrators and AI agents:** [`llms.md`](llms.md) is the full reference for the
> library and the CLI in one file. The package documentation is on
> [pkg.go.dev](https://pkg.go.dev/github.com/slimcdk/go-eloverblik/v1), and
> `go-eloverblik --help` states the rules every command shares, while each command's own
> `--help` gives its endpoint, arguments, output and examples.

## Installation

### Using Go Install

```bash
go install github.com/slimcdk/go-eloverblik@latest
```

### Download Pre-built Binaries

Download the archive for your platform and `checksums.txt` from
[GitHub Releases](https://github.com/slimcdk/go-eloverblik/releases). Each archive holds
the `go-eloverblik` binary plus `LICENSE` and `README.md`. The binary needs no Go
installation and brings its own time zone database.

| Platform | Archive |
|----------|---------|
| Linux on x86-64 | `linux_amd64` |
| Linux on 64-bit ARM, including a Raspberry Pi on a 64-bit system (`uname -m` prints `aarch64`) | `linux_arm64` |
| Linux on 32-bit ARM: a Raspberry Pi on a 32-bit system, from the Pi Zero and Pi 1 up (`uname -m` prints `armv6l` or `armv7l`) | `linux_arm` |
| macOS on Apple silicon / on Intel, macOS 13 Ventura or later | `darwin_arm64` / `darwin_amd64` |
| Windows on x86-64 / on Arm, Windows 10 or later | `windows_amd64` / `windows_arm64` |

#### Linux and Raspberry Pi

```bash
grep '_linux_arm.tar.gz$' checksums.txt | sha256sum -c -
tar -xzf go-eloverblik_<version>_linux_arm.tar.gz
sudo install go-eloverblik /usr/local/bin/
```

In a minimal container image the binary runs as is, but HTTPS calls need CA certificates.
`alpine` and `gcr.io/distroless/static` ship them; `debian:*-slim`, `busybox` and `scratch`
do not. Install `ca-certificates`, copy `/etc/ssl/certs/ca-certificates.crt` from your
build stage, or point `SSL_CERT_FILE` at a bundle. Without them every API call fails with
`x509: certificate signed by unknown authority`.

#### macOS

```bash
grep '_darwin_arm64.tar.gz$' checksums.txt | shasum -a 256 -c -
tar -xzf go-eloverblik_<version>_darwin_arm64.tar.gz
xattr -d com.apple.quarantine go-eloverblik  # only if a browser downloaded the archive
sudo mkdir -p /usr/local/bin && sudo mv go-eloverblik /usr/local/bin/
```

The binaries are not signed with an Apple Developer ID or notarized. A browser marks the
download as quarantined, and macOS then blocks the first run with "go-eloverblik Not
Opened" or "cannot be opened because the developer cannot be verified". The `xattr` line
removes that mark; System Settings > Privacy & Security > Open Anyway does the same once.
Downloading with `curl -LO`, or installing with `go install`, sets no mark.

#### Windows

In PowerShell:

```powershell
Get-FileHash -Algorithm SHA256 go-eloverblik_<version>_windows_amd64.zip  # compare with checksums.txt
Expand-Archive go-eloverblik_<version>_windows_amd64.zip "$env:LOCALAPPDATA\Programs\go-eloverblik"
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$userPath;$env:LOCALAPPDATA\Programs\go-eloverblik", "User")
```

Open a new terminal so it picks up the `PATH` change. The shell examples in this README
are written for bash. In PowerShell, set and pass the token like this, and continue a long
command on the next line with a backtick (`` ` ``) instead of `\`:

```powershell
$env:ELO_TOKEN = "your-refresh-token-here"
go-eloverblik token --token $env:ELO_TOKEN
```

In cmd it is `set ELO_TOKEN=your-refresh-token-here`, then `--token %ELO_TOKEN%`, and `^`
continues a line.

### Build from Source

```bash
git clone https://github.com/slimcdk/go-eloverblik.git
cd go-eloverblik
go build .
```

## Quick Start

### Getting Your API Token

1. Visit [Eloverblik.dk](https://eloverblik.dk/)
2. Log in with MitID (MitID Erhverv for a Third-Party API token)
3. Choose "API-adgang" in the menu
4. Click "Opret token" for the Customer API (allow use of your CPR number the first time,
   and name the token), or "Opret refresh token" for the Third-Party API (valid for 1 year)
5. Copy the token: this is your refresh token

### CLI Usage

```bash
# Set your token as an environment variable
export ELO_TOKEN="your-refresh-token-here"

# See what the token is: which API, which roles, when it expires. No API call.
go-eloverblik token --token=$ELO_TOKEN

# Get your metering points
go-eloverblik --token=$ELO_TOKEN customer installations

# Get time series data. --to is EXCLUSIVE, so this is the whole of January
go-eloverblik --token=$ELO_TOKEN customer timeseries 571313000000000001 \
  --from=2024-01-01 --to=2024-02-01

# Or use a named period, which gets the boundaries right for you
go-eloverblik --token=$ELO_TOKEN customer timeseries 571313000000000001 \
  --period=last_month --aggregation=Day --flatten

# Export data as JSON, converted by the CLI from the CSV the API returns
go-eloverblik --token=$ELO_TOKEN customer export-charges 571313000000000001 \
  --format=json

# Get charges information
go-eloverblik --token=$ELO_TOKEN customer charges 571313000000000001
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
        []string{"571313000000000001"},
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
| `/api/token` | `token --data-access` | `GetDataAccessToken()` | Get access token (the CLI decodes its claims, it does not print the token) |
| `/meteringpoints/meteringpoints` | `customer installations` | `GetMeteringPoints()` | List metering points |
| `/meteringpoints/meteringpoint/getdetails` | `customer details` | `GetMeteringPointDetails()` | Get detailed info |
| `/meterdata/gettimeseries/{from}/{to}/{aggregation}` | `customer timeseries` | `GetTimeSeries()` | Get consumption data |
| `/meteringpoints/meteringpoint/getcharges` | `customer charges` | `GetCustomerCharges()` | Get charges/tariffs |
| `/meteringpoints/meteringpoint/getchargelinkswithcharges` | `customer charge-links` | `GetChargeLinksWithCharges()` | Get charge links with dated prices — **not enabled by Energinet, answers 404** ([note](#note-on-charge-links)) |
| `/meteringpoints/meteringpoint/relation/add` | `customer add-relation` | `AddRelationByID()` | Link metering point |
| `/meteringpoints/meteringpoint/relation/add/{id}/{code}` | `customer add-relation-by-code` | `AddRelationByWebAccessCode()` | Link via code — **retired by Energinet with DataHub 3.0, answers 410** ([note](#datahub-3)) |
| `/meteringpoints/meteringpoint/relation/{id}` | `customer delete-relation` | `DeleteRelation()` | Unlink metering point — **retired by Energinet with DataHub 3.0, answers 410** ([note](#datahub-3)) |
| `/meterdata/timeseries/export/{from}/{to}/{aggregation}` | `customer export-timeseries` | `ExportTimeSeries()` | Export timeseries |
| `/meteringpoints/masterdata/export` | `customer export-masterdata` | `ExportMasterdata()` | Export masterdata |
| `/meteringpoints/charges/export` | `customer export-charges` | `ExportCharges()` | Export charges |
| `/api/isalive` | `customer alive` | `IsAlive()` | Health check |

### Third-Party API

| Endpoint | CLI Command | Library Method | Description |
|----------|-------------|----------------|-------------|
| `/api/token` | `token --data-access` | `GetDataAccessToken()` | Get access token (the CLI decodes its claims, it does not print the token) |
| `/authorization/authorizations` | `thirdparty authorizations` | `GetAuthorizations()` | List authorizations |
| `/authorization/authorization/meteringpoints/{scope}/{identifier}` | `thirdparty metering-points` | `GetMeteringPointsForScope()` | Get metering points |
| `/authorization/authorization/meteringpointids/{scope}/{identifier}` | `thirdparty metering-point-ids` | `GetMeteringPointIDsForScope()` | Get IDs only |
| `/meteringpoint/getdetails` | `thirdparty details` | `GetMeteringPointDetails()` | Get detailed info |
| `/meterdata/gettimeseries/{from}/{to}/{aggregation}` | `thirdparty timeseries` | `GetTimeSeries()` | Get consumption data |
| `/meteringpoint/getcharges` | `thirdparty charges` | `GetThirdPartyCharges()` | Get charges |
| `/meteringpoint/getchargelinkswithcharges` | `thirdparty charge-links` | `GetChargeLinksWithCharges()` | Get charge links with dated prices — **not enabled by Energinet, answers 404** ([note](#note-on-charge-links)) |
| `/api/isalive` | `thirdparty alive` | `IsAlive()` | Health check |

<a id="note-on-charge-links"></a>

> **Note on `charge-links` / `getchargelinkswithcharges`: Energinet has not switched this
> endpoint on.** Both OpenAPI documents declare it and document a `404` for it: "When the
> Charges integration feature is disabled". That is what the live API answers on **both the
> Customer API and the Third-Party API**. Checked on **2026-07-13** with a valid Customer
> token and a valid Third-Party token, on every documented path:
>
> ```
> POST /customerapi/api/meteringpoints/meteringpoint/getchargelinkswithcharges  -> 404
> POST /thirdpartyapi/api/meteringpoint/getchargelinkswithcharges               -> 404
> ```
>
> The same tokens got `200 OK` from `getcharges` and `getdetails` in the same session, so
> this is not an authentication or authorization problem — the feature is switched off.
> This client implements the request and response as both specifications describe them,
> except that one call applies the same interval to every metering point, and is ready for
> the day Energinet enables it; until then every call returns a 404 error.
>
> **What to use instead today:** `customer charges` / `thirdparty charges`
> (`GetCustomerCharges` / `GetThirdPartyCharges`). They return the subscriptions and
> tariffs of a metering point (the Customer API adds its fees) — but **only those that are
> currently valid or take effect in the future**, so they cannot price consumption that
> already happened. That gap is exactly what `charge-links` is meant to close, and there is
> no other endpoint that closes it.

<a id="datahub-3"></a>

### DataHub 3.0

Energinet put DataHub 3.0 into operation on 18 September 2026. The Eloverblik API kept its
endpoints and its `api-version` of 1.0, but what it answers changed in places:

- **Two Customer API endpoints are retired** and answer `410 Gone`: linking a metering point
  with a web access code, and deleting a relation. The client reports both as
  `ErrorEndpointRetired`. Web access codes are replaced by
  [data sharing](https://docs.eloverblik.dk/docs/guides/data-sharing) in ElOverblik, which
  has no API. `AddRelationByWebAccessCode` and `DeleteRelation` are deprecated; the
  `add-relation-by-code` and `delete-relation` commands are hidden and no longer call the API.
- **A metering point can fail on its own** inside a successful time series response: 30015
  when there is no data, 30016 when the relation has expired, 30018 when the period lies
  outside the metering point's data, e.g. because it starts before the metering point was
  registered in DataHub. Check every result with `Err()`, see
  [Error Handling](#error-handling).
- **The same metering point can come back more than once**, once per access period, when
  Energinet enables more than the latest one. Do not key results by metering point ID alone.
- **Several master data fields are retired or unavailable** according to Energinet's
  [field descriptions](https://docs.eloverblik.dk/docs/guides/metering-point-data-field-descriptions),
  among them the settlement method, the consumer category, the meter reading occurrence and
  the estimated annual volume, and, for now, the consumer, balance supplier and tax reduction
  start dates. Expect them to be empty.
- The metering point list gained `isMovedOut` (`IsMovedOut`), and the type of metering point
  gained the value `D19` (capacity settlement).

## CLI Reference

### Help Output

Running `go-eloverblik --help` prints what the tool is, the rules every command shares,
examples and a grouped command tree:

```
A CLI for the Danish Eloverblik platform: electricity metering data from Energinet's
DataHub, read through Eloverblik's two APIs at api.eloverblik.dk.

  customer     The Customer API: the metering points of the person or company the
               refresh token belongs to.
  thirdparty   The Third-Party API: the metering points customers have authorized a
               third party to read, through powers of attorney.
  token        Decode the token given with --token, e.g. to see which API it is for.

Authentication
  Pass the refresh token created at eloverblik.dk with --token, e.g. --token "$ELO_TOKEN".
  Every customer, thirdparty and token command requires it. A token works with the API
  it was issued for only; "go-eloverblik token" prints its tokenType, which names it.
  The CLI exchanges the refresh token for a short lived data access token itself
  (GET /token) and never prints that one. Every run that sends an authenticated request
  fetches one, and the API allows 2 /token calls a minute per IP, so pass up to 10
  metering points to one run rather than running once per metering point.

Rules that change the result
  - Dates are Copenhagen calendar dates, and a range is half-open, [from, to): --from is
    included, --to is not. --to defaults to today, so the range ends with yesterday.
    --from 2026-09-01 --to 2026-10-01 is all of September. For timeseries and
    export-timeseries, from and to on the same date is rejected (API error 30002).
  - A time series range spans at most 730 days (API error 30014).
  - Commands that take metering point IDs take 1 to 10 of them, each exactly 18 digits.
  - Results go to stdout as JSON, except export-* (CSV unless --format json) and alive
    (one line of text). Warnings, errors and the headers --print-response-headers prints
    go to stderr. A failed command exits with status 1.
  - A metering point can fail on its own inside a successful response, and the command
    still exits with status 0: check "success", "errorCode" and "errorText" of every
    element ("error" of every result for charge-links). timeseries --flatten leaves a
    failed metering point out and reports it as a warning on stderr instead. The same
    metering point can also come back once per access period.
  - charges returns the charges valid now or taking effect later, never past prices.
  - charge-links currently answers 404 on both APIs: Energinet has not enabled it.
  - add-relation-by-code and delete-relation are retired: Energinet retired their
    endpoints with DataHub 3.0, and both commands fail without calling the API.
  - A 429 (rate limit) or 503 (DataHub busy) answer is retried up to twice, after a wait
    of several seconds, so a command can take a while before it answers or fails.

Full reference for the CLI and the Go library:
https://github.com/slimcdk/go-eloverblik/blob/master/llms.md

Usage:
  go-eloverblik [command]

Examples:
  export ELO_TOKEN='<refresh token from eloverblik.dk>'
  go-eloverblik token --token "$ELO_TOKEN"
  go-eloverblik customer installations --token "$ELO_TOKEN"
  go-eloverblik customer timeseries 571313000000000001 --from 2026-09-01 --to 2026-10-01 --aggregation Day --token "$ELO_TOKEN"
  go-eloverblik thirdparty authorizations --token "$ELO_TOKEN"
  go-eloverblik customer timeseries --help

Available Commands:

  completion
    bash                     Generate the autocompletion script for bash
    fish                     Generate the autocompletion script for fish
    powershell               Generate the autocompletion script for powershell
    zsh                      Generate the autocompletion script for zsh

  customer
    add-relation             Link one or more metering points to the authenticated user by ID
    alive                    Check if the API is operational
    charge-links             Get charge links with dated charge prices (404 while the feature is disabled)
    charges                  Get charges (subscriptions, fees, tariffs) for one or more metering points
    details                  Get metering point details
    export-charges           Export charges (customer API only)
    export-masterdata        Export metering point masterdata (customer API only)
    export-timeseries        Export time series as CSV or JSON (customer API only)
    installations            Get metering points (installations)
    timeseries               Get time series for one or more metering points

  thirdparty
    alive                    Check if the API is operational
    authorizations           Get authorizations (powers of attorney) granted by customers
    charge-links             Get charge links with dated charge prices (404 while the feature is disabled)
    charges                  Get charges (subscriptions, tariffs) for one or more metering points
    details                  Get metering point details
    metering-point-ids       Get metering point IDs accessible under a specific authorization scope
    metering-points          Get metering points accessible under a specific authorization scope
    timeseries               Get time series for one or more metering points
  token                      Show what the Eloverblik token says about itself

Flags:
  -h, --help                     help for go-eloverblik
      --print-response-headers   Print HTTP response headers from the Eloverblik API to stderr
      --token string             Eloverblik refresh token, created at eloverblik.dk (required by the customer, thirdparty and token commands)

Use "go-eloverblik [command] --help" for more information about a command.
```

`charge-links` is registered on both `customer` and `thirdparty`, and both currently fail:
Energinet has not enabled `getchargelinkswithcharges` on either API. See
[the note above](#note-on-charge-links).

Every command below the root has cobra's full help. For a command such as
`go-eloverblik customer timeseries --help`, it names the endpoint the command calls, its
arguments and flags, the JSON it prints and examples, followed by the global flags.

### Global Flags

Every command below the root lists them in its help:

```
Global Flags:
      --print-response-headers   Print HTTP response headers from the Eloverblik API to stderr
      --token string             Eloverblik refresh token, created at eloverblik.dk (required by the customer, thirdparty and token commands)
```

`help` and `completion` do not need `--token`, so `go-eloverblik help <command>` and
`go-eloverblik completion bash` work without one.

`--print-response-headers` is a debugging aid. The headers of every API call, including
the token call, are written to stderr, so stdout stays clean, parseable output:

```bash
go-eloverblik customer details <metering-id> --token=$TOKEN --print-response-headers 2>headers.txt
```

Truncated: the token call's block below shows only some of its headers, and the block of
the `details` call follows it.

```
< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Mon, 01 Jan 2024 00:00:00 GMT
...
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
  "tokenName": "example",
  "name": "Test User",
  "company": "Test Company ApS",
  "cvr": "12345678",
  "roles": ["ReadPrivate", "ReadBusiness"],
  "expiresAt": "2027-12-28T14:20:00+01:00"
}
```

Shown indented and shortened here. The command prints the claims as a single line of JSON
with no trailing newline, and a real token usually also carries `tokenId`, `subject`,
`userId`, `thirdPartyId`, `loginType`, `webApp`, `issuer` and `audience`. Pipe it through
`jq .` to read it.

Add `--data-access` to exchange the refresh token for a short lived data access token and
decode that one instead. That does make a request, and the client to use is taken from the
token itself:

```bash
go-eloverblik token --data-access --token=$TOKEN
```

### Customer Commands

```bash
# Installation Management
go-eloverblik customer installations                    # List metering points linked to you
go-eloverblik customer installations --include-all      # Also unlinked ones registered to your CPR/CVR
go-eloverblik customer details <metering-id>...         # Get detailed information

# Relations
go-eloverblik customer add-relation <metering-id>...    # Add relation by ID
# add-relation-by-code and delete-relation are retired: Energinet's endpoints answer 410
# since DataHub 3.0. The commands are hidden and say so without calling the API.

# Data Retrieval
go-eloverblik customer timeseries <metering-id>... --period=last_month
go-eloverblik customer timeseries <metering-id>... --from=now-30d --to=now

# Use --from/--to for specific ranges, read in Copenhagen time whatever the host's zone.
# --from is included; --to is excluded and defaults to today's date in Copenhagen, so the
# range ends with yesterday.
# Both take:
#   YYYY-MM-DD   (midnight at the start of that day in Copenhagen)
#   now
#   now-30d (days), now-4w (weeks), now-2m (months), now-1y (years)
#
# Use --period for common ranges (cannot be used with --from/--to):
#   yesterday, this_week, last_week, this_month, last_month,
#   this_year, last_year
# Weeks start on Monday. For timeseries and export-timeseries a this_* period ends with
# yesterday; for charge-links it runs up to now, today included. A this_* period fails on
# its first day (a Monday, the 1st, 1 January), which has no complete day yet.

go-eloverblik customer timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --aggregation=Hour \
  --flatten
# --aggregation: Actual, Quarter, Hour, Day, Month, Year
# --flatten: simplify output

go-eloverblik customer charges <metering-id>...         # Get charges and tariffs
# NOTE: 'charges' only returns charges that are currently valid or take effect in the
# future. It cannot price consumption that already happened.

# Charge links with the dated price series of every linked charge.
# NOT AVAILABLE: Energinet has not enabled getchargelinkswithcharges. Checked 2026-07-13
# with a valid customer token, the Customer API answered 404 while 'charges' answered 200.
# The command implements the endpoint as specified, with one interval for all metering
# points, and is ready for the day it is enabled; today it returns a 404 error. Until then,
# 'charges' above is the closest data available. --to is excluded, as for timeseries.
go-eloverblik customer charge-links <metering-id>... --period=last_month
go-eloverblik customer charge-links <metering-id>... --from=YYYY-MM-DD --to=YYYY-MM-DD

# Data Export (CSV or JSON). The API returns CSV; --format=json makes the CLI convert it to
# an array of objects keyed by the CSV header, without the byte order mark the CSV starts
# with, and [] when there are no rows.
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
go-eloverblik thirdparty authorizations                 # List valid or active authorizations

# Metering Points
go-eloverblik thirdparty metering-points <scope> <identifier>
  # Scope: authorizationId, customerCVR, customerKey

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
# NOT AVAILABLE: Energinet has not enabled getchargelinkswithcharges. Checked 2026-07-13
# with a valid third-party token, the Third-Party API answered 404 while 'charges' answered
# 200 — exactly as the Customer API did. The command implements the endpoint as specified,
# with one interval for all metering points, and is ready for the day it is enabled; today
# it returns a 404 error. --to is excluded, as for timeseries.
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

Both constructors accept optional options. Both clients always call the production API at
`api.eloverblik.dk`. The package variables `Mode`, `TestMode`, `ReleaseMode` and `ApiType`,
and the `APIType` type, are deprecated and have no effect; `TokenClaims.APIType()` is
unrelated and not deprecated.

Create one client per refresh token and share it: it is safe for concurrent use by
multiple goroutines. It exchanges the refresh token for a data access token at `/token` on
the first call that needs one, caches it, and fetches a new one once the cached token has
expired or expires within five minutes, so a long-running process can keep the same
client. A data access token lasts about 24 hours.

The API allows 2 `/token` calls a minute per IP, so goroutines that need a token while one
is being fetched wait for that request and share its outcome, the token or the error,
instead of sending their own. When a renewal fails while the cached token has not expired
yet, the call gets the cached token and the next call tries again. A data access token
whose expiry cannot be read, because it is not a JWT or its `exp` claim is missing, null,
zero or negative, counts as never expiring: the client keeps it and leaves it to the API to
reject it.

### Debugging Response Headers

`WithResponseHeaderOutput` writes the HTTP response headers of every API call, including
the token call and the streamed exports, to the given `io.Writer`:

```go
customer := eloverblik.NewCustomer("refresh-token", eloverblik.WithResponseHeaderOutput(os.Stderr))
```

Truncated, with only some of the headers of one call:

```
< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Mon, 01 Jan 2024 00:00:00 GMT
...
```

### Reading Token Claims

Both Eloverblik tokens are JWTs. `ParseToken` decodes the claims of any of them, and the
client can read its own:

```go
claims, err := eloverblik.ParseToken(refreshToken)

claims.TokenName   // the name given to the token in the portal
claims.Roles       // []string{"ReadPrivate", "ReadBusiness"}
claims.Company     // "Test Company ApS"
claims.ExpiresAt   // time.Time, in Copenhagen time; zero when exp is missing, null, zero or negative
claims.IsExpired() // no request needed to find out; false when the token carries no expiry
claims.APIType()   // (eloverblik.ThirdPartyApi, nil), read from the token type; an error when it names neither API

client := eloverblik.NewThirdParty(refreshToken) // a third-party token, as APIType() said
claims, err = client.RefreshTokenClaims()        // no request
claims, err = client.DataAccessTokenClaims()     // fetches or renews the data access token first when needed
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

> **Energinet has not switched it on — on either API.** Both OpenAPI documents declare
> `getchargelinkswithcharges` and document a `404` for it: "When the Charges integration
> feature is disabled". That is what the live API answers on the **Customer API and the
> Third-Party API alike**. Checked on **2026-07-13** with a valid Customer token and a
> valid Third-Party token, on every documented path, in a session where `getcharges`
> returned `200 OK` for the same tokens — so it is not an auth problem: the feature is
> switched off. This client implements the endpoint's request and response as both
> specifications describe them, except that one call applies the same interval to every
> metering point, and is ready for the day Energinet enables it. Until then,
> `GetChargeLinksWithCharges` returns a 404 error on both clients, and **there is no way to
> price historic consumption through this API**: `GetCustomerCharges` /
> `GetThirdPartyCharges` (the `charges` commands) are the closest available data, and they
> only carry present and future prices.

The code below is what the endpoint will return once it is enabled — it is included so
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

### Exports

`ExportTimeSeries`, `ExportMasterdata` and `ExportCharges` (Customer API only) return the
CSV file the API generates, as an `io.ReadCloser` streamed as it arrives, and the caller
must close it. The CSV is separated by semicolons, starts with a UTF-8 byte order mark and
has Danish column names. The library returns it as it is: JSON exists only in the CLI,
whose `--format json` converts the CSV.

```go
csv, err := client.ExportTimeSeries(ids, from, to, eloverblik.Day) // client is a Customer
if err != nil {
    log.Fatal(err)
}
defer csv.Close()

if _, err := io.Copy(os.Stdout, csv); err != nil {
    log.Fatal(err)
}
```

When an export is retried after a 429 or a 503, the client closes the body of every
attempt it discards; when it fails, the client closes the body itself.

### Aggregation Levels

The aggregation you ask for:

```go
eloverblik.Actual  // Raw meter readings
eloverblik.Quarter // 15-minute aggregation
eloverblik.Hour    // Hourly aggregation
eloverblik.Day     // Daily aggregation
eloverblik.Month   // Monthly aggregation
eloverblik.Year    // Yearly aggregation
```

The resolution the API answers with is a different vocabulary, and it is worth knowing
before you parse a response yourself:

| Aggregation | `resolution` on the wire | Shape of the response |
|---|---|---|
| `Quarter` | `PT15M` | one period per day, 96 points (92 or 100 on daylight saving days) |
| `Hour` | `PT1H` | one period per day, 24 points (23 or 25 on daylight saving days) |
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
eloverblik.AuthScopeID          // Scope by authorization ID
eloverblik.AuthScopeCustomerCVR // Scope by customer CVR number
eloverblik.AuthScopeCustomerKey // Scope by customer key
```

### Dates Are Half-Open

The API reads a requested range as `[dateFrom, dateTo)`, at the granularity of a date:
`dateFrom` is included, `dateTo` is not, and a request where the two are equal is rejected
outright with error 30002.

```go
cph, err := time.LoadLocation("Europe/Copenhagen") // never fails: the package embeds time/tzdata
if err != nil {
    // handle error
}

// Asking for 1 July through 4 July returns 1, 2 and 3 July — three days, not four.
from := time.Date(2026, 7, 1, 0, 0, 0, 0, cph)
to := time.Date(2026, 7, 4, 0, 0, 0, 0, cph)
timeseries, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)
```

`GetTimeSeries` (and `ExportTimeSeries`) convert `from` and `to` to Copenhagen time and
send only the date, so build them in `Europe/Copenhagen` or UTC. On a host east of
Copenhagen, local midnight is still the previous day in Copenhagen.
`GetChargeLinksWithCharges` is the exception: it sends both bounds as timestamps in
Copenhagen time and keeps the time of day, so pass Copenhagen midnights to ask for whole
days.

So to get a single day, ask for that day and the next one. This is the off-by-one that
makes `--period yesterday` sound like it should send the same date twice; it must not.

### Period Constants

The `Period` type and `GetDatesFromPeriod` cover the common ranges and already return an
exclusive `to`, so a period never drops its own last day. They work in Copenhagen time
(`Europe/Copenhagen`) whatever the host's time zone: today is the current date in
Copenhagen, `from` and `to` are returned in `Europe/Copenhagen`, and weeks run from Monday
to Sunday.

| Period | `from` | `to` (excluded) |
|--------|--------|-----------------|
| `Yesterday` | 00:00 yesterday | 00:00 today |
| `ThisWeek` | 00:00 on this week's Monday | now |
| `LastWeek` | 00:00 on last week's Monday | 00:00 on this week's Monday |
| `ThisMonth` | 00:00 on the 1st of this month | now |
| `LastMonth` | 00:00 on the 1st of last month | 00:00 on the 1st of this month |
| `ThisYear` | 00:00 on 1 January this year | now |
| `LastYear` | 00:00 on 1 January last year | 00:00 on 1 January this year |

A this_* period runs up to now, so a date-based call such as `GetTimeSeries` stops before
today, which is not complete. On its first day (a Monday for `ThisWeek`, the 1st for
`ThisMonth`, 1 January for `ThisYear`) `from` and `to` would fall on the same date, which
the API rejects with error 30002, so `GetDatesFromPeriod` returns an error wrapping
`ErrorPeriodHasNoCompleteDay` instead, e.g. "this_week: period started today and has no
complete day yet". The error is returned whichever call the bounds are for,
`GetChargeLinksWithCharges` included. On the other days that call, which sends timestamps,
gets a this_* period up to now, today included.

```go
import (
    "errors"

    eloverblik "github.com/slimcdk/go-eloverblik/v1"
)

// Last month, in full: from is the 1st of last month, to is the 1st of this month
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    // handle error
}
timeseries, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)

// This month so far, or last month on the 1st, when this month has no complete day yet
from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.ThisMonth)
if errors.Is(err, eloverblik.ErrorPeriodHasNoCompleteDay) {
    from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
}
```

The API only holds time series for the previous five years plus the current one. It
refuses a `to` later than tomorrow (error 30003) and moves a `to` of tomorrow back to
today, so today's consumption is never available.

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

// No retries: a 429 or 503 is returned to you at once
client = eloverblik.NewCustomer(refreshToken, eloverblik.WithoutRetry())
```

`WithoutRetry` removes only the retries and the waits between them. The client sets no
request timeout and no method takes a context, so a call still blocks until the API answers
or the connection fails.

Energinet's documents recommend asking for **at most 10 metering points per request**.
They do not say that the API rejects more, and whether it does has not been verified. The
CLI enforces 1 to 10; the library does not limit it and sends every ID in one request, so
split a longer list yourself, e.g. with `slices.Chunk(ids, 10)`.

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
        []string{"571313000000000001"},
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
go-eloverblik --token=$ELO_TOKEN customer export-timeseries 571313000000000001 \
  --from=2024-01-01 --to=2025-01-01 > consumption_2024.csv
```

### Export Data to JSON File

```bash
go-eloverblik --token=$ELO_TOKEN customer export-charges 571313000000000001 \
  --format=json > charges.json
```

### Process Multiple Metering Points

```go
meteringPoints := []string{
    "571313000000000001",
    "571313000000000002",
}

details, err := client.GetMeteringPointDetails(meteringPoints)
if err != nil {
    log.Fatal(err)
}

for _, detail := range details {
    if err := detail.Err(); err != nil {
        log.Printf("skipping: %v", err)
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
if err != nil {
    log.Fatal(err)
}

for _, ts := range timeseries {
    // A metering point that failed on its own has nothing to flatten
    if err := ts.Err(); err != nil {
        log.Printf("skipping: %v", err)
        continue
    }

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
        eloverblik.AuthScopeID,
        auth.ID,
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
// ExpiresIn is zero too when the token carries no expiry
if !claims.ExpiresAt.IsZero() && claims.ExpiresIn() < 7*24*time.Hour {
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

The full path a third party walks: list the powers of attorney customers have granted,
collect the metering points of all of them, and read yesterday's consumption, 10 metering
points at a time.

```go
client := eloverblik.NewThirdParty(refreshToken)

authorizations, err := client.GetAuthorizations()
if err != nil {
    log.Fatal(err)
}

// Pool the metering points of every authorization before batching them
var ids []string
for _, auth := range authorizations {
    log.Printf("%s (CVR %s), valid until %s", auth.CustomerName, auth.CustomerCVR, auth.ValidTo)

    authIDs, err := client.GetMeteringPointIDsForScope(eloverblik.AuthScopeID, auth.ID)
    if err != nil {
        log.Printf("  %v", err)
        continue
    }
    ids = append(ids, authIDs...)
}

// Ask for each metering point once, should two authorizations cover the same one
slices.Sort(ids)
ids = slices.Compact(ids)

from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.Yesterday)
if err != nil {
    log.Fatal(err)
}

// Energinet recommends at most 10 metering points per request
for batch := range slices.Chunk(ids, 10) {
    timeseries, err := client.GetTimeSeries(batch, from, to, eloverblik.Hour)
    if err != nil {
        log.Print(err)
        continue
    }

    for _, ts := range timeseries {
        // A metering point can fail on its own while the others succeed
        if err := ts.Err(); err != nil {
            log.Printf("skipping: %v", err)
            continue
        }
        for _, point := range ts.Flatten() {
            fmt.Printf("%s  %s → %s  %.2f %s\n",
                ts.ID,
                point.From.Format(time.RFC3339), point.To.Format(time.RFC3339),
                point.Measurement, point.Unit)
        }
    }
}
```

Note the pooling and the batching. Listing the IDs takes a call per authorization, but the
time series then take one call per 10 metering points, however they are spread over the
authorizations; batching per authorization would send at least one time series request
per authorization, even for one that covers a single metering point. Without batching, a
third party with a few hundred metering points walks straight into the 120 calls per
minute limit; the client retries the resulting 429, but not making the call at all is
faster.

### Debug a Call That Fails

When the API says no and you want to know what it really answered, print the response
headers. They go to stderr, so stdout stays a clean JSON stream you can still pipe:

```bash
go-eloverblik customer timeseries 571313180400000000 \
    --period=yesterday --aggregation=Hour \
    --token=$TOKEN --print-response-headers 2>headers.txt | jq .

cat headers.txt
```

Truncated: only some of the token call's headers are shown, and the block of the
`timeseries` call follows it.

```
< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
< Api-Supported-Versions: 1.0
< Content-Type: application/json; charset=utf-8
< Date: Mon, 13 Jul 2026 00:24:46 GMT
...
```

In the library, the same switch is an option, and it covers the token call and the
streamed exports too:

```go
client := eloverblik.NewCustomer(refreshToken,
    eloverblik.WithResponseHeaderOutput(os.Stderr))
```

### Error Handling

Failures arrive at two levels, and both matter.

**The call itself** fails with an error. When the API names an error code the client knows,
or answers 401, 410 or 429, that error matches a sentinel with `errors.Is`; on those three
statuses it does so for a code the client does not know, too. A code is read only from
five digits in brackets at the start of the message, as in `[20010] Relation not found`.
Anything else matches none and lands in the `err != nil` branch. That includes a 503 that
still fails after the retries, and a problem document such as the 404 that
`getchargelinkswithcharges` answers today, which arrives as an `*APIError` carrying the
status, title and trace ID, and in `Code` the API error code its detail opens with, if
any, also one the client has no sentinel for:

```go
timeseries, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)
switch {
case errors.Is(err, eloverblik.ErrorTokenNotValid):
    // API code 50001: the refresh token is invalid, has expired or been revoked;
    // generate a new one
case errors.Is(err, eloverblik.ErrorUnauthorized):
    // a 401 whose API code, if any, has no sentinel of its own, or API code 20012, e.g. no
    // active relation or authorization for the metering point. The client renews its own
    // data access token before it expires.
case errors.Is(err, eloverblik.ErrorTooManyRequests):
    // still rate limited after the retries; back off for a minute
case errors.Is(err, eloverblik.ErrorNoCprConsent):
    // GetMeteringPoints(true) needs CPR consent, granted once in the portal
case errors.Is(err, eloverblik.ErrorToDateCanNotBeEqualToFromDate):
    // the range is half-open: ask for the day AND the next one
case errors.Is(err, eloverblik.ErrorPeriodNotAllowed):
    // more than 730 days, or a to that is not after from once a future to is moved to today
case err != nil:
    log.Fatal(err)
}
```

A call to an endpoint Energinet has retired, such as `AddRelationByWebAccessCode`, fails
with `ErrorEndpointRetired`. `ErrorNoError` is deprecated: API code 10000 means success,
and no call returns it.

**Individual metering points** can fail inside an otherwise successful response — one
missing relation does not fail the batch, it fails that item. `Err()` reports it, nil when
the metering point succeeded, and unwraps to the same sentinels:

```go
timeseries, err := client.GetTimeSeries(meteringPoints, from, to, eloverblik.Day)
if err != nil {
    log.Fatal(err)
}

for _, ts := range timeseries {
    if err := ts.Err(); err != nil {
        // e.g. "eloverblik: metering point 571313000000000001: 30018
        // MeteringPointDataNotAvailableForTheRequestedPeriod" when the period starts
        // before the metering point was registered in DataHub
        log.Printf("skipping: %v", err)
        continue
    }

    process(ts.Flatten())
}
```

A metering point fails on its own with 20003/20004 (invalid ID), 20008 (not found), 20010
(no relation), 20011 (unexpected error), 30010 (period not covered by the authorization),
30014 (period not allowed), 30015 (no data), 30016 (relation expired), 30018 (period
outside the metering point's data) or 40014 (no authorization). `Err()` works the same on
metering point details, charges and relation results.

A 429 or a 503 is retried for you (twice, honouring `Retry-After`) before it ever becomes
an error. Everything else — a 401, a 404, a rejected date range — is returned straight
away, because retrying it would only waste a call against the rate limit.

## Development

`go build .` builds the CLI and `go test ./...` runs the tests. The checks to run before
every commit are kept in one place, [AGENTS.md](AGENTS.md): the full list and how CI runs
each one, including golangci-lint pinned to v2.14.0, the 32-bit runs, and
`go mod tidy -diff`, which the release gate and GoReleaser run too. It also holds the
project's conventions and how a release is cut.

## Project Structure

```
.
├── cmd/                    # CLI command implementations
│   ├── alive.go            # Health check commands
│   ├── authorizations.go   # Authorization commands
│   ├── chargelinks.go      # Charge links commands
│   ├── charges.go          # Charges commands
│   ├── customer.go         # Customer command group; builds the Customer client
│   ├── helpers.go          # Flag handling shared by the commands
│   ├── installations.go    # Customer installations command
│   ├── measurements.go     # details, timeseries, export-timeseries and export-masterdata commands, plus shared argument, date and output helpers
│   ├── relations.go        # Relation commands
│   ├── root.go             # Root command and initialization
│   ├── thirdparty.go       # Third-party specific commands
│   ├── token.go            # Token inspection command
│   └── *_test.go           # CLI tests
├── v1/                     # Library implementation
│   ├── auth.go             # Token exchange, IsAlive and the third-party authorization endpoints
│   ├── chargelinks.go      # Charge links endpoints
│   ├── charges.go          # Charges endpoints
│   ├── constvars.go        # Aggregations, resolutions and other constants
│   ├── doc.go              # Package overview, the start of the godoc
│   ├── eloverblik.go       # Client initialization
│   ├── errors.go           # Error handling
│   ├── export.go           # Streamed export responses
│   ├── interfaces.go       # API interfaces
│   ├── jwt.go              # Token claim decoding
│   ├── meters.go           # Metering point endpoints
│   ├── models.go           # Shared types (FlexibleTime, StatusResponse, StringResponse)
│   ├── options.go          # Client options
│   ├── periods.go          # Relative period helpers
│   ├── relations.go        # Relations endpoints
│   ├── timeseries.go       # Timeseries endpoints
│   ├── utils.go            # Internal helpers
│   └── *_test.go           # Unit tests, and the examples on pkg.go.dev (example_test.go)
├── docs/                   # Energinet's API documentation, see docs/README.md
├── .github/
│   ├── dependabot.yml      # Dependabot updates of the Go modules, Actions and devcontainer
│   └── workflows/
│       ├── release.yml     # Tag-triggered test gate and GoReleaser build
│       ├── security.yml    # govulncheck and CodeQL
│       └── test.yml        # Test, lint, tidy and build
├── .golangci.yml           # Linter configuration
├── .goreleaser.yaml        # Release build configuration
├── AGENTS.md               # Checks and conventions for contributors and coding agents
├── CHANGELOG.md            # Notable changes per release, published as the release notes
├── CLAUDE.md               # Symbolic link to AGENTS.md
├── LICENSE                 # MIT license
├── README.md               # This file
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── llms.md                 # Full reference for the library and the CLI, for integrators and AI agents
└── main.go                 # CLI entry point
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run the checks listed in [AGENTS.md](AGENTS.md#checks)
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

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

- [Full reference for the library and the CLI (llms.md)](llms.md)
- [Package documentation on pkg.go.dev](https://pkg.go.dev/github.com/slimcdk/go-eloverblik/v1)
- [API Documentation](https://api.eloverblik.dk/customerapi/index.html)
- [Report Issues](https://github.com/slimcdk/go-eloverblik/issues)

## Related Projects

- [Eloverblik API Documentation](https://docs.eloverblik.dk/)
- [Eloverblik Portal](https://eloverblik.dk/)

---

Made for the Danish energy community
