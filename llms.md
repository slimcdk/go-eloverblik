# Eloverblik Go Client - AI-Optimized Documentation

This documentation is structured for AI agents to easily understand, parse, and implement the Eloverblik Go client library and CLI.

AI agents and integrators have three references: this file (the full reference, for the
library and the CLI), the package documentation on pkg.go.dev
(https://pkg.go.dev/github.com/slimcdk/go-eloverblik/v1, a package overview plus examples) for
the library, and the CLI's own `--help` for the CLI.

## Package Information

```yaml
module: github.com/slimcdk/go-eloverblik   # the module root is the CLI (package main), not the library
package: eloverblik                        # the identifier the import below binds
import: github.com/slimcdk/go-eloverblik/v1
install: go get github.com/slimcdk/go-eloverblik/v1
godoc: https://pkg.go.dev/github.com/slimcdk/go-eloverblik/v1 (offline, go doc -all github.com/slimcdk/go-eloverblik/v1)
cli_help: go-eloverblik --help carries the rules (token, dates, IDs, output); the --help of
  every API command it lists (the customer and thirdparty commands, and token) gives its
  endpoint, arguments, flags, output shape and examples
language: Go
purpose: Interface with Danish Eloverblik electricity data API
host: api.eloverblik.dk (always; the client has no test/preprod switch)
api_version: pinned to "1.0" via the api-version header on every request
apis:
  - Customer API (consumer/household electricity data)   -> /customerapi/api
  - Third-Party API (business/multi-customer data access) -> /thirdpartyapi/api
```

## Core Concepts

### 1. Authentication Flow
```
User obtains refresh token → NewCustomer/NewThirdParty creates client → Client automatically:
  1. Uses refresh token to get a data access token on the first call that needs one
  2. Caches the data access token on the client
  3. Renews it on the first call after it has expired or comes within 5 minutes of expiring,
     as its exp claim says
  4. Sends the data access token as `Authorization: Bearer <token>` on every call except
     IsAlive, which calls /isalive without authentication (the /token call itself carries
     the refresh token)
```
Note: a data access token lasts about 24 hours, and the client renews it itself, so a long
running process keeps the same client. A token whose exp is missing, null, zero or negative
counts as never expiring and is kept. The client is safe for concurrent use: goroutines that
need a token while one is being fetched wait for that /token request and share its outcome,
the token or the error, instead of each sending their own (the API allows 2 /token calls a
minute). A call made after a failed request sends a new one. When a renewal fails while the
cached token has not expired yet, the call gets the cached token, not the error, and the
next call tries again. A call fails on the token only when no working token is left: /token
failed and nothing unexpired is cached, e.g. with ErrorTokenNotValid for an expired or
revoked refresh token.

### 2. Client Types
```yaml
Customer:
  purpose: Individual consumers accessing their own data
  factory: eloverblik.NewCustomer(refreshToken string, opts ...Option) Customer
  capabilities: [installations, details, timeseries, charges, charge-links, relations, exports, token claims]

ThirdParty:
  purpose: Businesses accessing multiple customers' data via authorization
  factory: eloverblik.NewThirdParty(refreshToken string, opts ...Option) ThirdParty
  capabilities: [authorizations, metering-points, details, timeseries, charges, charge-links, token claims]

Client:
  purpose: The methods both clients share. Both Customer and ThirdParty embed it.
  methods: [GetDataAccessToken, RefreshTokenClaims, DataAccessTokenClaims,
            GetMeteringPointDetails, GetTimeSeries, GetChargeLinksWithCharges, IsAlive]
```

!! `charge-links` / `GetChargeLinksWithCharges` is listed above because it is declared in
both OpenAPI documents and implemented here, NOT because it works. Both documents describe
its 404 as "When the Charges integration feature is disabled", and that is what the live API
answered when checked on 2026-07-13 with valid Customer AND Third-Party tokens: **404 on
BOTH APIs**, while `getcharges` answered 200 on the same tokens in the same session. Do not
generate code that depends on it succeeding today; see its function block below for what to
do instead.

Deprecated and inert: `Mode`, `ReleaseMode`, `TestMode` and `ApiType` are package variables
left over from an earlier design, and `APIType` is the type only `ApiType` uses (unrelated to
the method `TokenClaims.APIType`). All five are marked `Deprecated:`, so staticcheck and
gopls flag their use, and go in v2. They have no effect: both constructors always target
`https://api.eloverblik.dk`, although `Mode` defaults to `TestMode` ("preprod"). Do not
generate code that sets them.

### 3. Client Options
```yaml
WithResponseHeaderOutput:
  signature: eloverblik.WithResponseHeaderOutput(w io.Writer) Option
  purpose: Debugging - write the HTTP response headers of every API call to w
  covers: [token call, regular calls, streamed exports]
  notes: A nil writer is ignored. Write errors are swallowed.

WithRetry:
  signature: eloverblik.WithRetry(count int, maxWait time.Duration) Option
  purpose: Override the retry policy for the documented rate limits
  args:
    count: retries after the initial attempt; a negative value is clamped to 0
    maxWait: caps a single wait, including a Retry-After asked for by the server;
             zero or negative falls back to DefaultRetryMaxWait
  default: 2 retries on HTTP 429 and 503 only, honouring Retry-After, capped at 60s

WithoutRetry:
  signature: eloverblik.WithoutRetry() Option
  purpose: Fail immediately instead of retrying a 429 or 503

Exported defaults:
  eloverblik.DefaultRetryCount   = 2               // retries after the initial attempt
  eloverblik.DefaultRetryWait    = 5 * time.Second // base backoff, jittered and doubled by resty
  eloverblik.DefaultRetryMaxWait = 60 * time.Second
```

### 4. Token Claims
```yaml
ParseToken:
  signature: eloverblik.ParseToken(token string) (TokenClaims, error)
  purpose: Decode the claims of a refresh or data access token. No request, no signature
           verification (only Energinet holds the key). Works on both token types.
  fields: [TokenType, TokenName, TokenID, Name, Subject, Company, CVR, UserID,
           ThirdPartyID, "Roles []string", LoginType, WebApp, Issuer, Audience,
           "ExpiresAt time.Time (Europe/Copenhagen; zero when exp is missing, null, zero
           or negative)"]
  errors: not a JWT (not 3 dot separated parts), undecodable payload, an exp that is not an
          integer (null aside), "token carries no Eloverblik claims" when it has neither a
          tokenType nor a positive exp
  methods:
    - IsExpired() bool                   // false when ExpiresAt is zero
    - ExpiresIn() time.Duration          // 0 once expired or when ExpiresAt is zero
    - IsRefreshToken() bool              // TokenType contains "refresh"
    - IsDataAccessToken() bool           // TokenType contains "dataaccess"
    - APIType() (apiType, error)         // CustomerApi or ThirdPartyApi, read off TokenType
                                         // (not the deprecated type APIType)

Client.RefreshTokenClaims:
  signature: client.RefreshTokenClaims() (TokenClaims, error)
  purpose: Claims of the refresh token the client was created with. No request.

Client.DataAccessTokenClaims:
  signature: client.DataAccessTokenClaims() (TokenClaims, error)
  purpose: Claims of the data access token, fetching one first if the client has none, or
           renewing it when it has expired or expires within 5 minutes.

TokenType examples: CUSTOMERAPI_Refresh, THIRDPARTYAPI_Refresh, ThirdPartyApiDataAccess,
                    and presumably CustomerApiDataAccess for the Customer API (by analogy,
                    not confirmed)
Roles examples:     ReadPrivate, ReadBusiness (a comma separated claim, split into a slice)

CLI: go-eloverblik token --token=$TOKEN [--data-access]
```

### 5. Date Semantics (half-open range) - READ THIS BEFORE PICKING DATES

The API reads a requested range as **half-open at date granularity**: `[dateFrom, dateTo)`.
This is the reading the client documents and is built on. The OpenAPI documents do not say
whether dateTo is included; they say that equal dates are rejected (30002) and that dateTo
may be at most tomorrow (30003):

```yaml
inclusive: dateFrom
exclusive: dateTo
granularity: date (GetTimeSeries and ExportTimeSeries format both bounds with time.DateOnly
  in Europe/Copenhagen; GetChargeLinksWithCharges is the exception, see below)
equal dates: rejected outright with API error 30002 (ErrorToDateCanNotBeEqualToFromDate)
worked example:
  request: from=2026-07-01, to=2026-07-04, aggregation=Day
  returns: exactly 1 July, 2 July and 3 July. 4 July is NOT returned.
implication:
  to include a final day D, pass to = D + 1 day.
maximum span: 730 days (API error 30014 PeriodNotAllowed beyond it; see MaximumDayRequestLeap)
future to: a to later than tomorrow is rejected (30003); a to of tomorrow is moved back to
  today, so today's consumption is never returned
```

`GetDatesFromPeriod` returns an **exclusive** `to`: for yesterday, last_week, last_month and
last_year it is the start of the period that follows; for this_week, this_month and this_year
it is the current instant (`now`), which the date-formatted calls turn into today's date, so
today is never included. It computes every period in Europe/Copenhagen, whatever the host's
time zone: today is the current Copenhagen date, every bound but a `now` is 00:00 Copenhagen
time, and weeks run from Monday to Sunday:

```yaml
yesterday:  "[00:00 yesterday, 00:00 today)"
this_week:  "[00:00 on this week's Monday, now)"
last_week:  "[00:00 on last week's Monday, 00:00 on this week's Monday)"
this_month: "[1st of this month, now)"
last_month: "[1st of last month, 1st of this month)"
this_year:  "[1 January this year, now)"
last_year:  "[1 January last year, 1 January this year)"
first day:  >-
            on a Monday, the 1st of a month and 1 January, this_week, this_month and this_year
            would give from and to on the same date, so GetDatesFromPeriod returns an error
            wrapping ErrorPeriodHasNoCompleteDay instead ("this_week: period started today and
            has no complete day yet"), and zero from/to. Match it with errors.Is and fall back,
            e.g. from this_month to last_month.
```

Both `from` and `to` are converted to `Europe/Copenhagen` before being formatted, so a UTC
`time.Time` near midnight can land on the neighbouring Danish date. Prefer building dates in
Copenhagen local time or at midday UTC.

GetChargeLinksWithCharges is different: its request takes date-times, and the client sends both
bounds as full RFC 3339 timestamps converted to Copenhagen time, keeping the time of day. A UTC
midnight goes out as `2026-01-01T01:00:00+01:00` in winter and `2026-07-01T02:00:00+02:00` in
summer, so build its bounds at Copenhagen midnight, as GetDatesFromPeriod does; the midday-UTC
advice applies only to the date-formatted calls. For it a this_* period runs up to now, today
included.

### 6. Resolutions in the response (the OpenAPI description is wrong here)

The `Aggregation` you request is not the `resolution` string you get back. What the live wire
actually sends, per requested aggregation:

```yaml
Aggregation Day     -> resolution "PT1D"   one Period per day,   1 point per Period
Aggregation Month   -> resolution "P1M"    one Period per month, 1 point per Period
Aggregation Year    -> resolution "PT1Y"   one Period per year,  1 point per Period
Aggregation Hour    -> resolution "PT1H"   one Period per day,   24 points per Period
                                           (23 or 25 on a daylight saving day)
Aggregation Quarter -> resolution "PT15M"  15-minute points (96 a day, 92 or 100 on a
                                           daylight saving day)
Aggregation Actual  -> the meter's own reading resolution
```

The OpenAPI description documents `P1D` and `P1Y` instead, and adds `PXD` for profiled
energy quantities covering a variable number of days. The library declares and accepts **both
spellings**, so `Flatten()` is correct whichever the server sends:

```go
// Declared in package eloverblik; an excerpt, not for pasting
const (
    PT15M Resolution = "PT15M"
    PT1H  Resolution = "PT1H"
    PT1D  Resolution = "PT1D"  // sent by the live API for Day
    P1M   Resolution = "P1M"
    PT1Y  Resolution = "PT1Y"  // sent by the live API for Year
    P1D   Resolution = "P1D"   // spec spelling, accepted
    P1Y   Resolution = "P1Y"   // spec spelling, accepted
    PXD   Resolution = "PXD"   // spec only, variable width, points spread evenly
)
```

Notes that matter when interpreting the data:
- A Period holding a **single point** takes the interval the API states verbatim. This keeps
  a **partial period** correct: a Year period may legitimately run 27 April - 31 December.
- Periods holding several points step by **elapsed time** (a fixed duration) for PT15M and
  PT1H, so each of the 23 or 25 hourly points (92 or 100 quarters) the API sends for a
  daylight saving day lands at its true instant. Flatten does not create or drop points: it
  places the ones the API sent. They step by **calendar unit** (AddDate) for PT1D/P1D, P1M and
  PT1Y/P1Y, so each point starts at local midnight even across a 23- or 25-hour day and
  months of unequal length come out right. PXD spreads the points evenly over the period.
- An unknown resolution attributes the whole period to the point rather than collapsing it
  to a zero-width interval.

### 7. Rate Limits (documented by the API) and Batch Size (only recommended)

```yaml
token endpoint: 2 calls per minute per IP
all endpoints:  120 calls per minute per IP
overall:        1200 calls per minute across all users
batch size:     10 metering points per request, Energinet's usage recommendation ("Bundle
                requests for 10 metering points at a time"), not a documented limit; whether
                the API rejects more is unverified. The library sends whatever it is given;
                the CLI takes 1 to 10 IDs per command
on breach:      HTTP 429 (and HTTP 503 when DataHub cannot keep up)
client policy:  429 and 503 are retried DefaultRetryCount (2) times, honouring Retry-After,
                capped at DefaultRetryMaxWait (60s). Nothing else is retried - a 401, any
                other 4xx and a 500 are returned to the caller immediately. Transport errors
                are not retried either, so a request is never sent twice by accident. The
                client drains and closes the body of every export attempt it retries.
```

### 8. DataHub 3.0 (in operation since 18 September 2026)

Energinet replaced DataHub 2 with DataHub 3.0. The Eloverblik API kept its endpoints and its
api-version of 1.0; what changed:

```yaml
retired endpoints (Customer API, HTTP 410 Gone -> ErrorEndpointRetired):
  - PUT    /meteringpoints/meteringpoint/relation/add/{id}/{webAccessCode}
           AddRelationByWebAccessCode, CLI add-relation-by-code. Web access codes are gone;
           ElOverblik's data sharing replaces them and has no API.
  - DELETE /meteringpoints/meteringpoint/relation/{id}
           DeleteRelation, CLI delete-relation. A relation can no longer be deleted.
retired methods and commands: both methods are deprecated; both commands are hidden and fail
  without calling the API
per metering point failures (inside a 200, no market document; check ts.Err()):
  30014 PeriodNotAllowed, 30015 NoDataAvailable, 30016 RelationHasExpired,
  30018 MeteringPointDataNotAvailableForTheRequestedPeriod (also when dateFrom is before the
  metering point was registered in DataHub, so a long period no longer returns partial data),
  besides 20003/20004, 20008, 20010, 20011, 30010 and 40014
request level: 30017 ToDateCutOff (to on or before the oldest date held)
repeated IDs: with more than the latest access period enabled by Energinet, details and time
  series return the same metering point ID once per access period
master data: Energinet lists SettlementMethod, ConsumerCategory, MeterReadingOccurrence,
  EstimatedAnnualVolume and MeterCounterDigits/MultiplyFactor/Unit as retired, and
  ConsumerStartDate, BalanceSupplierStartDate and TaxSettlementDate as unavailable for now,
  so expect them empty. Balance supplier fields are not shared with third parties.
new: MeteringPoints.IsMovedOut (bool); typeOfMP D19 (Capacity Settlement)
```

## Function Signatures with Complete Parameter Specifications

### Client Creation

```go
// FUNCTION: NewCustomer
// PURPOSE: Create Customer API client
// SIGNATURE: NewCustomer(refreshToken string, opts ...Option) Customer
// INPUTS:
//   - refreshToken (string): Obtained from eloverblik.dk portal
//   - opts (...Option): Optional client options (nil options are skipped)
// OUTPUT: Customer interface (embeds Client)
// NOTES: Safe for concurrent use. Create one per refresh token and share it: it fetches,
//        caches and renews the data access token itself.
// EXAMPLE:
client := eloverblik.NewCustomer("your-refresh-token-here")
_ = client // every Customer method below is called on it
```

```go
// FUNCTION: NewThirdParty
// PURPOSE: Create Third-Party API client
// SIGNATURE: NewThirdParty(refreshToken string, opts ...Option) ThirdParty
// INPUTS:
//   - refreshToken (string): Obtained from eloverblik.dk portal
//   - opts (...Option): Optional client options
// OUTPUT: ThirdParty interface (embeds Client)
// NOTES: As for NewCustomer: safe for concurrent use, one per refresh token, shared.
// EXAMPLE:
client := eloverblik.NewThirdParty("your-refresh-token-here")
_ = client // every ThirdParty method below is called on it
```

```go
// FUNCTION: WithResponseHeaderOutput
// PURPOSE: Debugging - write the HTTP response headers of every API call to an io.Writer
// SIGNATURE: WithResponseHeaderOutput(w io.Writer) Option
// INPUT: w (io.Writer) - Destination for the header blocks, e.g. os.Stderr
// OUTPUT: Option (pass to NewCustomer or NewThirdParty)
// NOTES:
//   - Implemented as an http.RoundTripper wrapping resty's transport, not an
//     after-response middleware, so it also covers the streamed exports (which run with
//     SetDoNotParseResponse(true) and skip resty middlewares)
//   - Never consumes the response body, exports keep streaming
//   - Write errors are ignored, debug output never breaks an API call
//   - Concurrent requests are serialised on the writer, so blocks never interleave
// EXAMPLE:
client := eloverblik.NewCustomer("your-refresh-token-here", eloverblik.WithResponseHeaderOutput(os.Stderr))
_ = client

// OUTPUT FORMAT (one block per response, header keys sorted alphabetically; this sample is
// truncated, a real block lists every header the response carried):
// < GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
// < Api-Supported-Versions: 1.0
// < Content-Type: application/json; charset=utf-8
// < Date: Mon, 01 Jan 2024 00:00:00 GMT
```

```go
// FUNCTION: WithRetry / WithoutRetry
// SIGNATURE: WithRetry(count int, maxWait time.Duration) Option
//            WithoutRetry() Option
// PURPOSE: Replace the default retry policy (2 retries on 429/503, Retry-After honoured)
// NOTES: The policy is assigned, not appended, so passing WithRetry twice keeps the last
//        one instead of stacking conditions.
// EXAMPLE:
client := eloverblik.NewCustomer(token, eloverblik.WithRetry(4, 30*time.Second))
noRetry := eloverblik.NewCustomer(token, eloverblik.WithoutRetry()) // retrying turned off
_, _ = client, noRetry
```

### Token Claims

```go
// FUNCTION: ParseToken
// SIGNATURE: ParseToken(token string) (TokenClaims, error)
// PURPOSE: Decode a refresh or data access token. No request, no signature verification.
// EXAMPLE:
claims, err := eloverblik.ParseToken(os.Getenv("ELO_TOKEN"))
if err != nil { /* not a JWT, or carries no Eloverblik claims */ }
api, err := claims.APIType() // eloverblik.CustomerApi or eloverblik.ThirdPartyApi
if err != nil { /* the token type names neither API */ }
fmt.Println(claims.TokenName, claims.Roles, claims.ExpiresAt, claims.IsExpired(), api == eloverblik.ThirdPartyApi)
```

```go
// METHODS: RefreshTokenClaims / DataAccessTokenClaims (on both clients)
// SIGNATURE: client.RefreshTokenClaims() (TokenClaims, error)   // no request
//            client.DataAccessTokenClaims() (TokenClaims, error) // fetches or renews a token if needed
```

### Customer API Methods

```go
// FUNCTION: GetMeteringPoints  (Customer only)
// PURPOSE: List metering points (installations) for the authenticated user
// SIGNATURE: GetMeteringPoints(includeAll bool) ([]MeteringPoints, error)
// INPUTS:
//   - includeAll (bool): false = only points actively linked/related to the user (default in
//     the CLI). true = also merge in non-linked points registered to the user's CPR or CVR.
//     Sent as the query parameter includeAll.
// ERRORS: includeAll=true without CPR consent returns ErrorNoCprConsent (API code 10007)
// OUTPUTS:
//   - []MeteringPoints, error
// MeteringPoints FIELDS:
//   MeteringPointID, TypeOfMP, BalanceSupplierName, StreetCode, StreetName, BuildingNumber,
//   FloorID, RoomID, Postcode, CityName, CitySubDivisionName, MunicipalityCode,
//   LocationDescription, SettlementMethod, MeterReadingOccurrence, FirstConsumerPartyName,
//   SecondConsumerPartyName, ConsumerCVR, DataAccessCVR, MeterNumber,
//   ConsumerStartDate (FlexibleTime), HasRelation (bool), IsMovedOut (bool),
//   ChildMeteringPoints ([]ChildMeteringPoints)
// EXAMPLE:
points, err := client.GetMeteringPoints(false)
if err != nil { /* handle error */ }
var ids []string
for _, point := range points {
    ids = append(ids, point.MeteringPointID) // Use these IDs for other API calls
}
```

```go
// FUNCTION: GetMeteringPointDetails  (Customer and ThirdParty)
// PURPOSE: Get detailed information about specific metering points
// SIGNATURE: GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetailsResponse, error)
// INPUTS:
//   - meteringPointIDs ([]string): IDs of exactly 18 digits each, numeric only; at most 10
//     per request, as Energinet recommends (the library does not check the count)
// OUTPUTS:
//   - []MeteringPointDetailsResponse: one per metering point and access period (an ID can
//     repeat, see below)
//   - error: transport error, or an API-level error that failed the whole request
// MeteringPointDetailsResponse = { Result MeteringPointDetail } + embedded StatusResponse:
//   - Success (bool): true if this specific point succeeded
//   - ErrorCode (int): 10000 = success, other = error
//   - ErrorText (string), ID (string), StackTrace (string)
//   - Err() error: nil on success, else an error that unwraps to the code's sentinel
// MeteringPointDetail FIELDS (59, all string unless noted):
//   MeteringPointID, ParentMeteringPointID, TypeOfMP, EnergyTimeSeriesMeasureUnit,
//   EstimatedAnnualVolume, SettlementMethod, MeterNumber,
//   GridOperatorName, GridOperatorID, GridOperatorIDSchemeAgencyID,
//   MeteringGridAreaIdentification, NetSettlementGroup, PhysicalStatusOfMP,
//   ConsumerCategory, PowerLimitKW, PowerLimitKWDecimal (*float64, NULL on most points -
//   check for nil, a plain float64 would report a limit of 0), PowerLimitA, SubTypeOfMP,
//   ProductionObligation, MpCapacity, MpConnectionType, DisconnectionType, Product,
//   ConsumerCVR, DataAccessCVR, ConsumerStartDate (FlexibleTime), MeterReadingOccurrence,
//   MpReadingCharacteristics, MeterCounterDigits, MeterCounterMultiplyFactor,
//   MeterCounterUnit, MeterCounterType, BalanceSupplierName, BalanceSupplierID,
//   BalanceSupplierIDSchemeAgencyID, BalanceSupplierStartDate (FlexibleTime), TaxReduction,
//   TaxSettlementDate (FlexibleTime), MpRelationType, StreetCode, StreetName, BuildingNumber,
//   FloorID, RoomID, Postcode, CityName, CitySubDivisionName, MunicipalityCode,
//   LocationDescription, FirstConsumerPartyName, SecondConsumerPartyName, ProtectedName,
//   Occurrence (FlexibleTime), MeteringPointAlias, AssetType, MpAddressWashInstructions,
//   DarReference, ContactAddresses ([]ContactAddress), ChildMeteringPoints ([]ChildMeteringPoint)
// EXPECT EMPTY (Energinet lists them, among others, as retired or, for now, unavailable):
//   SettlementMethod, ConsumerCategory, MeterReadingOccurrence, EstimatedAnnualVolume,
//   MeterCounterDigits, MeterCounterMultiplyFactor, MeterCounterUnit, ConsumerStartDate,
//   BalanceSupplierStartDate, TaxSettlementDate. Both OpenAPI documents describe
//   MpRelationType as not used, with no value ever returned. Balance supplier fields are
//   never shared with a third party.
// ERROR HANDLING:
//   Always check every result with detail.Err(), not just the error return. The same
//   metering point ID can appear more than once, once per access period.
// EXAMPLE:
ids := []string{"571313155411053087", "571313155411782079"}
details, err := client.GetMeteringPointDetails(ids)
if err != nil { /* handle network / API error */ }
for _, detail := range details {
    if err := detail.Err(); err != nil {
        log.Printf("skipping: %v", err) // eloverblik: metering point <id>: <code> <name>
        continue
    }
    fmt.Println(detail.ID, detail.Result.GridOperatorName)
    if detail.Result.PowerLimitKWDecimal != nil {
        limit := *detail.Result.PowerLimitKWDecimal
        _ = limit
    }
}
```

```go
// FUNCTION: GetTimeSeries  (Customer and ThirdParty)
// PURPOSE: Retrieve electricity consumption time series data
// SIGNATURE: GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) ([]TimeSeries, error)
// INPUTS:
//   - meteringPointIDs ([]string): metering point IDs, at most 10 per request (recommended,
//     not checked by the library)
//   - from (time.Time): Start date, INCLUSIVE
//   - to   (time.Time): End date, EXCLUSIVE (see "Date Semantics"; equal dates -> error 30002)
//   - aggregation (Aggregation): Actual, Quarter, Hour, Day, Month, Year
// DATE HANDLING:
//   - Both bounds are converted to Europe/Copenhagen and formatted as YYYY-MM-DD
//   - Maximum span 730 days (error 30014 PeriodNotAllowed beyond it)
//   - The client checks neither equal dates nor the span; the API rejects them. Per the
//     OpenAPI documents, data goes back only five years plus the current year
// OUTPUTS:
//   - []TimeSeries (one per metering point and access period: an ID can repeat), error
// DATA STRUCTURE:
//   TimeSeries.MyEnergyDataMarketDocument.TimeSeries[].Periods[].Points[]
//   PeriodResponse: Resolution (string), TimeInterval {Start, End}, Points
//   PointResponse:  Position (int), OutQuantityQuantity (float64), OutQuantityQuality (string)
//   TimeSeries also embeds StatusResponse (Success, ErrorCode, ErrorText, ID, StackTrace)
// ERROR HANDLING:
//   A metering point can fail on its own inside a 200, without a market document:
//   30014, 30015 NoDataAvailable, 30016 RelationHasExpired, 30018 data not available for
//   the period (e.g. dateFrom before the metering point was registered in DataHub), 20010,
//   40014 and others. ts.Err() reports it, nil on success. Check it before Flatten().
// HELPER METHOD: ts.Flatten() []FlatTimeSeriesPoint - resolves each point to its real
//   [From, To) interval in Copenhagen local time: by elapsed time for PT15M and PT1H, by
//   calendar unit for days, months and years. See "Resolutions".
// EXAMPLE:
from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
to := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) // exclusive: yields 1, 2 and 3 July
ts, err := client.GetTimeSeries(
    []string{"571313155411053087"},
    from, to,
    eloverblik.Day,
)
if err != nil { /* handle error */ }
for _, series := range ts {
    if err := series.Err(); err != nil {
        log.Printf("skipping: %v", err)
        continue
    }
    for _, p := range series.Flatten() {
        fmt.Printf("%s -> %s: %.3f %s (%s)\n",
            p.From.Format(time.RFC3339), p.To.Format(time.RFC3339),
            p.Measurement, p.Unit, p.Quality)
    }
}
```

```go
// Declared in package eloverblik; an excerpt, not for pasting
// TYPE: FlatTimeSeriesPoint (what Flatten() returns)
type FlatTimeSeriesPoint struct {
    From         time.Time  `json:"from"`         // inclusive, Europe/Copenhagen
    To           time.Time  `json:"to"`           // exclusive, Europe/Copenhagen
    Measurement  float64    `json:"measurement"`  // the quantity, e.g. kWh
    Quality      string     `json:"quality"`      // A04 measured, A03 estimated, A02 not available, A05 incomplete
    Unit         string     `json:"unit"`         // e.g. KWH
    CurveType    string     `json:"curvetype"`
    BusinessType string     `json:"businesstype"`
    Resolution   Resolution `json:"resolution"`   // PT1H, PT1D, P1M, PT1Y, PT15M ...
}
// There is no Timestamp/Value pair: use From/To and Measurement.
```

```go
// FUNCTION: GetCustomerCharges  (Customer only)
// PURPOSE: Get pricing information (subscriptions, fees, tariffs) valid NOW or in the FUTURE
// SIGNATURE: GetCustomerCharges(meteringPointIDs []string) ([]CustomerChargeResponse, error)
// INPUTS:
//   - meteringPointIDs ([]string): metering point IDs, at most 10 per request (recommended)
// OUTPUTS:
//   - []CustomerChargeResponse, error
// CustomerChargeResponse = { Result CustomerCharges } + embedded StatusResponse
// CustomerCharges:
//   - MeteringPointID (string)
//   - Subscriptions ([]Charge), Fees ([]Charge), Tariffs ([]TariffCharge)
// Charge:       PriceID, Name, Description, Owner, ValidFromDate (FlexibleTime),
//               ValidToDate (FlexibleTime), PeriodType, Price (float64), Quantity (int)
// TariffCharge: PriceID, Name, Description, Owner, ValidFromDate, ValidToDate, PeriodType,
//               Prices ([]TariffPrice)
// TariffPrice:  Position (STRING, "1".."24" for an hourly tariff), Price (float64)
// LIMITATION: historic prices are NOT returned - only charges that are currently valid or
//             take effect in the future, so this CANNOT price consumption that already
//             happened. GetChargeLinksWithCharges is the endpoint meant to close that gap,
//             but it answers 404 on both APIs (the OpenAPI documents' "Charges integration
//             feature is disabled"; checked 2026-07-13), so today this is the closest
//             available data and historic pricing is not possible through this API.
//             PriceID is expected, but not verified, to match ChargeIdentifier.Code of the
//             charge's price series once that endpoint answers, together with Owner;
//             ChargeIdentifier.Type is a type code, and which code goes with subscriptions,
//             fees or tariffs is undocumented.
// EXAMPLE:
charges, err := client.GetCustomerCharges([]string{"571313155411053087"})
if err != nil { /* handle error */ }
for _, charge := range charges {
    if !charge.Success { continue }
    for _, tariff := range charge.Result.Tariffs {
        for _, price := range tariff.Prices {
            hour, _ := strconv.Atoi(price.Position) // Position is a string
            _ = hour
            _ = price.Price
        }
    }
}
```

```go
// FUNCTION: GetChargeLinksWithCharges
// AVAILABLE ON: both the Customer and the Third-Party client (eloverblik.Client)
// SIGNATURE: GetChargeLinksWithCharges(meteringPointIDs []string, from, to time.Time) (*ChargeLinksWithChargesResponse, error)
//
// !!!! THIS CALL RETURNS 404 ON BOTH APIS TODAY: THE FEATURE IS DISABLED !!!!
//   Both OpenAPI documents declare the endpoint and describe its 404 as "When the Charges
//   integration feature is disabled". Checked 2026-07-13 with a valid Customer token AND a
//   valid Third-Party token, on every documented path:
//     POST /customerapi/api/meteringpoints/meteringpoint/getchargelinkswithcharges -> 404
//     POST /thirdpartyapi/api/meteringpoint/getchargelinkswithcharges              -> 404
//   getcharges and getdetails returned 200 with the SAME tokens in the same session, so this
//   is NOT an auth problem. This client implements the endpoint as both specs describe it,
//   except that it applies one interval to every metering point (see INPUTS), and is ready
//   for the day Energinet enables the feature. Until then every call returns a 404
//   *eloverblik.APIError.
//
//   USE INSTEAD TODAY: GetCustomerCharges / GetThirdPartyCharges (CLI: the `charges`
//   commands). They return subscriptions and tariffs (plus fees on the Customer API), but
//   ONLY those currently valid or taking effect in the future - they CANNOT price
//   consumption that already happened.
//   There is no other endpoint that can, so historic pricing is simply unavailable for now.
//   Do not generate code whose happy path depends on GetChargeLinksWithCharges succeeding.
//
// PURPOSE (what it will return once enabled): the charge links of metering points together
//          with the dated price series of every charge they link to - the missing half that
//          GetCustomerCharges/GetThirdPartyCharges cannot supply, i.e. the data needed to
//          price historic consumption.
// INPUTS:
//   - meteringPointIDs ([]string): metering point IDs, at most 10 per request (recommended)
//   - from, to (time.Time): half-open [from, to), applied to every metering point.
//     The wire format takes an interval PER metering point; this client sends the same one
//     for all of them. Both are sent as RFC 3339 timestamps in Copenhagen time, time of day
//     kept: pass Copenhagen midnights for whole days.
// PATHS: customer   -> /meteringpoints/meteringpoint/getchargelinkswithcharges  (404 today)
//        thirdparty -> /meteringpoint/getchargelinkswithcharges                 (404 today)
// OUTPUTS:
//   - *ChargeLinksWithChargesResponse, error
// RETURNS:
//   - Results ([]ChargeLinksWithChargesResult): one per metering point
//     Fields: MeteringPointID, Error (per metering point, empty on success), ChargeLinks
//   - ChargeInformations ([]ChargeInformation): the charges, deduplicated and returned once
//     at the top level, NOT nested inside the links
// KEY TYPES:
//   ChargeLink:              MeteringPointID, ChargeIdentifier, ChargeLinkPeriods
//   ChargeIdentifier:        Code, Owner, Type (the join key between a ChargeLink and a
//                            ChargeInformation; comparable, so usable as a map key).
//                            Type is a type code, e.g. "D03" for the tariff in the test
//                            fixture, a constructed one (the endpoint has never answered);
//                            which code means which kind of charge is undocumented
//   ChargeLinkPeriod:        Factor (int, e.g. number of subscriptions),
//                            From, To (FlexibleTime; To is zero for an open ended link)
//   ChargeInformation:       ChargeIdentifier, TaxIndicator (bool), Resolution (e.g. PT1H),
//                            PricingCategory, ChargeInformationPeriods, ChargeSeriesPoints
//   ChargeInformationPeriod: Name, Description, TransparentInvoicing (bool), From, To,
//                            VATClassification
//   ChargeSeriesPoint:       From, To, Price (float64; the price is valid in [From, To))
// EXAMPLE (this is the shape of the response once the feature is enabled; running it today
//          returns a 404 error on both the customer and the thirdparty client):
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    return err
}
links, err := client.GetChargeLinksWithCharges([]string{"571313155411053087"}, from, to)
if err != nil {
    return err // today this is always the 404: fall back to the charges endpoint
}

charges := make(map[eloverblik.ChargeIdentifier]eloverblik.ChargeInformation)
for _, info := range links.ChargeInformations {
    charges[info.ChargeIdentifier] = info
}
for _, result := range links.Results {
    if result.Error != "" { continue } // errors are reported per metering point
    for _, link := range result.ChargeLinks {
        info := charges[link.ChargeIdentifier]
        for _, point := range info.ChargeSeriesPoints {
            // multiply by the consumption in [point.From, point.To) and by the Factor of
            // the charge link period covering it to get the amount charged
            _ = point.Price
        }
    }
}
```

```go
// FUNCTION: ExportTimeSeries  (Customer only)
// PURPOSE: Export time series data as CSV stream
// SIGNATURE: ExportTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation Aggregation) (io.ReadCloser, error)
// INPUTS:
//   - meteringPointIDs ([]string): metering point IDs, at most 10 per request (recommended)
//   - from, to (time.Time): half-open [from, to), same semantics as GetTimeSeries
//   - aggregation (Aggregation): Data granularity
// OUTPUTS:
//   - io.ReadCloser: CSV data stream (Danish format, semicolon-delimited). Caller closes.
//     The library returns the API's CSV as it is; JSON is only the CLI's own conversion
//     (--format json, see "CSV to JSON Conversion").
//   - error: a transport error or a non-2xx, reported as "failed to export time series: <cause>".
//     The cause is wrapped, so errors.Is / errors.As still match the API error (e.g.
//     ErrorPeriodNotAllowed, *APIError). The error body is read whatever its Content-Type,
//     and the client closes it, as it closes the body of every attempt it retries.
// CSV FORMAT:
//   - Delimiter: semicolon (;)
//   - Encoding: UTF-8 with BOM
//   - Headers: Danish language
//   - Columns: MålepunktsID, Fra_dato, Til_dato, Mængde, Måleenhed, Kvalitet, Type
// USAGE PATTERN:
stream, err := client.ExportTimeSeries(ids, from, to, eloverblik.Hour)
if err != nil {
    return err // there is no stream to close
}
defer stream.Close()
// Option 1: Write to file
file, err := os.Create("output.csv")
if err != nil {
    return err
}
defer file.Close()
if _, err := io.Copy(file, stream); err != nil {
    return err
}
// Option 2, instead of option 1 (the stream can be read once): parse it, as in
// "Pattern 3: Export and Parse CSV Data"
//   reader := csv.NewReader(stream)
//   reader.Comma = ';'
```

```go
// FUNCTION: ExportMasterdata  (Customer only)
// PURPOSE: Export metering point master data as CSV
// SIGNATURE: ExportMasterdata(meteringPointIDs []string) (io.ReadCloser, error)
// OUTPUTS:
//   - io.ReadCloser: CSV stream with the master data columns. Caller closes.
//   - error: HTTP/network errors, or a non-2xx status, wrapped as
//     "failed to export masterdata: <cause>" (as for ExportTimeSeries)
// EXAMPLE:
stream, err := client.ExportMasterdata([]string{"571313155411053087"})
if err != nil {
    return err
}
defer stream.Close()
io.Copy(os.Stdout, stream)
```

```go
// FUNCTION: ExportCharges  (Customer only)
// PURPOSE: Export charges as CSV
// SIGNATURE: ExportCharges(meteringPointIDs []string) (io.ReadCloser, error)
// OUTPUTS:
//   - io.ReadCloser: CSV with subscriptions, fees, tariffs. Caller closes.
//   - error: HTTP/network errors, or a non-2xx status, wrapped as
//     "failed to export charges: <cause>" (as for ExportTimeSeries)
// CSV STRUCTURE: One row per charge item, including hourly tariff positions
```

```go
// FUNCTION: AddRelationByID  (Customer only)
// PURPOSE: Link metering points to the authenticated user
// SIGNATURE: AddRelationByID(meteringPointIDs []string) ([]StringResponse, error)
// OUTPUTS:
//   - []StringResponse: one per ID. StringResponse = { Result string } + StatusResponse
// EXAMPLE:
responses, err := client.AddRelationByID([]string{"571313155411053087"})
if err != nil { /* handle error */ }
for _, resp := range responses {
    if resp.Success {
        fmt.Println("Linked:", resp.Result)
    }
}
```

```go
// FUNCTION: AddRelationByWebAccessCode  (Customer only) - DEPRECATED, DO NOT USE
// STATUS: Energinet retired the endpoint with DataHub 3.0. It answers 410 Gone, returned as
//   ErrorEndpointRetired. Web access codes are gone; ElOverblik's data sharing replaces
//   them and has no API (https://docs.eloverblik.dk/docs/guides/data-sharing).
// SIGNATURE: AddRelationByWebAccessCode(meteringPointID, webAccessCode string) (string, error)
// Use AddRelationByID to link metering points registered to the user's CPR or CVR.
```

```go
// FUNCTION: DeleteRelation  (Customer only) - DEPRECATED, DO NOT USE
// STATUS: Energinet retired the endpoint with DataHub 3.0. It answers 410 Gone, returned as
//   ErrorEndpointRetired, and no longer deletes anything. There is no replacement.
// SIGNATURE: DeleteRelation(meteringPointID string) (bool, error)
```

```go
// FUNCTION: IsAlive  (Customer and ThirdParty)
// PURPOSE: Health check for API availability
// SIGNATURE: IsAlive() (bool, error)
// INPUTS: none. NO TOKEN IS USED - it does not authenticate.
// OUTPUTS:
//   - bool: true if the /isalive endpoint answered 2xx
//   - error: transport errors only
// NOTE: each client probes its OWN API on the shared host api.eloverblik.dk: `customer alive`
//       hits /customerapi/api/isalive and `thirdparty alive` hits /thirdpartyapi/api/isalive.
```

### Third-Party API Methods

```go
// FUNCTION: GetAuthorizations  (ThirdParty only)
// PURPOSE: List the valid or active authorizations (powers of attorney) customers have granted
//          the third party; deleted or expired ones are not returned
// SIGNATURE: GetAuthorizations() ([]Authorization, error)
// INPUTS: none
// Authorization FIELDS:
//   - ID (string): Authorization ID; select its metering points with AuthScopeID
//   - ThirdPartyName (string)
//   - ValidFrom, ValidTo (STRING, not parsed)
//   - CustomerName (string)
//   - CustomerCVR (string)
//   - CustomerKey (string): optional, may be empty - the key the third party gave when it
//     requested the authorization (usable with AuthScopeCustomerKey when set)
//   - IncludeFutureMeteringPoints (bool)
//   - Timestamp (FlexibleTime)
// EXAMPLE:
auths, err := client.GetAuthorizations()
if err != nil { /* handle error */ }
for _, auth := range auths {
    meteringPoints, err := client.GetMeteringPointsForScope(
        eloverblik.AuthScopeID, // every authorization has an ID; a customerKey may be empty
        auth.ID,
    )
    if err != nil {
        return err
    }
    fmt.Println(auth.CustomerName, len(meteringPoints), "metering points")
}
```

```go
// FUNCTION: GetMeteringPointsForScope  (ThirdParty only)
// PURPOSE: Get metering points for a specific authorization scope
// SIGNATURE: GetMeteringPointsForScope(scope AuthorizationScope, identifier string) ([]ThirdPartyMeteringPoint, error)
// INPUTS:
//   - scope (AuthorizationScope): eloverblik.AuthScopeID          ("authorizationId")
//                                 eloverblik.AuthScopeCustomerCVR ("customerCVR")
//                                 eloverblik.AuthScopeCustomerKey ("customerKey")
//   - identifier (string): the authorization ID, CVR number or customer key
// ThirdPartyMeteringPoint FIELDS:
//   MeteringPointID, TypeOfMP, AccessFrom (string), AccessTo (string), StreetCode,
//   StreetName, BuildingNumber, FloorID, RoomID, Postcode, CityName, CitySubDivisionName,
//   MunicipalityCode, LocationDescription, SettlementMethod, MeterReadingOccurrence,
//   FirstConsumerPartyName, SecondConsumerPartyName, ConsumerCVR, DataAccessCVR,
//   MeterNumber, ConsumerStartDate (FlexibleTime), ChildMeteringPoints ([]ChildMeteringPoint)
// WORKFLOW:
//   1. GetAuthorizations() to get the authorization IDs
//   2. For each auth, GetMeteringPointsForScope(AuthScopeID, auth.ID) to get its points
//   3. Pool the metering point IDs of all authorizations and use them, 10 per request, with
//      GetTimeSeries(), GetMeteringPointDetails(), etc. (see Common Pattern 2)
// EXAMPLE:
points, err := client.GetMeteringPointsForScope(
    eloverblik.AuthScopeID,
    "authorization-id-from-GetAuthorizations",
)
if err != nil {
    return err
}
for _, p := range points {
    fmt.Println(p.MeteringPointID, p.AccessFrom, p.AccessTo)
}
```

```go
// FUNCTION: GetMeteringPointIDsForScope  (ThirdParty only)
// PURPOSE: Get only the IDs (a smaller response than GetMeteringPointsForScope)
// SIGNATURE: GetMeteringPointIDsForScope(scope AuthorizationScope, identifier string) ([]string, error)
// USE CASE: When you only need IDs, not the metadata. The Third-Party OpenAPI document
//           recommends switching to it when GetMeteringPointsForScope errors or times out,
//           and requesting the details of the IDs in smaller bundles.
```

```go
// FUNCTION: GetThirdPartyCharges  (ThirdParty only)
// PURPOSE: Get charges for third-party accessed metering points
// SIGNATURE: GetThirdPartyCharges(meteringPointIDs []string) ([]ThirdPartyChargeResponse, error)
// OUTPUTS:
//   - []ThirdPartyChargeResponse = { Result ThirdPartyCharges } + StatusResponse
// ThirdPartyCharges: MeteringPointID, Subscriptions ([]Charge), Tariffs ([]TariffCharge)
// DIFFERENCE FROM CUSTOMER: no Fees list.
// NOTE: Only returns charges that are currently valid or take effect in the future, so it
//       cannot price consumption that already happened. GetChargeLinksWithCharges is the
//       endpoint for historic prices, but it answers 404 on BOTH APIs (the Charges
//       integration feature is disabled; checked 2026-07-13), so this is the closest data
//       available today.
```

## CLI Command Patterns

### Command Structure
```
go-eloverblik --token=<refresh-token> <api-type> <command> [args] [flags]

Components:
  --token: Global flag, required by every customer, thirdparty and token command (help and
           completion run without it)
  --print-response-headers: Global flag, prints HTTP response headers to stderr (debugging)
  <api-type>: "customer" or "thirdparty" (the "token" command sits directly under the root)
  <command>: Action to perform
  [args]: Positional arguments (usually metering point IDs, 1-10, each 18 digits)
  [flags]: Optional command-specific flags
```

### Global Flags
```yaml
--token <string>:
  required: by every customer, thirdparty and token command, which fail with
            `required flag "token" not set` when it is missing or empty. It is a persistent
            flag on the root command, not one cobra requires, so help, `help <command>` and
            `completion <shell>` work without it.
  purpose: The long lived Eloverblik REFRESH token from the portal, not a data access token.
           The client exchanges it for a data access token itself, once in every run that
           sends an authenticated request (alive and token without --data-access send none).

--print-response-headers:
  required: false
  default: false
  purpose: Print HTTP response headers from the Eloverblik API to stderr
  notes: Headers go to stderr, so stdout stays clean, parseable JSON
  example: |
    go-eloverblik customer details <metering-id> --token=$TOKEN --print-response-headers 2>headers.txt

    # headers.txt starts like this (truncated: a real block lists every header the
    # response carried, and one block follows per API call)
    < GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
    < Api-Supported-Versions: 1.0
    < Content-Type: application/json; charset=utf-8
    < Date: Mon, 01 Jan 2024 00:00:00 GMT
```

### Help Output (`go-eloverblik --help`)

The root help carries the rules an agent with only the binary needs (token, dates, IDs,
output), then lists the commands grouped by API type in a two-level tree. Every command below
the root, the customer and thirdparty groups included, prints cobra's full help instead; for
a command that is its usage line with the arguments, a Long text with the endpoint it calls
and the shape of its output, examples, its flags with their defaults, and the global flags
such as --token. The exact output of `go-eloverblik --help`:

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

`customer add-relation-by-code` and `customer delete-relation` still exist but are hidden:
Energinet retired their endpoints with DataHub 3.0, and both fail with ErrorEndpointRetired
without calling the API.

### Command Flags

```yaml
customer installations:
  --include-all: bool, default false. Merge in metering points not actively linked to the
                 user. Without CPR consent this fails with error 10007.

customer|thirdparty timeseries:
  --from, --to, --period: see "Date Specification"
  --aggregation: string, default "Hour". One of Actual, Quarter, Hour, Day, Month, Year.
  --flatten: bool, default false. Emit a JSON object keyed by metering point ID whose values
             are []FlatTimeSeriesPoint, instead of the raw nested document.

customer export-timeseries:
  --from, --to, --period, --aggregation: as above
  --format: string, default "csv". "csv" streams the API's CSV unchanged; "json" converts it
            (see "CSV to JSON Conversion"); any other value gives the CSV.

customer export-masterdata, customer export-charges:
  --format: string, default "csv" ("csv" or "json", as above)

customer|thirdparty charge-links:
  --from, --to, --period: see "Date Specification". --to is EXCLUSIVE here too. The bounds
          are sent as timestamps in Copenhagen time, so 2026-07-01 goes out as
          2026-07-01T00:00:00+02:00, now and now-30d keep the time of day, and a this_*
          period runs up to now, today included.
  status: the command is registered on both APIs and the flags are accepted, but the call
          always fails today with a 404 on both APIs, which both OpenAPI documents describe
          as the Charges integration feature being disabled (checked 2026-07-13). Use
          `charges` instead, accepting that it carries only currently valid and future prices.

token:
  --data-access: bool, default false. Exchange the refresh token for a data access token and
                 decode that one instead (this makes one request). The client to use is read
                 off the token's own type, so the API does not have to be named.
```

### Date Specification (--from / --to / --period)

The `timeseries`, `export-timeseries` and `charge-links` commands support two mutually
exclusive ways to specify date ranges. Remember `--to` is EXCLUSIVE (see "Date Semantics").

```yaml
Option A - Explicit dates with --from and --to:
  --from: Required start date (inclusive)
  --to: End date, exclusive (defaults to today's date in Copenhagen, whatever the host's
        zone, so the range ends with yesterday)
  Formats, all read in Copenhagen time (Europe/Copenhagen) whatever the host's zone:
    - YYYY-MM-DD: Absolute date, midnight at the start of that day in Copenhagen (e.g., 2026-07-01)
    - now: Current date/time
    - now-Nd: N days ago (e.g., now-30d)
    - now-Nw: N weeks ago (e.g., now-4w)
    - now-Nm: N months ago (e.g., now-2m)
    - now-Ny: N years ago (e.g., now-1y)
    - now-N forms: counted back from the current time on the Copenhagen calendar

Option B - Predefined period with --period:
  --period: Named time range (cannot be combined with --from or --to)
  Values: yesterday, this_week, last_week, this_month, last_month, this_year, last_year
  Note: the period helper already returns an exclusive end date, computed in Copenhagen
        time; weeks start on Monday, and a this_* period is an error on its first day
        (a Monday, the 1st, 1 January), which has no complete day yet

Errors:
  - "--period cannot be used with --from or --to"
  - "either --period or --from is required"
  - ErrorPeriodHasNoCompleteDay: "this_week: period started today and has no complete day yet"

Examples:
  - --from=2026-07-01 --to=2026-07-04   # 1, 2 and 3 July - the 4th is NOT included
  - --from=now-30d                       # Last 30 days (--to defaults to today)
  - --from=now-1y --to=now-6m            # Relative range
  - --period=last_month                  # Predefined period, ends on the 1st of this month
```

### CLI to Library Mapping

```yaml
- CLI: go-eloverblik token --token=$TOKEN
  Library: eloverblik.ParseToken(token)
  Returns: JSON object of TokenClaims. No request is made.

- CLI: go-eloverblik token --token=$TOKEN --data-access
  Library: |
    claims, _ := eloverblik.ParseToken(token)
    api, _ := claims.APIType()
    client := eloverblik.NewCustomer(token) // or NewThirdParty, per api
    claims, _ = client.DataAccessTokenClaims()
  Returns: JSON object of the data access token's claims

- CLI: go-eloverblik customer installations
  Library: client.GetMeteringPoints(false)   # --include-all maps to true
  Returns: JSON array of metering points

- CLI: go-eloverblik customer details 571313155411053087
  Library: client.GetMeteringPointDetails([]string{"571313155411053087"})
  Returns: JSON array with detailed information

- CLI: go-eloverblik customer timeseries 571313155411053087 --from=2026-07-01 --to=2026-07-04
  Library: |
    cph, _ := time.LoadLocation("Europe/Copenhagen") // the package embeds the zone database, so this works on every platform
    from, _ := time.ParseInLocation(time.DateOnly, "2026-07-01", cph)
    to, _ := time.ParseInLocation(time.DateOnly, "2026-07-04", cph)   // exclusive
    client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
  Returns: JSON with the nested time series document

- CLI: go-eloverblik customer timeseries 571313155411053087 --from=now-30d --flatten
  Library: |
    cph, _ := time.LoadLocation("Europe/Copenhagen")
    now := time.Now().In(cph)
    from := now.AddDate(0, 0, -30)
    to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cph) // --to default: today's Copenhagen date, exclusive
    tss, err := client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
    // then, once err is nil, for each ts in tss: skip it if ts.Err() != nil, else ts.Flatten()
  Returns: JSON object, metering point ID -> []FlatTimeSeriesPoint. A metering point that
           failed on its own is left out and reported as a warning on stderr; one that comes
           back once per access period gets the points of every period.

- CLI: go-eloverblik customer timeseries 571313155411053087 --period=last_month
  Library: |
    from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
    client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
  Returns: JSON with the nested time series document

- CLI: go-eloverblik customer charges 571313155411053087
  Library: client.GetCustomerCharges([]string{"571313155411053087"})
  Returns: JSON array of current and future charges (never historic ones)

- CLI: go-eloverblik thirdparty charges 571313155411053087
  Library: client.GetThirdPartyCharges([]string{"571313155411053087"})
  Returns: JSON array of current and future charges (no fees list, never historic ones)

- CLI: go-eloverblik customer charge-links 571313155411053087 --period=last_month
       go-eloverblik thirdparty charge-links 571313155411053087 --period=last_month
  Library: |
    from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
    client.GetChargeLinksWithCharges([]string{"571313155411053087"}, from, to)
  Returns: nothing today - BOTH the customer and the third-party API answer 404 for
           getchargelinkswithcharges, which both OpenAPI documents describe as the Charges
           integration feature being disabled (checked 2026-07-13 with valid tokens of each
           kind, while `charges` answered 200 on the same tokens). Once enabled it returns JSON
           with the charge links per metering point and the dated price series of the
           charges. Reach for `charges` in the meantime.

- CLI: go-eloverblik customer export-timeseries 571313155411053087 --from=2026-07-01 --format=json
  Library: |
    stream, err := client.ExportTimeSeries(...) // the API's CSV; close it when done
    // The library stops here: it returns CSV. The JSON is the CLI's own conversion of that
    // CSV (see "CSV to JSON Conversion"); the API has no JSON export.
  Returns: JSON array of objects keyed by the Danish CSV header, every value a string

- CLI: go-eloverblik thirdparty authorizations
  Library: client.GetAuthorizations()
  Returns: JSON array of authorization grants

- CLI: go-eloverblik thirdparty metering-points authorizationId <authorization id>
  Library: client.GetMeteringPointsForScope(eloverblik.AuthScopeID, "<authorization id>")
  Returns: JSON array of ThirdPartyMeteringPoint

- CLI: go-eloverblik customer alive
  Library: client.IsAlive()
  Returns: a human-readable line on stdout (not JSON)
```

### CSV to JSON Conversion

When using export commands with `--format=json`:
```yaml
Input: CSV stream with semicolon delimiter, UTF-8 BOM, Danish headers
Process:
  1. Drop the leading UTF-8 byte order mark, so the first key is the header's own name
     ("MålepunktsID", not "\uFEFFMålepunktsID")
  2. Parse CSV with ';' delimiter (LazyQuotes, TrimLeadingSpace, variable field count)
  3. Read header row as object keys
  4. Read data rows as values (every value stays a string, e.g. "0,198" with a decimal comma)
  5. Output an indented JSON array of objects
Output: JSON array where each object represents one CSV row; [] (never null) when the CSV
        has a header and no rows
Done by: the CLI only. The library's export methods return the API's CSV as it is.
```

## Common Patterns

### Pattern 1: Get All Consumption Data
```go
// 1. Create client
client := eloverblik.NewCustomer(token)

// 2. Get metering points
points, err := client.GetMeteringPoints(false)
if err != nil {
    return err
}

// 3. For each point, get time series
for _, point := range points {
    from := time.Now().AddDate(0, -1, 0) // 1 month ago
    to := time.Now()                     // exclusive

    ts, err := client.GetTimeSeries(
        []string{point.MeteringPointID},
        from, to,
        eloverblik.Day,
    )
    if err != nil {
        log.Printf("skipping %s: %v", point.MeteringPointID, err)
        continue
    }

    for _, series := range ts {
        if err := series.Err(); err != nil { // a metering point can fail on its own
            log.Printf("skipping: %v", err)
            continue
        }
        for _, p := range series.Flatten() {
            // p.From, p.To, p.Measurement, p.Unit, p.Quality, p.Resolution
            _ = p
        }
    }
}
```

### Pattern 2: Third-Party Multi-Customer Access
```go
// 1. Create client
client := eloverblik.NewThirdParty(token)

// 2. Get all authorizations
auths, err := client.GetAuthorizations()
if err != nil {
    return err
}

// 3. Pool the metering point IDs of every authorization
var ids []string
for _, auth := range auths {
    authIDs, err := client.GetMeteringPointIDsForScope(
        eloverblik.AuthScopeID, // per authorization; customerKey is optional and may be empty
        auth.ID,
    )
    if err != nil {
        return err
    }
    ids = append(ids, authIDs...)
}
slices.Sort(ids)
ids = slices.Compact(ids) // in case two authorizations cover the same metering point

// 4. Get data for all of them, 10 at a time (see Pattern 7): pooled, customers with only
//    one or two metering points share a request instead of costing one each
for batch := range slices.Chunk(ids, 10) {
    series, err := client.GetTimeSeries(batch, from, to, eloverblik.Hour)
    if err != nil {
        return err
    }
    for _, ts := range series {
        if err := ts.Err(); err != nil { // a metering point can fail on its own
            log.Printf("skipping: %v", err)
            continue
        }
        for _, p := range ts.Flatten() {
            fmt.Println(ts.ID, p.From.Format(time.RFC3339), p.Measurement, p.Unit)
        }
    }
}
```

### Pattern 3: Export and Parse CSV Data
```go
// 1. Export as CSV stream (Customer API only)
stream, err := client.ExportTimeSeries(ids, from, to, eloverblik.Hour)
if err != nil {
    return err
}
defer stream.Close()

// 2. Drop the UTF-8 byte order mark the CSV starts with, as the CLI does; left in, it
//    becomes part of the first header ("\uFEFFMålepunktsID")
body := bufio.NewReader(stream)
if mark, _, _ := body.ReadRune(); mark != '\uFEFF' {
    _ = body.UnreadRune()
}

// 3. Parse CSV
reader := csv.NewReader(body)
reader.Comma = ';'
reader.LazyQuotes = true
reader.TrimLeadingSpace = true
reader.FieldsPerRecord = -1 // rows may vary in width, as in the CLI

// 4. Read headers
headers, err := reader.Read()
if err != nil {
    return err
}

// 5. Read data
for {
    record, err := reader.Read()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    row := make(map[string]string)
    for i, value := range record {
        if i < len(headers) {
            row[headers[i]] = value
        }
    }
    fmt.Println(row["MålepunktsID"], row["Fra_dato"], row["Mængde"]) // strings, decimal comma
}
```

### Pattern 4: Use Predefined Periods
```go
// GetDatesFromPeriod returns an EXCLUSIVE to, so it can be passed straight through.
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    // an invalid period name, or a this_* period on its first day, which wraps
    // eloverblik.ErrorPeriodHasNoCompleteDay (check with errors.Is and fall back)
    log.Fatal(err)
}

ts, err := client.GetTimeSeries(
    []string{"571313155411053087"},
    from, to,
    eloverblik.Day,
)
if err != nil {
    return err
}
fmt.Println(len(ts), "time series") // check each one's Err() before Flatten()
```

### Pattern 5: Read a Token's Claims Before Doing Anything
Answer "which API is this token for, has it expired, which roles does it carry" without
spending an API call or guessing the client type.

```go
claims, err := eloverblik.ParseToken(os.Getenv("ELO_TOKEN"))
if err != nil {
    log.Fatalf("not an Eloverblik token: %v", err)
}
if claims.IsExpired() {
    log.Fatalf("token expired at %s - generate a new one in the Eloverblik portal", claims.ExpiresAt)
}
if !claims.IsRefreshToken() {
    log.Fatal("this is a data access token; the constructors want the long lived refresh token")
}
log.Printf("token %q, roles %v, valid for another %s", claims.TokenName, claims.Roles, claims.ExpiresIn())

// Build the right client from the token itself, instead of asking the user.
api, err := claims.APIType()
if err != nil {
    log.Fatal(err) // token type says neither customer nor thirdparty
}

var client eloverblik.Client = eloverblik.NewCustomer(os.Getenv("ELO_TOKEN"))
if api == eloverblik.ThirdPartyApi {
    client = eloverblik.NewThirdParty(os.Getenv("ELO_TOKEN"))
}
_ = client // hand it to the code that makes the calls

// The roles the token carries, e.g. ReadPrivate, ReadBusiness. Energinet does not document
// what each grants; the names suggest private customers and CVR registered ones (unverified).
for _, role := range claims.Roles {
    log.Println("role:", role)
}
```
CLI equivalent: `go-eloverblik token --token=$TOKEN` (add `--data-access` to decode the short
lived token the client would fetch, which does make one request).

### Pattern 6: Debug a Failing Call with Response Headers
The API's own diagnostics live in the response headers: `Api-Supported-Versions` tells whether
the pinned api-version is still served, `Retry-After`, when the API sends one, tells how long
a 429 wants you to wait, and the request URL in the block tells exactly which path was hit
(this is how the `getchargelinkswithcharges` 404 was checked on both APIs, on every
documented path).

```go
// Library: send the header blocks anywhere - stderr, a file, a bytes.Buffer in a test.
var debug bytes.Buffer
client := eloverblik.NewCustomer(token, eloverblik.WithResponseHeaderOutput(&debug))

if _, err := client.GetChargeLinksWithCharges(ids, from, to); err != nil {
    log.Printf("call failed: %v\nheaders seen:\n%s", err, debug.String())
}
```
```bash
# CLI: headers to stderr, JSON to stdout - so they can be separated
go-eloverblik thirdparty charge-links 571313155411053087 \
    --token=$TOKEN --period=last_month --print-response-headers \
    1>data.json 2>headers.txt
```
The token call appears first in the output, then one block per API call, in the order they were
made. A retried call prints one block per attempt, so a 429 followed by a 200 is visible.

### Pattern 7: Respect the Rate Limits (batch 10, let the client retry)
The API allows 120 calls per minute per IP (and only 2 to `/token`), and recommends at most 10
metering points per request. Batch the IDs and leave the default retry policy alone: it already
retries a 429 or 503 twice, honouring `Retry-After` when present (waiting at least 5s and at
most 60s).

```go
const batchSize = 10 // the API's recommended maximum

// Reuse ONE client: it fetches one data access token, caches it and renews it only shortly
// before it expires (about once a day), so a loop over 500 metering points costs one /token
// call, staying under the 2/minute limit.
client := eloverblik.NewCustomer(token) // default retry: 2 retries (3 attempts) on 429/503, Retry-After honoured

batches := slices.Collect(slices.Chunk(allIDs, batchSize))
for i, waits := 0, 0; i < len(batches); i++ {
    details, err := client.GetMeteringPointDetails(batches[i])
    if err != nil {
        // The 429/503 retries are already exhausted at this point; back off and retry
        // this batch, or stop.
        if errors.Is(err, eloverblik.ErrorTooManyRequests) && waits < 3 {
            waits++
            time.Sleep(time.Minute)
            i-- // the same batch again
            continue
        }
        return err
    }
    waits = 0
    for _, detail := range details {
        if !detail.Success { // per metering point failures do not fail the batch
            log.Printf("%s: [%d] %s", detail.ID, detail.ErrorCode, detail.ErrorText)
            continue
        }
        fmt.Println(detail.Result.MeteringPointID, detail.Result.TypeOfMP)
    }
}
```
Do NOT create a client per batch: each new client fetches its own data access token, and the
`/token` limit allows only two of those a minute. Do NOT pass `WithoutRetry()` in a batch
loop unless you implement the backoff yourself.

### Pattern 8: Price Historic Consumption (join time series to charge-links prices)

!! THIS PATTERN CANNOT RUN TODAY. It depends on `GetChargeLinksWithCharges`, which answers
**404 on the Customer API and on the Third-Party API alike**: both OpenAPI documents describe
that 404 as "When the Charges integration feature is disabled", and that is what the live API
answered when checked on 2026-07-13 with valid tokens of both kinds, in a session where
`getcharges` answered 200. The pattern is kept because the client implements the endpoint as
both OpenAPI documents specify it (one interval for every metering point aside) and it will
work the day Energinet enables the feature. **Until then there is no way to price historic
consumption through this API.** The closest available data is
`GetCustomerCharges`/`GetThirdPartyCharges` (CLI: `charges`), which only returns the charges
linked to the metering point now or from a future date, at their current prices: past and
future price changes of an existing charge are not included. That prices consumption going
forward only until a price changes, and never the past. If an agent is asked to price past consumption today, say so
rather than emitting code that will 404.

`GetCustomerCharges`/`GetThirdPartyCharges` only return the charges linked now or in the
future, at their current prices, so they cannot price the past. `GetChargeLinksWithCharges`
returns the dated price series. The join key is the `ChargeIdentifier` (Code + Owner + Type)
between a `ChargeLink` and a `ChargeInformation`. A present-tense `Charge`/`TariffCharge` (from
`GetCustomerCharges`/`GetThirdPartyCharges`) is expected to match its historic series by
`PriceID` = `ChargeIdentifier.Code` and `Owner` = `ChargeIdentifier.Owner`, with the list it
came from (subscriptions, fees, tariffs) somehow reflected in `ChargeIdentifier.Type`. That
`Type` is a type code (the constructed test fixture has `"D03"` for a tariff), and which code
means which kind of charge is undocumented. Neither OpenAPI document states this link, and the
endpoint has never answered, so it is unverified.

```go
from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth) // to is exclusive

// 1. The consumption
series, err := client.GetTimeSeries([]string{id}, from, to, eloverblik.Hour)
if err != nil {
    return err
}

// 2. The dated prices. On BOTH the customer and the third-party client this currently
//    returns a 404 error, because the Charges integration feature is disabled. There is no
//    substitute: GetCustomerCharges/GetThirdPartyCharges carry present and future prices
//    only, so the past cannot be priced until this call starts answering.
links, err := client.GetChargeLinksWithCharges([]string{id}, from, to)
if err != nil {
    return err
}

// 3. Index the charges by their identifier (the struct is comparable, so it is a valid key)
charges := make(map[eloverblik.ChargeIdentifier]eloverblik.ChargeInformation, len(links.ChargeInformations))
for _, info := range links.ChargeInformations {
    charges[info.ChargeIdentifier] = info
}

// 4. For every consumed interval, find the price valid in it and the link factor
var total float64
for _, ts := range series {
    if err := ts.Err(); err != nil { // a metering point can fail on its own
        return err
    }
    for _, point := range ts.Flatten() { // point.From, point.To, point.Measurement
        for _, result := range links.Results {
            if result.MeteringPointID != id || result.Error != "" {
                continue
            }
            for _, link := range result.ChargeLinks {
                info, ok := charges[link.ChargeIdentifier]
                if !ok {
                    continue
                }

                // The link must be active in the point's interval. A zero To is open ended.
                factor := 0
                for _, period := range link.ChargeLinkPeriods {
                    if !period.From.After(point.From) && (period.To.IsZero() || period.To.After(point.From)) {
                        factor = period.Factor
                    }
                }
                if factor == 0 {
                    continue
                }

                // The price is valid in [From, To). A zero To is open ended.
                for _, price := range info.ChargeSeriesPoints {
                    if !price.From.After(point.From) && (price.To.IsZero() || price.To.After(point.From)) {
                        total += point.Measurement * price.Price * float64(factor)
                        break
                    }
                }
                _ = info.TaxIndicator // true for taxes, e.g. to report them separately
            }
        }
    }
}
```
Notes:
- A subscription is not priced per kWh; `Factor` is the quantity and the price applies per
  period, not per point. Only multiply a tariff by `point.Measurement`. Which
  `PricingCategory` or `ChargeIdentifier.Type` code marks a subscription, a fee or a tariff is
  undocumented (the tariff in the constructed test fixture has `Type` `"D03"`).
- `ChargeInformation.Resolution` says how wide each `ChargeSeriesPoint` is (e.g. `PT1H` for an
  hourly tariff), which should match the aggregation the consumption was fetched with.
- Both the consumption points and the price points use half-open `[From, To)` intervals, so an
  interval boundary belongs to the later one. Compare with `!From.After(t) && To.After(t)`.

### Pattern 9: The Half-Open Range, and the Off-by-One It Prevents
```go
cph, err := time.LoadLocation("Europe/Copenhagen") // the package embeds the zone database: works on every platform
if err != nil {
    return err
}

// WRONG (the intuitive reading): "give me 1 July through 3 July"
from := time.Date(2026, 7, 1, 0, 0, 0, 0, cph)
to   := time.Date(2026, 7, 3, 0, 0, 0, 0, cph)
client.GetTimeSeries(ids, from, to, eloverblik.Day)
// -> returns 1 July and 2 July only. 3 July is silently missing.

// RIGHT: to is EXCLUSIVE, so pass the day AFTER the last one you want
from = time.Date(2026, 7, 1, 0, 0, 0, 0, cph)
to   = time.Date(2026, 7, 4, 0, 0, 0, 0, cph)
client.GetTimeSeries(ids, from, to, eloverblik.Day)
// -> returns exactly 1, 2 and 3 July.

// ALSO WRONG: "just yesterday" as a single day
from = time.Date(2026, 7, 3, 0, 0, 0, 0, cph)
to   = from
// -> API error 30002 (ErrorToDateCanNotBeEqualToFromDate). Equal dates are rejected, not
//    interpreted as one day.

// The safe route: let the helper do it. GetDatesFromPeriod already returns an exclusive to.
from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.Yesterday)
// from = 00:00 yesterday, to = 00:00 today, both in Copenhagen; err is nil for Yesterday
```

## Error Handling Patterns

### How errors surface
```yaml
transport failure: >-
                   returned as-is from the http client. The export calls (ExportTimeSeries,
                   ExportMasterdata, ExportCharges) wrap their export request's failure, transport
                   or non-2xx, as "failed to export <time series|masterdata|charges>: %w" (a failed
                   token fetch before it comes back unwrapped), so match with errors.Is / errors.As
non-2xx status:    |
                   ALWAYS an error. Which one depends on the body:
                   - "[code] message" with a known code -> that code's sentinel, whatever the
                     status (a 429 carrying [10004] is ErrorMaximumNumberOfMeteringPointsExceeded,
                     not ErrorTooManyRequests)
                   - an RFC 7807 problem document (what both specs declare for 400/401/403/404,
                     and what the charge-links endpoint answers its 404 with) ->
                     *eloverblik.APIError (StatusCode, Code, Title, Detail, TraceID, Errors; use
                     errors.As). Code holds a "[NNNNN]" code Detail opens with, known to the
                     client or not, else 0. It unwraps to the sentinel of a known code, else to
                     ErrorUnauthorized / ErrorEndpointRetired / ErrorTooManyRequests on a
                     401 / 410 / 429, and to nothing on any other status
                   - no usable body (empty, not JSON, or the no-error code 10000) ->
                     429 -> ErrorTooManyRequests, 401 -> ErrorUnauthorized,
                     410 -> ErrorEndpointRetired; any other status -> ErrorClientConnection(status),
                     a fresh error on every call that errors.Is cannot match
                   - a message without a code -> the 401/410/429 sentinel wrapped with the
                     message, or "failed to parse error in api error message <msg>" on any
                     other status
reading the body:  a parsed call (every call but the exports and IsAlive) reads the error body
                   only under a JSON Content-Type; under any other Content-Type, or when the
                   body is not well-formed JSON, the error is judged by its HTTP status alone,
                   as "no usable body" above. The exports read the body of a failed request
                   whatever its Content-Type, so a JSON string or problem document sent as
                   text/plain still maps to its sentinel or *APIError
API error message: the code is read only from exactly five digits in brackets at the start,
                   e.g. "[20010] Relation not found", and mapped to an exported sentinel error
                   (compare with errors.Is). "[20240101] ..." or "[20010" without its closing
                   bracket carries no code
unknown code:      >-
                   a five digit code the client has no sentinel for -> on a 401 / 410 / 429 the
                   status sentinel wrapped with the message, e.g. "unauthorized access: unhandled
                   error: '[99999] ...'", which errors.Is matches; on any other status
                   "unhandled error: '[99999] ...'", matching no sentinel
per-item failure:  a batch call still returns 200; the failing metering point carries
                   Success=false, ErrorCode and ErrorText in its own StatusResponse, and its
                   Err() returns an error that unwraps to the code's sentinel
retried:           429 and 503 only (twice by default). Never a 401, another 4xx, or a 500.
```

### Error codes worth special-casing
```yaml
10002 ErrorToManyRequestItems:                    its message reads "to many request items" (sic);
                                                  neither current OpenAPI document lists the code,
                                                  so when it is sent is unknown
10004 ErrorMaximumNumberOfMeteringPointsExceeded: a per-request metering-point maximum; neither
                                                  current OpenAPI document lists the code, and
                                                  the API documents no such limit as enforced
10007 ErrorNoCprConsent:                          GetMeteringPoints(true) without CPR consent
20010 ErrorRelationNotFound:                      per metering point - no relation to it
20012 ErrorUnauthorized:                          no active relation or authorization for the
                                                  metering point; also what a 401 without a
                                                  known code maps to
30002 ErrorToDateCanNotBeEqualToFromDate:         from == to; the range is half-open, see above
30014 ErrorPeriodNotAllowed:                      more than 730 days, or to not after from once
                                                  a future to is moved back to today. The old
                                                  name ErrorNumberOfDaysExcceded is the same error
30015 ErrorNoDataAvailable:                       per metering point - no readings in the period
30016 ErrorRelationHasExpired:                    per metering point - relation ended before or
                                                  during the period
30017 ErrorToDateCutOff:                          to is on or before the oldest date held
30018 ErrorMeteringPointDataNotAvailableForTheRequestedPeriod:
                                                  per metering point - the period lies outside
                                                  its data, e.g. from is before it was registered
40014 ErrorNoAuthorizationsFound:                 per metering point - no authorization covers it
50001 ErrorTokenNotValid:                         the refresh token is invalid or inactive
ErrorTooManyRequests:                             a 429 the retries could not absorb, when its
                                                  body carries no API error code the client knows
ErrorEndpointRetired:                             a 410, from AddRelationByWebAccessCode or
                                                  DeleteRelation
ErrorPeriodHasNoCompleteDay:                      not from the API; GetDatesFromPeriod returns it,
                                                  wrapped, for a this_* period on its first day
10000 ErrorNoError:                               deprecated and never returned; 10000 means
                                                  success and maps to no error
```

### Pattern 1: Check Both Error Returns and Every Result
```go
details, err := client.GetMeteringPointDetails(ids)
if err != nil {
    // Transport or whole-request API error - no results available
    return err
}

// Check each individual result
for _, detail := range details {
    if err := detail.Err(); err != nil {
        // API-level error for this specific ID, e.g.
        // "eloverblik: metering point 571313155411053087: 20010 RelationNotFound"
        log.Printf("skipping: %v", err)
        continue
    }
    fmt.Println(detail.Result.MeteringPointID, detail.Result.GridOperatorName) // use the result
}
```

### Pattern 2: Handle Empty Dates and Null Numbers
```go
// FlexibleTime fields may be zero values (the wire sends "" or null)
if !meteringPoint.ConsumerStartDate.IsZero() {
    startDate := meteringPoint.ConsumerStartDate.Time
    _ = startDate
}

// PowerLimitKWDecimal is a *float64: the API sends null for it on most metering points
if detail.Result.PowerLimitKWDecimal != nil {
    limit := *detail.Result.PowerLimitKWDecimal
    _ = limit
}

// ContactAddress.ProtectedAddress is a STRING on the wire ("False"), not a bool
protected := strings.EqualFold(addr.ProtectedAddress, "true")
_ = protected
```

### Pattern 3: Validate Metering Point IDs
```go
// An ID is 18 ASCII digits: check the text, do not parse a number. 18 digits do not fit
// in the int of a 32-bit platform (linux/arm, 386), where strconv.Atoi rejects every
// valid ID, and a parser would accept a leading + or - as well.
validateMeteringPointID := func(id string) error {
    if len(id) != 18 || strings.ContainsFunc(id, func(r rune) bool { return r < '0' || r > '9' }) {
        return fmt.Errorf("ID must be 18 digits: %q", id)
    }
    return nil
}
for _, id := range ids {
    if err := validateMeteringPointID(id); err != nil {
        return err
    }
}
// The CLI applies the same check to its positional arguments, plus a 1-10 count limit. The
// library checks neither, and leaves rejecting a malformed ID to the API (20003, 20004).
```

## Data Type Reference

### FlexibleTime
```go
// Declared in package eloverblik; an excerpt, not for pasting (the usage at the
// end is example code)
type FlexibleTime struct {
    time.Time
}

// Handles: empty string "", null, or an RFC3339 timestamp
// JSON unmarshaling:
//   "" or null → zero time.Time
//   "2024-01-01T00:00:00Z" → parsed time.Time (fractional seconds accepted)
//   anything time.Parse(time.RFC3339, ...) rejects, e.g. "2024-01-01" or a timestamp without
//   an offset → an error that fails the decode of the whole response
// JSON marshaling:
//   zero time.Time → null
//   valid time.Time → RFC3339 string

// Usage (ft is any FlexibleTime field, e.g. MeteringPoints.ConsumerStartDate):
if !ft.IsZero() {
    fmt.Println(ft.Format(time.RFC3339))
}
```

### Aggregation (what you request)
```go
// Declared in package eloverblik; an excerpt, not for pasting
type Aggregation string

const (
    Actual  Aggregation = "Actual"   // the meter's own reading resolution
    Quarter Aggregation = "Quarter"  // 15-minute aggregation
    Hour    Aggregation = "Hour"     // 60-minute aggregation
    Day     Aggregation = "Day"      // Daily total
    Month   Aggregation = "Month"    // Monthly total
    Year    Aggregation = "Year"     // Yearly total
)
```

### Resolution (what you get back)
```go
// Declared in package eloverblik; an excerpt, not for pasting
type Resolution string

const (
    PT15M Resolution = "PT15M" // Quarter
    PT1H  Resolution = "PT1H"  // Hour  (one Period per day, 24 points; 23 or 25 on a DST day)
    PT1D  Resolution = "PT1D"  // Day   (live API spelling; one Period per day, 1 point)
    P1M   Resolution = "P1M"   // Month (one Period per month, 1 point)
    PT1Y  Resolution = "PT1Y"  // Year  (live API spelling; one Period per year, 1 point)

    // The OpenAPI description spells the day and year resolutions differently, and adds PXD
    // for profiled quantities covering a variable number of days. All are accepted.
    P1D Resolution = "P1D"
    P1Y Resolution = "P1Y"
    PXD Resolution = "PXD"
)
```

### AuthorizationScope
```go
// Declared in package eloverblik; an excerpt, not for pasting
type AuthorizationScope string

const (
    AuthScopeID          AuthorizationScope = "authorizationId"
    AuthScopeCustomerCVR AuthorizationScope = "customerCVR"
    AuthScopeCustomerKey AuthorizationScope = "customerKey"
)
// These are the exact strings the CLI expects as the <scope> argument, too.
```

### apiType
```go
// Declared in package eloverblik; an excerpt, not for pasting
const (
    CustomerApi apiType = iota // the zero value
    ThirdPartyApi
)
// The type itself is unexported; the constants are exported and are what
// TokenClaims.APIType() returns. Not to be confused with the exported type APIType, which
// only the deprecated, inert package variable ApiType uses.
```

### Period
```go
// Declared in package eloverblik; an excerpt, not for pasting (the EXAMPLE at
// the end is example code)
type Period string

const (
    Yesterday Period = "yesterday"
    ThisWeek  Period = "this_week"
    LastWeek  Period = "last_week"
    ThisMonth Period = "this_month"
    LastMonth Period = "last_month"
    ThisYear  Period = "this_year"
    LastYear  Period = "last_year"
)

// FUNCTION: GetDatesFromPeriod
// SIGNATURE: GetDatesFromPeriod(period Period) (from time.Time, to time.Time, err error)
// PURPOSE: Convert a Period constant to concrete from/to time.Time values
// OUTPUTS:
//   - from (time.Time): start of the period, INCLUSIVE
//   - to (time.Time): EXCLUSIVE end - the start of the period that follows, or now for the
//     open ended "this_*" periods
//   - err (error): non-nil if the period string is not one of the constants, or, wrapping
//     ErrorPeriodHasNoCompleteDay, for this_week, this_month and this_year on their first day
//     (a Monday, the 1st of the month, 1 January), when from and to would fall on the same
//     date. from and to are zero on an error.
// BEHAVIOR:
//   - Weeks start on Monday (ISO 8601 and the Danish calendar)
//   - Period names are matched case-insensitively
//   - Everything is computed in Europe/Copenhagen, whatever the host's time zone: today is
//     the current Copenhagen date, and from and to are returned in Europe/Copenhagen, every
//     bound but a now at 00:00
// EXAMPLE:
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.ThisMonth)
if errors.Is(err, eloverblik.ErrorPeriodHasNoCompleteDay) {
    from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.LastMonth) // today is the 1st
}
if err != nil {
    return err
}
ts, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)
if err != nil {
    return err
}
fmt.Println(len(ts), "time series")
```

## Constraints and Limits

```yaml
Metering Point IDs:
  format: Numeric string
  length: Exactly 18 digits
  example: "571313155411053087"
  batch_size: 1-10 IDs per request is the API's usage recommendation ("Bundle requests for 10
              metering points at a time"), not a documented limit. The technical description
              (v2.0 history) dropped the per-request metering-point limit as "not implemented".
              Whether the API rejects more than 10 today is unverified. The library does not
              check the count; the CLI accepts 1 to 10 IDs per command.

Date Ranges:
  semantics: half-open [from, to). For GetTimeSeries/ExportTimeSeries this is at date granularity,
             and from == to (the same Copenhagen date) is rejected (error 30002)
  timezone: GetTimeSeries and ExportTimeSeries convert both bounds to Europe/Copenhagen and send them
            as YYYY-MM-DD, dropping the time of day. GetChargeLinksWithCharges sends them in the JSON
            body as RFC 3339 timestamps in Copenhagen time (spec format date-time). The time of day is
            kept and sub-seconds are dropped, so pass Europe/Copenhagen midnights there for whole
            days. A UTC midnight is sent as 02:00+02:00 in summer and 01:00+01:00 in winter.
            GetDatesFromPeriod builds Copenhagen midnights on any host, and the CLI reads a
            YYYY-MM-DD as a Copenhagen midnight
  maximum span: 730 days (error 30014, ErrorPeriodNotAllowed). eloverblik.MaximumDayRequestLeap
                = 730 and eloverblik.MaximumRequestDuration = 730 * 24h are exported for this.
  latest to:    tomorrow (error 30003 beyond it); a to of tomorrow is moved back to today

API Rate Limits:
  token endpoint: 2 calls / minute / IP
  all endpoints:  120 calls / minute / IP
  overall:        1200 calls / minute
  on breach:      429 (and 503 when DataHub is overloaded), both retried twice by default,
                  honouring Retry-After

Token Validity:
  refresh_token:     long lived (typically a year); read the exact expiry with ParseToken
  data_access_token: about 24 hours. Fetched on first use, cached, and renewed by the client on
                     the first call after it has expired or comes within 5 minutes of expiring
                     (per its exp claim; none, null, zero or negative means it never expires), so
                     one client serves a long running process. Concurrent callers share one
                     /token request and its outcome. A failed renewal of a still valid token
                     returns the cached token. A 401 on a data call is returned as it is, and
                     neither triggers a renewal nor a retry.
```

## CLI Output Formats

```yaml
Default: JSON (via encoding/json) on stdout
alive: a human-readable line, not JSON
Export Commands:
  --format=csv: Semicolon-delimited, UTF-8 BOM, Danish headers (streamed straight through)
  --format=json: Converted by the CLI from the CSV to an indented JSON array of objects keyed
                 by the header, every value a string; the BOM is dropped, and no rows give []
--print-response-headers: header blocks on stderr, so stdout stays parseable
```

## Testing Patterns

```yaml
Coverage:
  v1 package:  93.6% of statements
  cmd package: 67.7% of statements

Test Files:
  - v1/*_test.go intercept the resty transport with httpmock (options_test.go also runs httptest servers);
    cmd/*_test.go swap in mock Client/Customer implementations through clientInstance.
    No test reaches api.eloverblik.dk; the response fixtures of the endpoints that answer come
    from live API responses, while the charge-links fixture is constructed, since that endpoint
    has never answered
  - Both success and error paths are covered, including non-2xx statuses with empty bodies,
    the retry policy, the header printer, token claim decoding and renewal, and the half-open
    periods
  - v1/example_test.go holds the godoc examples; go test runs the two with a fixed output
    (ParseToken and StatusResponse.Err), and none of the examples reaches Eloverblik

Run Tests:
  command: go test ./...
  with_coverage: go test -coverprofile=coverage.out ./...
  with_race: go test -race -coverprofile=coverage.out -covermode=atomic ./...
  all checks: AGENTS.md (CLAUDE.md is a symlink to it) lists every check CI runs and how to run it locally
```

## Complete Working Example

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
    // 1. Get the refresh token from the environment
    token := os.Getenv("ELO_TOKEN")
    if token == "" {
        log.Fatal("ELO_TOKEN environment variable required")
    }

    // 2. Look at the token before spending a call on it
    claims, err := eloverblik.ParseToken(token)
    if err != nil {
        log.Fatalf("not an Eloverblik token: %v", err)
    }
    if claims.IsExpired() {
        log.Fatalf("token expired at %s", claims.ExpiresAt)
    }
    log.Printf("token %q for %v, expires in %s", claims.TokenName, claims.Roles, claims.ExpiresIn())

    // 3. Create the client (add WithResponseHeaderOutput(os.Stderr) when debugging)
    client := eloverblik.NewCustomer(token)

    // 4. Check API health (no token used)
    alive, err := client.IsAlive()
    if err != nil || !alive {
        log.Fatal("API not available")
    }

    // 5. Get metering points
    points, err := client.GetMeteringPoints(false)
    if err != nil {
        log.Fatal(err)
    }
    if len(points) == 0 {
        log.Fatal("No metering points found")
    }
    fmt.Printf("Found %d metering points\n", len(points))

    // 6. Get the first point's details
    firstID := points[0].MeteringPointID
    details, err := client.GetMeteringPointDetails([]string{firstID})
    if err != nil {
        log.Fatal(err)
    }
    if !details[0].Success {
        log.Fatalf("Details failed: [%d] %s", details[0].ErrorCode, details[0].ErrorText)
    }
    fmt.Printf("Grid operator: %s\n", details[0].Result.GridOperatorName)

    // 7. Get last month's consumption. to is exclusive, the helper already accounts for it.
    from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
    if err != nil {
        log.Fatal(err)
    }

    ts, err := client.GetTimeSeries([]string{firstID}, from, to, eloverblik.Day)
    if err != nil {
        log.Fatal(err)
    }

    // 8. Flatten and total it
    var total float64
    for _, series := range ts {
        if err := series.Err(); err != nil { // a metering point can fail on its own
            log.Printf("skipping: %v", err)
            continue
        }
        for _, p := range series.Flatten() {
            total += p.Measurement
            fmt.Printf("%s: %.3f %s (%s)\n",
                p.From.Format(time.DateOnly), p.Measurement, p.Unit, p.Quality)
        }
    }
    fmt.Printf("\nTotal consumption: %.2f kWh\n", total)

    // 9. Get the dated prices for the same period, to price that consumption.
    //    NOTE: this step cannot succeed today. Both APIs answer 404 for
    //    getchargelinkswithcharges, which both OpenAPI documents describe as the Charges
    //    integration feature being disabled (checked 2026-07-13), so the error is logged and
    //    the example ends instead of dying. The only alternative is
    //    client.GetCustomerCharges(...), which gives present and future prices - not the
    //    historic ones this step needs.
    links, err := client.GetChargeLinksWithCharges([]string{firstID}, from, to)
    if err != nil {
        log.Printf("charge links unavailable (expected until Energinet enables the feature): %v", err)
        return
    }
    for _, info := range links.ChargeInformations {
        for _, price := range info.ChargeSeriesPoints {
            fmt.Printf("%s..%s  %.4f  tax=%t\n",
                price.From.Format(time.DateOnly), price.To.Format(time.DateOnly),
                price.Price, info.TaxIndicator)
        }
    }
}
```

## Example: Fetch Hourly Data for the Past 7 Days (Go Library)

Use this as a reference when generating Go code that consumes this package.

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    eloverblik "github.com/slimcdk/go-eloverblik/v1"
)

func main() {
    // Create a Customer API client (use NewThirdParty for business access)
    client := eloverblik.NewCustomer(os.Getenv("ELO_TOKEN"))

    // Option A: specify dates manually. to is EXCLUSIVE.
    from := time.Now().AddDate(0, 0, -7)
    to := time.Now()

    // Option B: use a predefined period helper, which returns an exclusive to
    // from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastWeek)

    // Fetch hourly time series for one or more metering point IDs (at most 10 per request,
    // as Energinet recommends; the library does not check)
    ts, err := client.GetTimeSeries(
        []string{"571313155411053087"},
        from, to,
        eloverblik.Hour, // Aggregation: Actual, Quarter, Hour, Day, Month, Year
    )
    if err != nil {
        log.Fatal(err)
    }

    // Flatten the nested API response into half-open [From, To) intervals with a measurement.
    // Hour comes back as one Period per day holding 24 points (23 or 25 on a DST day). Flatten
    // steps through them by elapsed time, one hour per point from the period start, so each
    // of the points the API sent for a DST day lands at its true instant.
    for _, series := range ts {
        if err := series.Err(); err != nil { // a metering point can fail on its own
            log.Printf("skipping: %v", err)
            continue
        }
        for _, point := range series.Flatten() {
            fmt.Printf("%s -> %s  %.3f %s (quality: %s, resolution: %s)\n",
                point.From.Format(time.RFC3339),
                point.To.Format(time.RFC3339),
                point.Measurement,
                point.Unit,
                point.Quality,
                point.Resolution,
            )
        }
    }

    // All API responses serialize to JSON
    out, _ := json.MarshalIndent(ts, "", "  ")
    fmt.Println(string(out))
}
```

## AI Agent Implementation Checklist

When implementing an Eloverblik client:

- [ ] Store the refresh token securely (environment variable or secure vault)
- [ ] `ParseToken` it first: check `IsExpired()`, read `APIType()` to pick Customer vs ThirdParty
- [ ] Reuse ONE client - it caches its data access token, and /token allows only 2 calls/minute
- [ ] Let the client renew the ~24h data access token: it does so before expiry, so a long
      running process keeps the same client, shared between goroutines
- [ ] Treat `to` as EXCLUSIVE; never pass `from == to` (error 30002)
- [ ] Build dates as Copenhagen midnights; expect `ErrorPeriodHasNoCompleteDay` from
      `GetDatesFromPeriod` for a this_* period on its first day, and fall back
- [ ] Stay within 730 days per request (error 30014, ErrorPeriodNotAllowed)
- [ ] Pool metering point IDs (across authorizations, for a third party) and send them 10 per
      request, as Energinet recommends; the library sends whatever it is given
- [ ] Leave the default retry policy on (429/503, Retry-After honoured), or implement backoff
- [ ] Handle both the `error` return and every result's `Err()` (charge links: the `Error`
      string of each `Results[]` entry); a metering point can fail on its own inside a 200
      (30015, 30016, 30018, 40014 ...)
- [ ] Do not key results by metering point ID alone: an ID repeats once per access period
- [ ] Do NOT call `AddRelationByWebAccessCode` or `DeleteRelation`: Energinet retired both
      with DataHub 3.0 and they fail with `ErrorEndpointRetired` (410)
- [ ] Validate metering point IDs (18 numeric digits)
- [ ] Handle `FlexibleTime` zero values with `IsZero()`, and `*float64` fields with a nil check
- [ ] Use `Flatten()` for time series; it yields From/To/Measurement, not Timestamp/Value
- [ ] Do not assume the resolution string: Day is PT1D, Year is PT1Y on the wire
- [ ] Do NOT build on `GetChargeLinksWithCharges` yet: it answers 404 on BOTH APIs, the
      Charges integration feature being disabled (checked 2026-07-13). It is the only endpoint that can price
      historic consumption, so that is currently impossible; use `GetCustomerCharges` /
      `GetThirdPartyCharges` for present and future prices and say plainly that the past
      cannot be priced
- [ ] Set `Comma = ';'` on the CSV reader for exports, and close the `io.ReadCloser`
- [ ] Turn on `WithResponseHeaderOutput` / `--print-response-headers` when a call misbehaves
- [ ] Parse CLI stdout as JSON, except `alive` (a text line) and the export commands (CSV unless
      `--format json`); header debug output and warnings go to stderr

## Quick Reference: Most Common Operations

```go
// Inspect a token without calling the API
claims, err := eloverblik.ParseToken(token)
if err != nil {
    return err
}
api, err := claims.APIType() // eloverblik.CustomerApi | eloverblik.ThirdPartyApi
if err != nil {
    return err
}
fmt.Println(api == eloverblik.ThirdPartyApi, claims.ExpiresAt)

// Get consumption for the last 30 days (to is exclusive)
client := eloverblik.NewCustomer(token)
points, err := client.GetMeteringPoints(false)
if err != nil {
    return err
}
if len(points) == 0 {
    return errors.New("no metering points")
}
id := points[0].MeteringPointID
from := time.Now().AddDate(0, 0, -30)
to := time.Now()
ts, err := client.GetTimeSeries([]string{id}, from, to, eloverblik.Hour)
if err != nil {
    return err
}
for _, series := range ts {
    if err := series.Err(); err != nil {
        continue // this metering point failed on its own
    }
    data := series.Flatten() // []FlatTimeSeriesPoint{From, To, Measurement, Unit, Quality, ...}
    fmt.Println(len(data), "points")
}

// Export to CSV (Customer API only)
stream, err := client.ExportTimeSeries([]string{id}, from, to, eloverblik.Hour)
if err != nil {
    return err
}
defer stream.Close()
// CLI equivalent with JSON conversion: --format=json

// Get current charges (currently valid and future charges only - never historic prices,
// so this cannot price consumption that already happened). This is the closest available
// data today, because the charge-links endpoint below answers 404.
charges, err := client.GetCustomerCharges([]string{id})
if err != nil {
    return err
}
for _, charge := range charges {
    for _, tariff := range charge.Result.Tariffs {
        fmt.Println(tariff.PriceID, tariff.Owner, tariff.Name) // Prices[].Position is a STRING
    }
}

// Get the dated price series, needed to price historic consumption.
// !! 404 ON BOTH APIS TODAY - the Charges integration feature is disabled (checked
//    2026-07-13). Implemented per spec, ready for the day Energinet enables it.
links, err := client.GetChargeLinksWithCharges([]string{id}, from, to)
if err != nil {
    // expected today: a 404 *eloverblik.APIError ("eloverblik: 404 Not Found (traceId ...)")
} else {
    for _, info := range links.ChargeInformations {
        for _, point := range info.ChargeSeriesPoints {
            _ = point // point.From, point.To, point.Price, info.TaxIndicator, info.Resolution
        }
    }
}

// Debug a failing call
debugClient := eloverblik.NewCustomer(token, eloverblik.WithResponseHeaderOutput(os.Stderr))
_ = debugClient

// Third-party: access all customers, by authorization ID (a customerKey may be empty)
thirdParty := eloverblik.NewThirdParty(token)
auths, err := thirdParty.GetAuthorizations()
if err != nil {
    return err
}
for _, auth := range auths {
    points, err := thirdParty.GetMeteringPointsForScope(
        eloverblik.AuthScopeID,
        auth.ID,
    )
    if err != nil {
        return err
    }
    fmt.Println(auth.CustomerName, len(points), "metering points") // pool the IDs, see Common Pattern 2
}
```
