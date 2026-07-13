# Eloverblik Go Client - AI-Optimized Documentation

This documentation is structured for AI agents to easily understand, parse, and implement the Eloverblik Go client library and CLI.

## Package Information

```yaml
package: github.com/slimcdk/go-eloverblik
import: github.com/slimcdk/go-eloverblik/v1
install: go get github.com/slimcdk/go-eloverblik/v1
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
  3. Adds the Authorization header to all requests
```
Note: the cached data access token is fetched once and kept for the life of the client. It
is NOT re-fetched when it expires (a data access token lasts about 24 hours), so a long
running process should create a new client, or check the expiry itself:
```go
claims, _ := client.DataAccessTokenClaims()
if claims.IsExpired() { /* build a new client */ }
```

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
both OpenAPI documents and implemented here, NOT because it works. Energinet has not
deployed `getchargelinkswithcharges`: verified 2026-07-13 with valid Customer AND
Third-Party tokens, the live API answers **404 on BOTH APIs**, while `getcharges` answered
200 on the same tokens in the same session. Do not generate code that depends on it
succeeding today; see its function block below for what to do instead.

Exported-but-inert: `Mode`, `ReleaseMode`, `TestMode` and `ApiType` are package variables
left over from an earlier design. They have no effect: both constructors always target
`https://api.eloverblik.dk`. Do not generate code that sets them.

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
           ThirdPartyID, Roles []string, LoginType, WebApp, Issuer, Audience,
           ExpiresAt time.Time (Europe/Copenhagen)]
  errors: not a JWT (not 3 dot separated parts), undecodable payload,
          "token carries no Eloverblik claims" when neither tokenType nor exp is present
  methods:
    - IsExpired() bool
    - ExpiresIn() time.Duration          // 0 once expired or when no exp claim
    - IsRefreshToken() bool              // TokenType contains "refresh"
    - IsDataAccessToken() bool           // TokenType contains "dataaccess"
    - APIType() (apiType, error)         // CustomerApi or ThirdPartyApi, read off TokenType

Client.RefreshTokenClaims:
  signature: client.RefreshTokenClaims() (TokenClaims, error)
  purpose: Claims of the refresh token the client was created with. No request.

Client.DataAccessTokenClaims:
  signature: client.DataAccessTokenClaims() (TokenClaims, error)
  purpose: Claims of the data access token, fetching one first if the client has none.

TokenType examples: "CUSTOMERAPI_Refresh", "THIRDPARTYAPI_Refresh",
                    "CustomerApiDataAccess", "ThirdPartyApiDataAccess"
Roles examples:     ReadPrivate, ReadBusiness (a comma separated claim, split into a slice)

CLI: go-eloverblik token --token=$TOKEN [--data-access]
```

### 5. Date Semantics (half-open range) - READ THIS BEFORE PICKING DATES

The API reads a requested range as **half-open at date granularity**: `[dateFrom, dateTo)`.
Verified against the live API:

```yaml
inclusive: dateFrom
exclusive: dateTo
granularity: date (the client formats both bounds with time.DateOnly in Europe/Copenhagen)
equal dates: rejected outright with API error 30002 (ErrorToDateCanNotBeEqualToFromDate)
worked example:
  request: from=2026-07-01, to=2026-07-04, aggregation=Day
  returns: exactly 1 July, 2 July and 3 July. 4 July is NOT returned.
implication:
  to include a final day D, pass to = D + 1 day.
maximum span: 730 days (API error 30014 beyond it; see MaximumDayRequestLeap)
```

`GetDatesFromPeriod` follows the same rule and returns an **exclusive** `to`, i.e. the start
of the period that follows:

```yaml
yesterday:  [start of yesterday, start of today)
this_week:  [start of week (Sunday), now)
last_week:  [start of last week, start of this week)
this_month: [1st of this month, now)
last_month: [1st of last month, 1st of this month)
this_year:  [1 January this year, now)
last_year:  [1 January last year, 1 January this year)
```

Both `from` and `to` are converted to `Europe/Copenhagen` before being formatted, so a UTC
`time.Time` near midnight can land on the neighbouring Danish date. Prefer building dates in
Copenhagen local time or at midday UTC.

### 6. Resolutions in the response (the OpenAPI description is wrong here)

The `Aggregation` you request is not the `resolution` string you get back. What the live wire
actually sends, per requested aggregation:

```yaml
Aggregation Day     -> resolution "PT1D"   one Period per day,   1 point per Period
Aggregation Month   -> resolution "P1M"    one Period per month, 1 point per Period
Aggregation Year    -> resolution "PT1Y"   one Period per year,  1 point per Period
Aggregation Hour    -> resolution "PT1H"   one Period per day,   24 points per Period
Aggregation Quarter -> resolution "PT15M"  15-minute points
Aggregation Actual  -> the meter's own reading resolution
```

The OpenAPI description documents `P1D` and `P1Y` instead, and adds `PXD` for profiled
energy quantities covering a variable number of days. The library declares and accepts **both
spellings**, so `Flatten()` is correct whichever the server sends:

```go
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
- Periods holding several points step by **calendar unit**, not by a fixed duration, so a
  daylight saving day of 23 or 25 hours lands on the right boundary.
- An unknown resolution attributes the whole period to the point rather than collapsing it
  to a zero-width interval.

### 7. Rate Limits (documented by the API, enforced by it)

```yaml
token endpoint: 2 calls per minute per IP
all endpoints:  120 calls per minute per IP
overall:        1200 calls per minute across all users
batch size:     10 metering points per request (recommended and enforced; error 10002/10004 beyond)
on breach:      HTTP 429 (and HTTP 503 when DataHub cannot keep up)
client policy:  429 and 503 are retried DefaultRetryCount (2) times, honouring Retry-After,
                capped at DefaultRetryMaxWait (60s). Nothing else is retried - a 401, any
                other 4xx and a 500 are returned to the caller immediately. Transport errors
                are not retried either, so a request is never sent twice by accident.
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
// EXAMPLE:
client := eloverblik.NewCustomer("your-refresh-token-here")
```

```go
// FUNCTION: NewThirdParty
// PURPOSE: Create Third-Party API client
// SIGNATURE: NewThirdParty(refreshToken string, opts ...Option) ThirdParty
// INPUTS:
//   - refreshToken (string): Obtained from eloverblik.dk portal
//   - opts (...Option): Optional client options
// OUTPUT: ThirdParty interface (embeds Client)
// EXAMPLE:
client := eloverblik.NewThirdParty("your-refresh-token-here")
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

// OUTPUT FORMAT (one block per response, header keys sorted alphabetically):
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
client := eloverblik.NewCustomer(token, eloverblik.WithoutRetry())
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
fmt.Println(claims.TokenName, claims.Roles, claims.ExpiresAt, claims.IsExpired())
```

```go
// METHODS: RefreshTokenClaims / DataAccessTokenClaims (on both clients)
// SIGNATURE: client.RefreshTokenClaims() (TokenClaims, error)   // no request
//            client.DataAccessTokenClaims() (TokenClaims, error) // fetches a token if needed
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
//   ConsumerStartDate (FlexibleTime), HasRelation (bool),
//   ChildMeteringPoints ([]ChildMeteringPoints)
// EXAMPLE:
points, err := client.GetMeteringPoints(false)
if err != nil { /* handle error */ }
for _, point := range points {
    meteringID := point.MeteringPointID // Use this ID for other API calls
}
```

```go
// FUNCTION: GetMeteringPointDetails  (Customer and ThirdParty)
// PURPOSE: Get detailed information about specific metering points
// SIGNATURE: GetMeteringPointDetails(meteringPointIDs []string) ([]MeteringPointDetailsResponse, error)
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 IDs, each exactly 18 digits, numeric only
// OUTPUTS:
//   - []MeteringPointDetailsResponse: one response per input ID
//   - error: transport error, or an API-level error that failed the whole request
// MeteringPointDetailsResponse = { Result MeteringPointDetail } + embedded StatusResponse:
//   - Success (bool): true if this specific point succeeded
//   - ErrorCode (int): 10000 = success, other = error
//   - ErrorText (string), ID (string), StackTrace (string)
// MeteringPointDetail FIELDS (54, all string unless noted):
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
// ERROR HANDLING:
//   Always check the per-result Success field, not just the error return
// EXAMPLE:
ids := []string{"571313155411053087", "571313155411782079"}
details, err := client.GetMeteringPointDetails(ids)
if err != nil { /* handle network / API error */ }
for _, detail := range details {
    if !detail.Success {
        log.Printf("Failed for %s: [%d] %s", detail.ID, detail.ErrorCode, detail.ErrorText)
        continue
    }
    gridOp := detail.Result.GridOperatorName
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
//   - meteringPointIDs ([]string): 1-10 metering point IDs
//   - from (time.Time): Start date, INCLUSIVE
//   - to   (time.Time): End date, EXCLUSIVE (see "Date Semantics"; equal dates -> error 30002)
//   - aggregation (Aggregation): Actual, Quarter, Hour, Day, Month, Year
// DATE HANDLING:
//   - Both bounds are converted to Europe/Copenhagen and formatted as YYYY-MM-DD
//   - Maximum span 730 days (error 30014 beyond it)
// OUTPUTS:
//   - []TimeSeries (one per metering point), error
// DATA STRUCTURE:
//   TimeSeries.MyEnergyDataMarketDocument.TimeSeries[].Periods[].Points[]
//   PeriodResponse: Resolution (string), TimeInterval {Start, End}, Points
//   PointResponse:  Position (int), OutQuantityQuantity (float64), OutQuantityQuality (string)
//   TimeSeries also embeds StatusResponse (Success, ErrorCode, ErrorText, ID, StackTrace)
// HELPER METHOD: ts.Flatten() []FlatTimeSeriesPoint - resolves each point to its real
//   [From, To) interval in Copenhagen local time. See "Resolutions".
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
    for _, p := range series.Flatten() {
        fmt.Printf("%s -> %s: %.3f %s (%s)\n",
            p.From.Format(time.RFC3339), p.To.Format(time.RFC3339),
            p.Measurement, p.Unit, p.Quality)
    }
}
```

```go
// TYPE: FlatTimeSeriesPoint (what Flatten() returns)
type FlatTimeSeriesPoint struct {
    From         time.Time  `json:"from"`         // inclusive, Europe/Copenhagen
    To           time.Time  `json:"to"`           // exclusive, Europe/Copenhagen
    Measurement  float64    `json:"measurement"`  // the quantity, e.g. kWh
    Quality      string     `json:"quality"`      // e.g. A04 (estimated), A05 (measured)
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
//   - meteringPointIDs ([]string): 1-10 metering point IDs
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
//             but Energinet has not deployed it (404 on both APIs, verified 2026-07-13), so
//             today this is the closest available data and historic pricing is not possible
//             through this API. PriceID is the stable key to join a charge to its price
//             series once the endpoint goes live.
// EXAMPLE:
charges, err := client.GetCustomerCharges([]string{"571313155411053087"})
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
// !!!! NOT DEPLOYED BY ENERGINET - THIS CALL RETURNS 404 ON BOTH APIS TODAY !!!!
//   Verified 2026-07-13 with a valid Customer token AND a valid Third-Party token, on every
//   documented path:
//     POST /customerapi/api/meteringpoints/meteringpoint/getchargelinkswithcharges -> 404
//     POST /thirdpartyapi/api/meteringpoint/getchargelinkswithcharges              -> 404
//   getcharges and getdetails returned 200 with the SAME tokens in the same session, so this
//   is NOT an auth problem: the route is declared in both OpenAPI documents but is not
//   served. This client implements the endpoint exactly as both specs describe it and is
//   ready for the day Energinet deploys it. Until then every call errors with a 404.
//
//   USE INSTEAD TODAY: GetCustomerCharges / GetThirdPartyCharges (CLI: the `charges`
//   commands). They return subscriptions, fees and tariffs, but ONLY those currently valid
//   or taking effect in the future - they CANNOT price consumption that already happened.
//   There is no other endpoint that can, so historic pricing is simply unavailable for now.
//   Do not generate code whose happy path depends on GetChargeLinksWithCharges succeeding.
//
// PURPOSE (what it will return once deployed): the charge links of metering points together
//          with the dated price series of every charge they link to - the missing half that
//          GetCustomerCharges/GetThirdPartyCharges cannot supply, i.e. the data needed to
//          price historic consumption.
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 metering point IDs
//   - from, to (time.Time): half-open [from, to), applied to every metering point.
//     The wire format takes an interval PER metering point; this client sends the same one
//     for all of them.
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
//                            ChargeInformation; comparable, so usable as a map key)
//   ChargeLinkPeriod:        Factor (int, e.g. number of subscriptions),
//                            From, To (FlexibleTime; To is zero for an open ended link)
//   ChargeInformation:       ChargeIdentifier, TaxIndicator (bool), Resolution (e.g. PT1H),
//                            PricingCategory, ChargeInformationPeriods, ChargeSeriesPoints
//   ChargeInformationPeriod: Name, Description, TransparentInvoicing (bool), From, To,
//                            VATClassification
//   ChargeSeriesPoint:       From, To, Price (float64; the price is valid in [From, To))
// EXAMPLE (this is the shape of the response once the endpoint is deployed; running it today
//          returns a 404 error on both the customer and the thirdparty client):
from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
links, err := client.GetChargeLinksWithCharges([]string{"571313155411053087"}, from, to)
if err != nil { /* today this is always the 404: fall back to the charges endpoint */ }

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
//   - meteringPointIDs ([]string): 1-10 IDs
//   - from, to (time.Time): half-open [from, to), same semantics as GetTimeSeries
//   - aggregation (Aggregation): Data granularity
// OUTPUTS:
//   - io.ReadCloser: CSV data stream (Danish format, semicolon-delimited). Caller closes.
//   - error: a non-2xx is reported as "failed to export time series, status: ..."
// CSV FORMAT:
//   - Delimiter: semicolon (;)
//   - Encoding: UTF-8 with BOM
//   - Headers: Danish language
//   - Columns: MålepunktsID, Fra_dato, Til_dato, Mængde, Måleenhed, Kvalitet, Type
// USAGE PATTERN:
stream, err := client.ExportTimeSeries(ids, from, to, eloverblik.Hour)
if err != nil { /* handle error */ }
defer stream.Close()
// Option 1: Write to file
file, _ := os.Create("output.csv")
io.Copy(file, stream)
// Option 2: Parse CSV
reader := csv.NewReader(stream)
reader.Comma = ';'
records, _ := reader.ReadAll()
```

```go
// FUNCTION: ExportMasterdata  (Customer only)
// PURPOSE: Export metering point master data as CSV
// SIGNATURE: ExportMasterdata(meteringPointIDs []string) (io.ReadCloser, error)
// OUTPUTS:
//   - io.ReadCloser: CSV stream with the master data columns. Caller closes.
//   - error: HTTP/network errors, or a non-2xx status
// EXAMPLE:
stream, err := client.ExportMasterdata([]string{"571313155411053087"})
defer stream.Close()
io.Copy(os.Stdout, stream)
```

```go
// FUNCTION: ExportCharges  (Customer only)
// PURPOSE: Export charges as CSV
// SIGNATURE: ExportCharges(meteringPointIDs []string) (io.ReadCloser, error)
// OUTPUTS:
//   - io.ReadCloser: CSV with subscriptions, fees, tariffs. Caller closes.
//   - error: HTTP/network errors, or a non-2xx status
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
for _, resp := range responses {
    if resp.Success {
        fmt.Println("Linked:", resp.Result)
    }
}
```

```go
// FUNCTION: AddRelationByWebAccessCode  (Customer only)
// PURPOSE: Link a metering point using a web access code
// SIGNATURE: AddRelationByWebAccessCode(meteringPointID, webAccessCode string) (string, error)
// INPUTS:
//   - meteringPointID (string): Single 18-digit ID
//   - webAccessCode (string): 8-digit code from letter/email
// EXAMPLE:
result, err := client.AddRelationByWebAccessCode("571313155411053087", "12345678")
```

```go
// FUNCTION: DeleteRelation  (Customer only)
// PURPOSE: Unlink a metering point from the authenticated user
// SIGNATURE: DeleteRelation(meteringPointID string) (bool, error)
// OUTPUTS:
//   - bool: the boolean the API returns in its envelope; falls back to the HTTP status
//   - error: e.g. ErrorRelationNotFound (API code 20010), which the API reports in the body
// NOTE: the HTTP status alone does not tell whether the relation was deleted, so the body
//       is read. Always check the error, not just the bool.
```

```go
// FUNCTION: IsAlive  (Customer and ThirdParty)
// PURPOSE: Health check for API availability
// SIGNATURE: IsAlive() (bool, error)
// INPUTS: none. NO TOKEN IS USED - it does not authenticate.
// OUTPUTS:
//   - bool: true if the /isalive endpoint answered 2xx
//   - error: transport errors only
// NOTE: each client probes its OWN host, so `customer alive` hits the customer API and
//       `thirdparty alive` hits the third-party API.
```

### Third-Party API Methods

```go
// FUNCTION: GetAuthorizations  (ThirdParty only)
// PURPOSE: List all customer authorizations granted to the third party
// SIGNATURE: GetAuthorizations() ([]Authorization, error)
// INPUTS: none
// Authorization FIELDS:
//   - ID (string): Authorization ID
//   - ThirdPartyName (string)
//   - ValidFrom, ValidTo (STRING, not parsed)
//   - CustomerName (string)
//   - CustomerCVR (string)
//   - CustomerKey (string): use with AuthScopeCustomerKey
//   - IncludeFutureMeteringPoints (bool)
//   - Timestamp (FlexibleTime)
// EXAMPLE:
auths, err := client.GetAuthorizations()
for _, auth := range auths {
    meteringPoints, _ := client.GetMeteringPointsForScope(
        eloverblik.AuthScopeCustomerKey,
        auth.CustomerKey,
    )
    _ = meteringPoints
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
//   1. GetAuthorizations() to get customer keys
//   2. For each auth, GetMeteringPointsForScope() to get their points
//   3. Use the metering point IDs with GetTimeSeries(), GetMeteringPointDetails(), etc.
// EXAMPLE:
points, err := client.GetMeteringPointsForScope(
    eloverblik.AuthScopeCustomerKey,
    "customer-key-from-authorization",
)
```

```go
// FUNCTION: GetMeteringPointIDsForScope  (ThirdParty only)
// PURPOSE: Get only the IDs (a smaller response than GetMeteringPointsForScope)
// SIGNATURE: GetMeteringPointIDsForScope(scope AuthorizationScope, identifier string) ([]string, error)
// USE CASE: When you only need IDs, not the metadata
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
//       endpoint for historic prices, but Energinet has not deployed it - it answers 404 on
//       BOTH APIs (verified 2026-07-13), so this is the closest data available today.
```

## CLI Command Patterns

### Command Structure
```
go-eloverblik --token=<refresh-token> <api-type> <command> [args] [flags]

Components:
  --token: Global flag, required for all commands
  --print-response-headers: Global flag, prints HTTP response headers to stderr (debugging)
  <api-type>: "customer" or "thirdparty" (the "token" command sits directly under the root)
  <command>: Action to perform
  [args]: Positional arguments (usually metering point IDs, 1-10, each 18 digits)
  [flags]: Optional command-specific flags
```

### Global Flags
```yaml
--token <string>:
  required: true (a persistent flag on the root command)
  purpose: The long lived Eloverblik REFRESH token from the portal, not a data access token.
           The client exchanges it for a data access token itself.

--print-response-headers:
  required: false
  default: false
  purpose: Print HTTP response headers from the Eloverblik API to stderr
  notes: Headers go to stderr, so stdout stays clean, parseable JSON
  example: |
    go-eloverblik customer details <metering-id> --token=$TOKEN --print-response-headers 2>headers.txt

    < GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
    < Api-Supported-Versions: 1.0
    < Content-Type: application/json; charset=utf-8
    < Date: Mon, 01 Jan 2024 00:00:00 GMT
```

### Help Output (`go-eloverblik --help`)

Commands are grouped by API type in a two-level tree:

```
A CLI for the Danish Eloverblik platform

Usage:
  go-eloverblik [command]

Available Commands:

  completion
    bash                     Generate the autocompletion script for bash
    fish                     Generate the autocompletion script for fish
    powershell               Generate the autocompletion script for powershell
    zsh                      Generate the autocompletion script for zsh

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
```

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
  --from, --to, --period, --aggregation (as above)
  --format: string, default "csv". "csv" streams the API's CSV; "json" converts it.

customer export-masterdata, customer export-charges:
  --format: string, default "csv" ("csv" or "json")

customer|thirdparty charge-links:
  --from, --to, --period: see "Date Specification"
  status: the command is registered on both APIs and the flags are accepted, but the call
          always fails today - Energinet has not deployed getchargelinkswithcharges and both
          APIs answer 404 (verified 2026-07-13). Use `charges` instead, accepting that it
          carries only currently valid and future prices.

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
  --to: End date, exclusive (defaults to today)
  Formats:
    - YYYY-MM-DD: Absolute date (e.g., 2026-07-01)
    - now: Current date/time
    - now-Nd: N days ago (e.g., now-30d)
    - now-Nw: N weeks ago (e.g., now-4w)
    - now-Nm: N months ago (e.g., now-2m)
    - now-Ny: N years ago (e.g., now-1y)

Option B - Predefined period with --period:
  --period: Named time range (cannot be combined with --from or --to)
  Values: yesterday, this_week, last_week, this_month, last_month, this_year, last_year
  Note: the period helper already returns an exclusive end date

Errors:
  "--period cannot be used with --from or --to"
  "either --period or --from is required"

Examples:
  --from=2026-07-01 --to=2026-07-04   # 1, 2 and 3 July - the 4th is NOT included
  --from=now-30d                       # Last 30 days (--to defaults to today)
  --from=now-1y --to=now-6m            # Relative range
  --period=last_month                  # Predefined period, ends on the 1st of this month
```

### CLI to Library Mapping

```yaml
CLI: go-eloverblik token --token=$TOKEN
Library: eloverblik.ParseToken(token)
Returns: JSON object of TokenClaims. No request is made.

CLI: go-eloverblik token --token=$TOKEN --data-access
Library: |
  claims, _ := eloverblik.ParseToken(token)
  api, _ := claims.APIType()
  client := eloverblik.NewCustomer(token) // or NewThirdParty, per api
  claims, _ = client.DataAccessTokenClaims()
Returns: JSON object of the data access token's claims

CLI: go-eloverblik customer installations
Library: client.GetMeteringPoints(false)   # --include-all maps to true
Returns: JSON array of metering points

CLI: go-eloverblik customer details 571313155411053087
Library: client.GetMeteringPointDetails([]string{"571313155411053087"})
Returns: JSON array with detailed information

CLI: go-eloverblik customer timeseries 571313155411053087 --from=2026-07-01 --to=2026-07-04
Library: |
  from, _ := time.Parse(time.DateOnly, "2026-07-01")
  to, _ := time.Parse(time.DateOnly, "2026-07-04")   // exclusive
  client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
Returns: JSON with the nested time series document

CLI: go-eloverblik customer timeseries 571313155411053087 --from=now-30d --flatten
Library: |
  from := time.Now().AddDate(0, 0, -30)
  to := time.Now()
  tss, _ := client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
  flat := tss[0].Flatten()
Returns: JSON object, metering point ID -> []FlatTimeSeriesPoint

CLI: go-eloverblik customer timeseries 571313155411053087 --period=last_month
Library: |
  from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
  client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
Returns: JSON with the nested time series document

CLI: go-eloverblik customer charges 571313155411053087
Library: client.GetCustomerCharges([]string{"571313155411053087"})
Returns: JSON array of current and future charges (never historic ones)

CLI: go-eloverblik thirdparty charges 571313155411053087
Library: client.GetThirdPartyCharges([]string{"571313155411053087"})
Returns: JSON array of current and future charges (no fees list, never historic ones)

CLI: go-eloverblik customer charge-links 571313155411053087 --period=last_month
     go-eloverblik thirdparty charge-links 571313155411053087 --period=last_month
Library: |
  from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
  client.GetChargeLinksWithCharges([]string{"571313155411053087"}, from, to)
Returns: nothing today - Energinet has not deployed getchargelinkswithcharges and BOTH the
         customer and the third-party API answer 404 for it (verified 2026-07-13 with valid
         tokens of each kind, while `charges` answered 200 on the same tokens). Once deployed
         it returns JSON with the charge links per metering point and the dated price series
         of the charges. Reach for `charges` in the meantime.

CLI: go-eloverblik customer export-timeseries 571313155411053087 --from=2026-07-01 --format=json
Library: |
  stream, _ := client.ExportTimeSeries(...)
  // Then convert CSV to JSON
Returns: JSON array of consumption records

CLI: go-eloverblik thirdparty authorizations
Library: client.GetAuthorizations()
Returns: JSON array of authorization grants

CLI: go-eloverblik thirdparty metering-points customerKey <key>
Library: client.GetMeteringPointsForScope(eloverblik.AuthScopeCustomerKey, "<key>")
Returns: JSON array of ThirdPartyMeteringPoint

CLI: go-eloverblik customer alive
Library: client.IsAlive()
Returns: a human-readable line on stdout (not JSON)
```

### CSV to JSON Conversion

When using export commands with `--format=json`:
```yaml
Input: CSV stream with semicolon delimiter, UTF-8 BOM, Danish headers
Process:
  1. Parse CSV with ';' delimiter (LazyQuotes, TrimLeadingSpace, variable field count)
  2. Read header row as object keys
  3. Read data rows as values
  4. Output an indented JSON array of objects
Output: JSON array where each object represents one CSV row
```

## Common Patterns

### Pattern 1: Get All Consumption Data
```go
// 1. Create client
client := eloverblik.NewCustomer(token)

// 2. Get metering points
points, _ := client.GetMeteringPoints(false)

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
        continue
    }

    for _, series := range ts {
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
auths, _ := client.GetAuthorizations()

// 3. For each authorized customer
for _, auth := range auths {
    // 4. Get their metering points
    points, _ := client.GetMeteringPointsForScope(
        eloverblik.AuthScopeCustomerKey,
        auth.CustomerKey,
    )

    // 5. Get data for their points, 10 at a time (see Pattern 7)
    var ids []string
    for _, p := range points {
        ids = append(ids, p.MeteringPointID)
    }

    for _, batch := range chunk(ids, 10) {
        ts, _ := client.GetTimeSeries(batch, from, to, eloverblik.Hour)
        _ = ts
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

// 2. Parse CSV
reader := csv.NewReader(stream)
reader.Comma = ';'
reader.LazyQuotes = true
reader.TrimLeadingSpace = true

// 3. Read headers
headers, _ := reader.Read()

// 4. Read data
for {
    record, err := reader.Read()
    if err == io.EOF {
        break
    }

    row := make(map[string]string)
    for i, value := range record {
        if i < len(headers) {
            row[headers[i]] = value
        }
    }
}
```

### Pattern 4: Use Predefined Periods
```go
// GetDatesFromPeriod returns an EXCLUSIVE to, so it can be passed straight through.
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    log.Fatal(err) // invalid period name
}

ts, err := client.GetTimeSeries(
    []string{"571313155411053087"},
    from, to,
    eloverblik.Day,
)
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

// Roles decide what the token may read: ReadPrivate for private customers,
// ReadBusiness for CVR registered ones.
for _, role := range claims.Roles {
    log.Println("role:", role)
}
```
CLI equivalent: `go-eloverblik token --token=$TOKEN` (add `--data-access` to decode the short
lived token the client would fetch, which does make one request).

### Pattern 6: Debug a Failing Call with Response Headers
The API's own diagnostics live in the response headers: `Api-Supported-Versions` tells whether
the pinned api-version is still served, `Retry-After` tells how long a 429 wants you to wait,
and the request URL in the block tells exactly which path was hit (this is how the
`getchargelinkswithcharges` 404 was confirmed on both APIs, on every documented path).

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
retries a 429 or 503 twice, honouring `Retry-After`.

```go
// Reuse ONE client: it fetches a single data access token and caches it, so a loop over 500
// metering points still costs exactly one /token call, staying under the 2/minute limit.
client := eloverblik.NewCustomer(token) // default retry: 2 attempts on 429/503, Retry-After honoured

func chunk(ids []string, size int) [][]string {
    var batches [][]string
    for start := 0; start < len(ids); start += size {
        end := min(start+size, len(ids))
        batches = append(batches, ids[start:end])
    }
    return batches
}

const batchSize = 10 // the API's recommended and enforced maximum

for _, batch := range chunk(allIDs, batchSize) {
    details, err := client.GetMeteringPointDetails(batch)
    if err != nil {
        // The 429/503 retries are already exhausted at this point; back off or stop.
        if errors.Is(err, eloverblik.ErrorTooManyRequests) {
            time.Sleep(time.Minute)
            continue
        }
        return err
    }
    for _, detail := range details {
        if !detail.Success { // per metering point failures do not fail the batch
            log.Printf("%s: [%d] %s", detail.ID, detail.ErrorCode, detail.ErrorText)
            continue
        }
        _ = detail.Result
    }
}
```
Do NOT create a client per batch: each new client fetches its own data access token and two of
those inside a minute already hit the `/token` limit. Do NOT pass `WithoutRetry()` in a batch
loop unless you implement the backoff yourself.

### Pattern 8: Price Historic Consumption (join time series to charge-links prices)

!! THIS PATTERN CANNOT RUN TODAY. It depends on `GetChargeLinksWithCharges`, which Energinet
has not deployed: the live API answers **404 on the Customer API and on the Third-Party API
alike** (verified 2026-07-13 with valid tokens of both kinds, in a session where `getcharges`
answered 200). The pattern is kept because the client implements the endpoint exactly as both
OpenAPI documents specify and it will work the day Energinet deploys it. **Until then there is
no way to price historic consumption through this API.** The closest available data is
`GetCustomerCharges`/`GetThirdPartyCharges` (CLI: `charges`), which only returns charges that
are currently valid or take effect in the future — enough to price consumption going forward,
never enough to price the past. If an agent is asked to price past consumption today, say so
rather than emitting code that will 404.

`GetCustomerCharges`/`GetThirdPartyCharges` only return prices valid now or in the future, so
they cannot price the past. `GetChargeLinksWithCharges` returns the dated price series. The
join key is the `ChargeIdentifier` (Code + Owner + Type) between a `ChargeLink` and a
`ChargeInformation`; `PriceID` on a `Charge`/`TariffCharge` is the same stable identifier as
the `Code`, which is what lets a present-tense charge be matched to its historic series.

```go
from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth) // to is exclusive

// 1. The consumption
series, err := client.GetTimeSeries([]string{id}, from, to, eloverblik.Hour)
if err != nil {
    return err
}

// 2. The dated prices. On BOTH the customer and the third-party client this currently
//    returns a 404 error, because Energinet has not deployed the endpoint. There is no
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
- A subscription (`PricingCategory` / `ChargeIdentifier.Type` naming a subscription) is not
  priced per kWh; `Factor` is the quantity and the price applies per period, not per point.
  Only multiply a tariff by `point.Measurement`.
- `ChargeInformation.Resolution` says how wide each `ChargeSeriesPoint` is (e.g. `PT1H` for an
  hourly tariff), which should match the aggregation the consumption was fetched with.
- Both the consumption points and the price points use half-open `[From, To)` intervals, so an
  interval boundary belongs to the later one. Compare with `!From.After(t) && To.After(t)`.

### Pattern 9: The Half-Open Range, and the Off-by-One It Prevents
```go
// WRONG (the intuitive reading): "give me 1 July through 3 July"
from := time.Date(2026, 7, 1, 0, 0, 0, 0, cph)
to   := time.Date(2026, 7, 3, 0, 0, 0, 0, cph)
client.GetTimeSeries(ids, from, to, eloverblik.Day)
// -> returns 1 July and 2 July only. 3 July is silently missing.

// RIGHT: to is EXCLUSIVE, so pass the day AFTER the last one you want
from := time.Date(2026, 7, 1, 0, 0, 0, 0, cph)
to   := time.Date(2026, 7, 4, 0, 0, 0, 0, cph)
client.GetTimeSeries(ids, from, to, eloverblik.Day)
// -> returns exactly 1, 2 and 3 July. Verified against the live API.

// ALSO WRONG: "just yesterday" as a single day
from := startOfYesterday
to   := startOfYesterday
// -> API error 30002 (ErrorToDateCanNotBeEqualToFromDate). Equal dates are rejected, not
//    interpreted as one day.

// The safe route: let the helper do it. GetDatesFromPeriod already returns an exclusive to.
from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.Yesterday)
// from = start of yesterday, to = start of today
```

## Error Handling Patterns

### How errors surface
```yaml
transport failure: returned as-is from the http client
non-2xx status:    ALWAYS an error. A response with no parseable API message is judged by its
                   status: 429 -> ErrorTooManyRequests, 401 -> ErrorUnauthorized,
                   anything else -> ErrorClientConnection(status)
API error message: the code is read from the first characters, e.g. "[20010] Relation not
                   found", and mapped to an exported sentinel error (compare with errors.Is)
unknown code:      fmt.Errorf("unhandled error: '%s'", msg)
per-item failure:  a batch call still returns 200; the failing metering point carries
                   Success=false, ErrorCode and ErrorText in its own StatusResponse
retried:           429 and 503 only (twice by default). Never a 401, another 4xx, or a 500.
```

### Error codes worth special-casing
```yaml
10002 ErrorToManyRequestItems:                    more than 10 metering points in one request
10004 ErrorMaximumNumberOfMeteringPointsExceeded: batch size exceeded (also arrives as 429)
10007 ErrorNoCprConsent:                          GetMeteringPoints(true) without CPR consent
20010 ErrorRelationNotFound:                      DeleteRelation on a relation that is not there
30002 ErrorToDateCanNotBeEqualToFromDate:         from == to; the range is half-open, see above
30014 ErrorNumberOfDaysExcceded:                  more than 730 days requested
50001 ErrorTokenNotValid / 20012 ErrorUnauthorized: expired or wrong token
      ErrorTooManyRequests:                       a 429 the retries could not absorb
```

### Pattern 1: Check Both Error Returns and Success Fields
```go
details, err := client.GetMeteringPointDetails(ids)
if err != nil {
    // Transport or whole-request API error - no results available
    return err
}

// Check each individual result
for _, detail := range details {
    if !detail.Success {
        // API-level error for this specific ID
        log.Printf("Failed %s: code=%d msg=%s", detail.ID, detail.ErrorCode, detail.ErrorText)
        continue
    }
    processDetail(detail.Result)
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
func validateMeteringPointID(id string) error {
    if len(id) != 18 {
        return fmt.Errorf("ID must be 18 digits, got %d", len(id))
    }
    if _, err := strconv.Atoi(id); err != nil {
        return fmt.Errorf("ID must be numeric: %w", err)
    }
    return nil
}
// The CLI applies exactly this check to its positional arguments, plus a 1-10 count limit.
```

## Data Type Reference

### FlexibleTime
```go
type FlexibleTime struct {
    time.Time
}

// Handles: empty string "", null, or an RFC3339 timestamp
// JSON unmarshaling:
//   "" or null → zero time.Time
//   "2024-01-01T00:00:00Z" → parsed time.Time
// JSON marshaling:
//   zero time.Time → null
//   valid time.Time → RFC3339 string

// Usage:
if !ft.IsZero() {
    formatted := ft.Format(time.RFC3339)
}
```

### Aggregation (what you request)
```go
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
type Resolution string

const (
    PT15M Resolution = "PT15M" // Quarter
    PT1H  Resolution = "PT1H"  // Hour  (one Period per day, 24 points)
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
const (
    CustomerApi   // the zero value
    ThirdPartyApi
)
// The type itself is unexported; the constants are exported and are what
// TokenClaims.APIType() returns.
```

### Period
```go
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
//   - err (error): non-nil if the period string is not one of the constants
// BEHAVIOR:
//   - Weeks start on Sunday (Go's time.Weekday zero value)
//   - Period names are matched case-insensitively
//   - Everything is computed in the local time zone of time.Now()
// EXAMPLE:
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil { /* handle error */ }
ts, err := client.GetTimeSeries(ids, from, to, eloverblik.Day)
```

## Constraints and Limits

```yaml
Metering Point IDs:
  format: Numeric string
  length: Exactly 18 digits
  example: "571313155411053087"
  batch_size: 1-10 IDs per request (the API's recommended and enforced maximum)

Date Ranges:
  semantics: half-open [from, to) at date granularity; from == to is rejected (error 30002)
  timezone: both bounds are converted to Europe/Copenhagen before being formatted YYYY-MM-DD
  maximum span: 730 days (error 30014). eloverblik.MaximumDayRequestLeap = 730 and
                eloverblik.MaximumRequestDuration = 730 * 24h are exported for this.

API Rate Limits:
  token endpoint: 2 calls / minute / IP
  all endpoints:  120 calls / minute / IP
  overall:        1200 calls / minute
  on breach:      429 (and 503 when DataHub is overloaded), both retried twice by default,
                  honouring Retry-After

Token Validity:
  refresh_token:     long lived (typically a year); read the exact expiry with ParseToken
  data_access_token: about 24 hours. Fetched once on first use and cached for the life of the
                     client. NOT auto-refreshed - build a new client, or check
                     DataAccessTokenClaims().IsExpired() yourself.
```

## CLI Output Formats

```yaml
Default: JSON (via encoding/json) on stdout
alive: a human-readable line, not JSON
Export Commands:
  --format=csv: Semicolon-delimited, UTF-8 BOM, Danish headers (streamed straight through)
  --format=json: Converted from CSV to an indented JSON array
--print-response-headers: header blocks on stderr, so stdout stays parseable
```

## Testing Patterns

```yaml
Coverage:
  v1 package:  86.7% of statements
  cmd package: 50.4% of statements

Test Files:
  - *_test.go in v1/ and cmd/ use httpmock to intercept the resty transport
  - Both success and error paths are covered, including non-2xx statuses with empty bodies,
    the retry policy, the header printer, token claim decoding and the half-open periods

Run Tests:
  command: go test ./...
  with_coverage: go test -coverprofile=coverage.out ./...
  with_race: go test -race -coverprofile=coverage.out -covermode=atomic ./...
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
        for _, p := range series.Flatten() {
            total += p.Measurement
            fmt.Printf("%s: %.3f %s (%s)\n",
                p.From.Format(time.DateOnly), p.Measurement, p.Unit, p.Quality)
        }
    }
    fmt.Printf("\nTotal consumption: %.2f kWh\n", total)

    // 9. Get the dated prices for the same period, to price that consumption.
    //    NOTE: this step cannot succeed today. Energinet has not deployed
    //    getchargelinkswithcharges and both APIs answer 404 (verified 2026-07-13), so the
    //    error is logged and the example carries on instead of dying. The only alternative
    //    is client.GetCustomerCharges(...), which gives present and future prices - not the
    //    historic ones this step needs.
    links, err := client.GetChargeLinksWithCharges([]string{firstID}, from, to)
    if err != nil {
        log.Printf("charge links unavailable (expected until Energinet deploys the endpoint): %v", err)
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

    // Fetch hourly time series for one or more metering point IDs (max 10)
    ts, err := client.GetTimeSeries(
        []string{"571313155411053087"},
        from, to,
        eloverblik.Hour, // Aggregation: Actual, Quarter, Hour, Day, Month, Year
    )
    if err != nil {
        log.Fatal(err)
    }

    // Flatten the nested API response into half-open [From, To) intervals with a measurement.
    // Hour comes back as one Period per day holding 24 points; Flatten resolves each point to
    // its own hour, stepping by calendar unit so DST days of 23 or 25 hours stay correct.
    for _, series := range ts {
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
- [ ] Remember the client does NOT auto-refresh the ~24h data access token; rebuild the client
      for a long running process
- [ ] Treat `to` as EXCLUSIVE; never pass `from == to` (error 30002)
- [ ] Stay within 730 days per request (error 30014)
- [ ] Batch metering point IDs 10 at a time
- [ ] Leave the default retry policy on (429/503, Retry-After honoured), or implement backoff
- [ ] Handle both the `error` return and the per-item `Success` field
- [ ] Validate metering point IDs (18 numeric digits)
- [ ] Handle `FlexibleTime` zero values with `IsZero()`, and `*float64` fields with a nil check
- [ ] Use `Flatten()` for time series; it yields From/To/Measurement, not Timestamp/Value
- [ ] Do not assume the resolution string: Day is PT1D, Year is PT1Y on the wire
- [ ] Do NOT build on `GetChargeLinksWithCharges` yet: Energinet has not deployed it and it
      answers 404 on BOTH APIs (verified 2026-07-13). It is the only endpoint that can price
      historic consumption, so that is currently impossible; use `GetCustomerCharges` /
      `GetThirdPartyCharges` for present and future prices and say plainly that the past
      cannot be priced
- [ ] Set `Comma = ';'` on the CSV reader for exports, and close the `io.ReadCloser`
- [ ] Turn on `WithResponseHeaderOutput` / `--print-response-headers` when a call misbehaves
- [ ] Parse CLI stdout as JSON; header debug output goes to stderr

## Quick Reference: Most Common Operations

```go
// Inspect a token without calling the API
claims, _ := eloverblik.ParseToken(token)
api, _ := claims.APIType() // eloverblik.CustomerApi | eloverblik.ThirdPartyApi

// Get consumption for the last 30 days (to is exclusive)
client := eloverblik.NewCustomer(token)
points, _ := client.GetMeteringPoints(false)
id := points[0].MeteringPointID
from := time.Now().AddDate(0, 0, -30)
to := time.Now()
ts, _ := client.GetTimeSeries([]string{id}, from, to, eloverblik.Hour)
data := ts[0].Flatten() // []FlatTimeSeriesPoint{From, To, Measurement, Unit, Quality, ...}

// Export to CSV (Customer API only)
stream, _ := client.ExportTimeSeries([]string{id}, from, to, eloverblik.Hour)
defer stream.Close()
// CLI equivalent with JSON conversion: --format=json

// Get current charges (currently valid and future charges only - never historic prices,
// so this cannot price consumption that already happened). This is the closest available
// data today, because the charge-links endpoint below is not deployed.
charges, _ := client.GetCustomerCharges([]string{id})
for _, tariff := range charges[0].Result.Tariffs {
    _ = tariff.PriceID // the stable key; Prices[].Position is a STRING
}

// Get the dated price series, needed to price historic consumption.
// !! 404 ON BOTH APIS TODAY - Energinet has not deployed getchargelinkswithcharges
//    (verified 2026-07-13). Implemented per spec, ready for the day it goes live.
links, _ := client.GetChargeLinksWithCharges([]string{id}, from, to)
for _, info := range links.ChargeInformations {
    for _, point := range info.ChargeSeriesPoints {
        // point.From, point.To, point.Price, info.TaxIndicator, info.Resolution
    }
}

// Debug a failing call
client := eloverblik.NewCustomer(token, eloverblik.WithResponseHeaderOutput(os.Stderr))

// Third-party: access all customers
client := eloverblik.NewThirdParty(token)
auths, _ := client.GetAuthorizations()
for _, auth := range auths {
    points, _ := client.GetMeteringPointsForScope(
        eloverblik.AuthScopeCustomerKey,
        auth.CustomerKey,
    )
    _ = points
}
```
