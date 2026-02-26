# Eloverblik Go Client - AI-Optimized Documentation

This documentation is structured for AI agents to easily understand, parse, and implement the Eloverblik Go client library and CLI.

## Package Information

```yaml
package: github.com/slimcdk/go-eloverblik
import: github.com/slimcdk/go-eloverblik/v1
install: go get github.com/slimcdk/go-eloverblik/v1
language: Go
purpose: Interface with Danish Eloverblik electricity data API
apis:
  - Customer API (consumer/household electricity data)
  - Third-Party API (business/multi-customer data access)
```

## Core Concepts

### 1. Authentication Flow
```
User obtains refresh token → NewCustomer/NewThirdParty creates client → Client automatically:
  1. Uses refresh token to get access token
  2. Caches access token
  3. Auto-refreshes when expired
  4. Adds Authorization header to all requests
```

### 2. Client Types
```yaml
Customer:
  purpose: Individual consumers accessing their own data
  factory: eloverblik.NewCustomer(refreshToken string)
  capabilities: [installations, timeseries, charges, relations, exports]

ThirdParty:
  purpose: Businesses accessing multiple customers' data via authorization
  factory: eloverblik.NewThirdParty(refreshToken string)
  capabilities: [authorizations, metering-points, timeseries, charges]
```

## Function Signatures with Complete Parameter Specifications

### Client Creation

```go
// FUNCTION: NewCustomer
// PURPOSE: Create Customer API client
// INPUT: refreshToken (string) - Obtained from eloverblik.dk portal
// OUTPUT: Customer interface
// EXAMPLE:
client := eloverblik.NewCustomer("your-refresh-token-here")
```

```go
// FUNCTION: NewThirdParty
// PURPOSE: Create Third-Party API client
// INPUT: refreshToken (string) - Obtained from eloverblik.dk portal
// OUTPUT: ThirdParty interface
// EXAMPLE:
client := eloverblik.NewThirdParty("your-refresh-token-here")
```

### Customer API Methods

```go
// FUNCTION: GetMeteringPoints
// PURPOSE: List all metering points (installations) for authenticated user
// INPUTS:
//   - includeAll (bool): true = all points, false = only active
// OUTPUTS:
//   - []MeteringPoints: Array of metering point metadata
//   - error: nil on success, error object on failure
// RETURNS:
//   - MeteringPointID (string, 18 digits)
//   - TypeOfMP (string)
//   - StreetName, BuildingNumber, Postcode, CityName (strings)
//   - ConsumerStartDate (FlexibleTime - may be empty)
//   - HasRelation (bool)
// EXAMPLE:
points, err := client.GetMeteringPoints(true)
if err != nil { /* handle error */ }
for _, point := range points {
    meteringID := point.MeteringPointID // Use this ID for other API calls
}
```

```go
// FUNCTION: GetMeteringPointDetails
// PURPOSE: Get detailed information about specific metering points
// INPUTS:
//   - meteringPointIDs ([]string): Array of 18-digit metering point IDs
//     CONSTRAINTS: 1-10 IDs, each exactly 18 digits, numeric only
// OUTPUTS:
//   - []MeteringPointDetailsResponse: Array with one response per input ID
//   - error: HTTP/network errors
// RETURNS: Each MeteringPointDetailsResponse contains:
//   - Success (bool): true if this specific point succeeded
//   - ErrorCode (int): 10000 = success, other = error
//   - ErrorText (string): Human-readable error
//   - ID (string): The metering point ID
//   - Result (MeteringPointDetails): Detailed data if Success=true
// RESULT FIELDS:
//   - MeteringPointID, GridOperatorName, MeterNumber
//   - SettlementMethod, ConsumerStartDate, BalanceSupplierName
//   - ContactAddresses (array of address objects)
//   - And 40+ more fields
// ERROR HANDLING:
//   Always check Success field per result, not just error return
// EXAMPLE:
ids := []string{"571313155411053087", "571313155411782079"}
details, err := client.GetMeteringPointDetails(ids)
if err != nil { /* handle network error */ }
for _, detail := range details {
    if !detail.Success {
        log.Printf("Failed for %s: %s", detail.ID, detail.ErrorText)
        continue
    }
    // Use detail.Result fields
    gridOp := detail.Result.GridOperatorName
}
```

```go
// FUNCTION: GetTimeSeries
// PURPOSE: Retrieve electricity consumption time series data
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 metering point IDs
//   - from (time.Time): Start date/time (inclusive)
//   - to (time.Time): End date/time (exclusive)
//   - aggregation (Aggregation): Data granularity
//     OPTIONS: Actual, Quarter, Hour, Day, Month, Year
//     ENUM VALUES:
//       - eloverblik.Actual: Raw 15-min meter readings
//       - eloverblik.Quarter: 15-minute aggregation
//       - eloverblik.Hour: Hourly aggregation
//       - eloverblik.Day: Daily aggregation
//       - eloverblik.Month: Monthly aggregation
//       - eloverblik.Year: Yearly aggregation
// DATE CONSTRAINTS:
//   - Maximum range: varies by aggregation
//   - Timezone: Use time.UTC or Europe/Copenhagen
// OUTPUTS:
//   - []TimeSeries: Nested structure with consumption data
//   - error: HTTP/network errors
// DATA STRUCTURE:
//   TimeSeries.MyEnergyDataMarketDocument.TimeSeries[].Period[].Point[]
//   Each Point has: Position, Quantity, Quality
// HELPER METHOD: ts.Flatten() converts nested structure to flat array
// EXAMPLE:
from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
ts, err := client.GetTimeSeries(
    []string{"571313155411053087"},
    from, to,
    eloverblik.Hour,
)
if err != nil { /* handle error */ }
for _, series := range ts {
    flat := series.Flatten() // Simplifies nested structure
    for _, point := range flat {
        fmt.Printf("%s: %.3f kWh\n", point.Timestamp, point.Value)
    }
}
```

```go
// FUNCTION: GetCustomerCharges
// PURPOSE: Get pricing information (subscriptions, fees, tariffs)
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 metering point IDs
// OUTPUTS:
//   - []CustomerChargeResponse: Pricing data per metering point
//   - error: HTTP/network errors
// RETURNS: Each CustomerChargeResponse contains:
//   - MeteringPointID (string)
//   - Subscriptions ([]Subscription): Monthly fees
//     Fields: Name, Price, Quantity, ValidFromDate, ValidToDate
//   - Fees ([]Fee): One-time fees
//   - Tariffs ([]Tariff): Usage-based pricing
//     Has Prices array with Position (1-24 for hourly) and Price
// EXAMPLE:
charges, err := client.GetCustomerCharges([]string{"571313155411053087"})
for _, charge := range charges {
    for _, tariff := range charge.Result.Tariffs {
        if len(tariff.Prices) == 24 {
            // Hourly tariff - different price per hour
            for _, price := range tariff.Prices {
                hour := price.Position
                rate := price.Price
            }
        }
    }
}
```

```go
// FUNCTION: ExportTimeSeries
// PURPOSE: Export time series data as CSV stream
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 IDs
//   - from, to (time.Time): Date range
//   - aggregation (Aggregation): Data granularity
// OUTPUTS:
//   - io.ReadCloser: CSV data stream (Danish format, semicolon-delimited)
//   - error: HTTP/network errors
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
// FUNCTION: ExportMasterdata
// PURPOSE: Export metering point master data as CSV
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 IDs
// OUTPUTS:
//   - io.ReadCloser: CSV stream with 80+ columns of metadata
//   - error: HTTP/network errors
// CSV COLUMNS: Include grid operator, addresses, meter info, contracts, etc.
// EXAMPLE:
stream, err := client.ExportMasterdata([]string{"571313155411053087"})
defer stream.Close()
io.Copy(os.Stdout, stream) // Print to stdout
```

```go
// FUNCTION: ExportCharges
// PURPOSE: Export charges as CSV
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 IDs
// OUTPUTS:
//   - io.ReadCloser: CSV with subscriptions, fees, tariffs
//   - error: HTTP/network errors
// CSV STRUCTURE: One row per charge item, including hourly tariff positions
```

```go
// FUNCTION: AddRelationByID
// PURPOSE: Link metering points to authenticated user
// INPUTS:
//   - meteringPointIDs ([]string): IDs to link
// OUTPUTS:
//   - []StringResponse: One response per ID
//   - error: HTTP/network errors
// RESPONSE FIELDS:
//   - Success (bool)
//   - Result (string): Success message or error
// EXAMPLE:
responses, err := client.AddRelationByID([]string{"571313155411053087"})
for _, resp := range responses {
    if resp.Success {
        fmt.Println("Linked:", resp.Result)
    }
}
```

```go
// FUNCTION: AddRelationByWebAccessCode
// PURPOSE: Link metering point using web access code
// INPUTS:
//   - meteringPointID (string): Single 18-digit ID
//   - webAccessCode (string): 8-digit code from letter/email
// OUTPUTS:
//   - string: Success message
//   - error: HTTP/network errors
// EXAMPLE:
result, err := client.AddRelationByWebAccessCode("571313155411053087", "12345678")
```

```go
// FUNCTION: DeleteRelation
// PURPOSE: Unlink metering point from authenticated user
// INPUTS:
//   - meteringPointID (string): Single 18-digit ID
// OUTPUTS:
//   - bool: true if successfully deleted
//   - error: HTTP/network errors
```

```go
// FUNCTION: IsAlive
// PURPOSE: Health check for API availability
// INPUTS: none
// OUTPUTS:
//   - bool: true if API is operational
//   - error: Network errors only
// USE CASE: Pre-flight check before making multiple API calls
```

### Third-Party API Methods

```go
// FUNCTION: GetAuthorizations
// PURPOSE: List all customer authorizations granted to third party
// INPUTS: none
// OUTPUTS:
//   - []Authorization: Array of authorization grants
//   - error: HTTP/network errors
// AUTHORIZATION FIELDS:
//   - ID (string): Authorization ID
//   - CustomerName (string): Customer who granted access
//   - CustomerKey (string): Use for GetMeteringPointsForScope
//   - CustomerCVR (string): Company registration number
//   - ValidFrom, ValidTo (FlexibleTime): Authorization period
//   - IncludeFutureMeteringPoints (bool)
// EXAMPLE:
auths, err := client.GetAuthorizations()
for _, auth := range auths {
    customerKey := auth.CustomerKey // Use this for next call
    meteringPoints, _ := client.GetMeteringPointsForScope(
        eloverblik.CustomerKeyScope,
        customerKey,
    )
}
```

```go
// FUNCTION: GetMeteringPointsForScope
// PURPOSE: Get metering points for specific authorization scope
// INPUTS:
//   - scope (AuthorizationScope): Type of identifier
//     OPTIONS:
//       - eloverblik.AuthorizationIDScope: Use authorization.ID
//       - eloverblik.CustomerCVRScope: Use CVR number
//       - eloverblik.CustomerKeyScope: Use customer key
//   - identifier (string): The ID/CVR/key value
// OUTPUTS:
//   - []ThirdPartyMeteringPoint: Metering points with access dates
//   - error: HTTP/network errors
// METERING POINT FIELDS:
//   - MeteringPointID (string)
//   - TypeOfMP, StreetName, BuildingNumber, Postcode, CityName
//   - AccessFrom, AccessTo (FlexibleTime): Access period
// WORKFLOW:
//   1. GetAuthorizations() to get customer keys
//   2. For each auth, GetMeteringPointsForScope() to get their points
//   3. Use metering point IDs with GetTimeSeries(), etc.
// EXAMPLE:
points, err := client.GetMeteringPointsForScope(
    eloverblik.CustomerKeyScope,
    "customer-key-from-authorization",
)
```

```go
// FUNCTION: GetMeteringPointIDsForScope
// PURPOSE: Get only IDs (faster than GetMeteringPointsForScope)
// INPUTS: Same as GetMeteringPointsForScope
// OUTPUTS:
//   - []string: Array of metering point IDs
//   - error: HTTP/network errors
// USE CASE: When you only need IDs, not full metadata
```

```go
// FUNCTION: GetThirdPartyCharges
// PURPOSE: Get charges for third-party accessed metering points
// INPUTS:
//   - meteringPointIDs ([]string): 1-10 IDs
// OUTPUTS:
//   - []ThirdPartyChargeResponse: Similar to CustomerChargeResponse
//   - error: HTTP/network errors
// DIFFERENCE FROM CUSTOMER: May have different charge structures
```

## CLI Command Patterns

### Command Structure
```
go-eloverblik --token=<refresh-token> <api-type> <command> [args] [flags]

Components:
  --token: Global flag, required for all commands
  <api-type>: "customer" or "thirdparty"
  <command>: Action to perform
  [args]: Positional arguments (usually metering point IDs)
  [flags]: Optional command-specific flags
```

### Date Specification (--from / --to / --period)

The `timeseries` and `export-timeseries` commands support two mutually exclusive ways to specify date ranges:

```yaml
Option A - Explicit dates with --from and --to:
  --from: Required start date
  --to: End date (defaults to today)
  Formats:
    - YYYY-MM-DD: Absolute date (e.g., 2024-01-01)
    - now: Current date/time
    - now-Nd: N days ago (e.g., now-30d)
    - now-Nw: N weeks ago (e.g., now-4w)
    - now-Nm: N months ago (e.g., now-2m)
    - now-Ny: N years ago (e.g., now-1y)

Option B - Predefined period with --period:
  --period: Named time range (cannot be combined with --from or --to)
  Values: yesterday, this_week, last_week, this_month, last_month, this_year, last_year

Examples:
  --from=2024-01-01 --to=2024-01-31    # Explicit range
  --from=now-30d                         # Last 30 days (--to defaults to today)
  --from=now-1y --to=now-6m             # Relative range
  --period=last_month                    # Predefined period
```

### CLI to Library Mapping

```yaml
CLI: go-eloverblik customer installations
Library: client.GetMeteringPoints(true)
Returns: JSON array of metering points

CLI: go-eloverblik customer details 571313155411053087
Library: client.GetMeteringPointDetails([]string{"571313155411053087"})
Returns: JSON array with detailed information

CLI: go-eloverblik customer timeseries 571313155411053087 --from=2024-01-01 --to=2024-01-31
Library: |
  from := time.Parse(time.DateOnly, "2024-01-01")
  to := time.Parse(time.DateOnly, "2024-01-31")
  client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
Returns: JSON with nested time series structure

CLI: go-eloverblik customer timeseries 571313155411053087 --from=now-30d
Library: |
  from := time.Now().AddDate(0, 0, -30)
  to := time.Now()
  client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
Returns: JSON with nested time series structure

CLI: go-eloverblik customer timeseries 571313155411053087 --period=last_month
Library: |
  from, to, _ := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
  client.GetTimeSeries([]string{"571313155411053087"}, from, to, eloverblik.Hour)
Returns: JSON with nested time series structure

CLI: go-eloverblik customer export-timeseries 571313155411053087 --from=2024-01-01 --format=json
Library: |
  stream, _ := client.ExportTimeSeries(...)
  // Then convert CSV to JSON
Returns: JSON array of consumption records

CLI: go-eloverblik thirdparty authorizations
Library: client.GetAuthorizations()
Returns: JSON array of authorization grants
```

### CSV to JSON Conversion

When using export commands with `--format=json`:
```yaml
Input: CSV stream with semicolon delimiter, UTF-8 BOM, Danish headers
Process:
  1. Parse CSV with ';' delimiter
  2. Read header row as object keys
  3. Read data rows as values
  4. Output JSON array of objects
Output: JSON array where each object represents one CSV row
```

## Common Patterns

### Pattern 1: Get All Consumption Data
```go
// 1. Create client
client := eloverblik.NewCustomer(token)

// 2. Get metering points
points, _ := client.GetMeteringPoints(true)

// 3. For each point, get time series
for _, point := range points {
    from := time.Now().AddDate(0, -1, 0) // 1 month ago
    to := time.Now()

    ts, err := client.GetTimeSeries(
        []string{point.MeteringPointID},
        from, to,
        eloverblik.Day,
    )

    if err != nil {
        continue
    }

    // Process data
    for _, series := range ts {
        flat := series.Flatten()
        // flat is []FlatTimeSeriesPoint with Timestamp, Value, Unit, Quality
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
        eloverblik.CustomerKeyScope,
        auth.CustomerKey,
    )

    // 5. Get data for their points
    var ids []string
    for _, p := range points {
        ids = append(ids, p.MeteringPointID)
    }

    ts, _ := client.GetTimeSeries(ids, from, to, eloverblik.Hour)
    // Process data...
}
```

### Pattern 3: Export and Parse CSV Data
```go
// 1. Export as CSV stream
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

    // Map to struct or process directly
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
// Instead of calculating dates manually, use Period constants:
from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastMonth)
if err != nil {
    log.Fatal(err)
}

ts, err := client.GetTimeSeries(
    []string{"571313155411053087"},
    from, to,
    eloverblik.Day,
)
// Process data...
```

## Error Handling Patterns

### Pattern 1: Check Both Error Returns and Success Fields
```go
details, err := client.GetMeteringPointDetails(ids)
if err != nil {
    // Network/HTTP error - no results available
    return err
}

// Check each individual result
for _, detail := range details {
    if !detail.Success {
        // API-level error for this specific ID
        log.Printf("Failed %s: code=%d msg=%s",
            detail.ID,
            detail.ErrorCode,
            detail.ErrorText,
        )
        continue
    }

    // Process successful result
    processDetail(detail.Result)
}
```

### Pattern 2: Handle Empty Dates
```go
// FlexibleTime fields may be zero values
if !meteringPoint.ConsumerStartDate.IsZero() {
    startDate := meteringPoint.ConsumerStartDate.Time
    // Use as regular time.Time
}
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
```

## Data Type Reference

### FlexibleTime
```go
type FlexibleTime struct {
    time.Time
}

// Handles: empty string "", null, or RFC3339 timestamp
// JSON unmarshaling:
//   "" or null → zero time.Time
//   "2024-01-01T00:00:00Z" → parsed time.Time
// JSON marshaling:
//   zero time.Time → null
//   valid time.Time → RFC3339 string

// Usage:
if !ft.IsZero() {
    // Use ft.Time for standard time operations
    formatted := ft.Format(time.RFC3339)
}
```

### Aggregation
```go
type Aggregation string

const (
    Actual  Aggregation = "Actual"   // 15-min raw readings
    Quarter Aggregation = "Quarter"  // 15-min aggregation
    Hour    Aggregation = "Hour"     // 60-min aggregation
    Day     Aggregation = "Day"      // Daily total
    Month   Aggregation = "Month"    // Monthly total
    Year    Aggregation = "Year"     // Yearly total
)
```

### AuthorizationScope
```go
type AuthorizationScope string

const (
    AuthorizationIDScope AuthorizationScope = "authorizationid"
    CustomerCVRScope     AuthorizationScope = "customercvr"
    CustomerKeyScope     AuthorizationScope = "customerkey"
)
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
// PURPOSE: Convert a Period constant to concrete from/to time.Time values
// INPUTS:
//   - period (Period): One of the predefined period constants
// OUTPUTS:
//   - from (time.Time): Start of the period
//   - to (time.Time): End of the period
//   - err (error): Non-nil if period string is invalid
// BEHAVIOR:
//   - Yesterday: midnight yesterday to end of yesterday
//   - ThisWeek/LastWeek: week starts on Sunday
//   - ThisMonth/ThisYear: from start of period to now
//   - LastMonth/LastYear: full previous period
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
  batch_size: 1-10 IDs per request

Date Ranges:
  format: time.Time in UTC or Europe/Copenhagen
  typical_range: Up to 1 year per request

Time Series Aggregation:
  Actual: Max 31 days
  Hour: Max 1 year
  Day: Max 5 years
  Month: Max 10 years

API Rate Limits:
  status: Not publicly documented
  recommendation: Implement exponential backoff on 429 responses

Token Validity:
  refresh_token: Long-lived (months/years)
  access_token: Short-lived (hours), auto-refreshed by client
```

## CLI Output Formats

```yaml
Default: JSON (via encoding/json)
Export Commands:
  --format=csv: Semicolon-delimited, UTF-8 BOM, Danish headers
  --format=json: Converted from CSV to JSON array
```

## Testing Patterns

```yaml
Unit Test Coverage: 84.1%
  cmd package: 60.2%
  v1 package: 80.6%

Test Files:
  - *_test.go files use httpmock for HTTP mocking
  - Tests validate both success and error cases
  - CSV export tests verify format and encoding
  - Error handling tests for API failures
  - 50 comprehensive test cases

Run Tests:
  command: go test ./...
  with_coverage: go test -coverprofile=coverage.out ./...
  with_race: go test -race -coverprofile=coverage.out -covermode=atomic ./...
```

## Complete Working Example

```go
package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "time"

    eloverblik "github.com/slimcdk/go-eloverblik/v1"
)

func main() {
    // 1. Get token from environment
    token := os.Getenv("ELO_TOKEN")
    if token == "" {
        log.Fatal("ELO_TOKEN environment variable required")
    }

    // 2. Create client
    client := eloverblik.NewCustomer(token)

    // 3. Check API health
    alive, err := client.IsAlive()
    if err != nil || !alive {
        log.Fatal("API not available")
    }

    // 4. Get metering points
    points, err := client.GetMeteringPoints(true)
    if err != nil {
        log.Fatal(err)
    }

    if len(points) == 0 {
        log.Fatal("No metering points found")
    }

    fmt.Printf("Found %d metering points\n", len(points))

    // 5. Get first point's details
    firstID := points[0].MeteringPointID
    details, err := client.GetMeteringPointDetails([]string{firstID})
    if err != nil {
        log.Fatal(err)
    }

    if !details[0].Success {
        log.Fatalf("Details failed: %s", details[0].ErrorText)
    }

    fmt.Printf("Grid Operator: %s\n", details[0].Result.GridOperatorName)

    // 6. Get last month's consumption
    now := time.Now()
    from := now.AddDate(0, -1, 0)

    ts, err := client.GetTimeSeries(
        []string{firstID},
        from, now,
        eloverblik.Day,
    )
    if err != nil {
        log.Fatal(err)
    }

    // 7. Process and display data
    for _, series := range ts {
        flat := series.Flatten()

        var total float64
        for _, point := range flat {
            total += point.Value
            fmt.Printf("%s: %.3f %s\n",
                point.Timestamp.Format("2006-01-02"),
                point.Value,
                point.Unit,
            )
        }

        fmt.Printf("\nTotal consumption: %.2f kWh\n", total)
    }

    // 8. Export charges as CSV
    stream, err := client.ExportCharges([]string{firstID})
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    // 9. Parse and display charges
    reader := csv.NewReader(stream)
    reader.Comma = ';'

    headers, _ := reader.Read()
    fmt.Printf("\nCharges columns: %v\n", headers)

    for {
        record, err := reader.Read()
        if err != nil {
            break
        }
        fmt.Printf("Charge: %v\n", record)
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

    // Option A: specify dates manually
    from := time.Now().AddDate(0, 0, -7)
    to := time.Now()

    // Option B: use a predefined period helper
    // from, to, err := eloverblik.GetDatesFromPeriod(eloverblik.LastWeek)

    // Fetch hourly time series for one or more metering point IDs
    ts, err := client.GetTimeSeries(
        []string{"571313155411053087"},
        from, to,
        eloverblik.Hour, // Aggregation: Actual, Quarter, Hour, Day, Month, Year
    )
    if err != nil {
        log.Fatal(err)
    }

    // Flatten the nested API response into simple timestamp/value pairs
    for _, series := range ts {
        for _, point := range series.Flatten() {
            fmt.Printf("%s  %.3f %s (quality: %s)\n",
                point.Timestamp.Format(time.RFC3339),
                point.Value,
                point.Unit,
                point.Quality,
            )
        }
    }

    // All API responses serialize to JSON
    out, _ := json.MarshalIndent(ts, "", "  ")
    fmt.Println(string(out))
}
```

## AI Agent Implementation Checklist

When implementing Eloverblik client:

- [ ] Store refresh token securely (environment variable or secure vault)
- [ ] Create appropriate client (Customer vs ThirdParty)
- [ ] Always check `IsAlive()` before bulk operations
- [ ] Handle both `error` return and `Success` field in responses
- [ ] Validate metering point ID format (18 numeric digits)
- [ ] Respect batch size limits (1-10 IDs per request)
- [ ] Handle `FlexibleTime` zero values with `IsZero()` check
- [ ] Use `Flatten()` method for simplified time series data
- [ ] Set proper CSV reader options (`Comma = ';'`) for exports
- [ ] Implement exponential backoff for rate limiting
- [ ] Close `io.ReadCloser` streams with `defer stream.Close()`
- [ ] Use appropriate aggregation level for date range
- [ ] Convert times to UTC or Europe/Copenhagen timezone
- [ ] Parse CLI output as JSON for programmatic use
- [ ] Test with both success and error scenarios

## Quick Reference: Most Common Operations

```go
// Get consumption for last 30 days
client := eloverblik.NewCustomer(token)
points, _ := client.GetMeteringPoints(true)
id := points[0].MeteringPointID
from := time.Now().AddDate(0, 0, -30)
to := time.Now()
ts, _ := client.GetTimeSeries([]string{id}, from, to, eloverblik.Hour)
data := ts[0].Flatten()

// Export to JSON file
stream, _ := client.ExportTimeSeries([]string{id}, from, to, eloverblik.Hour)
// Convert CSV to JSON using cli: --format=json

// Get current charges
charges, _ := client.GetCustomerCharges([]string{id})
for _, tariff := range charges[0].Result.Tariffs {
    // Process hourly rates
}

// Third-party: Access all customers
client := eloverblik.NewThirdParty(token)
auths, _ := client.GetAuthorizations()
for _, auth := range auths {
    points, _ := client.GetMeteringPointsForScope(
        eloverblik.CustomerKeyScope,
        auth.CustomerKey,
    )
    // Process points...
}
```
