# Go Client for Eloverblik.dk

[![Tests](https://github.com/slimcdk/go-eloverblik/workflows/Tests/badge.svg)](https://github.com/slimcdk/go-eloverblik/actions?query=workflow%3ATests)
[![Go Report Card](https://goreportcard.com/badge/github.com/slimcdk/go-eloverblik)](https://goreportcard.com/report/github.com/slimcdk/go-eloverblik)
[![Go Reference](https://pkg.go.dev/badge/github.com/slimcdk/go-eloverblik.svg)](https://pkg.go.dev/github.com/slimcdk/go-eloverblik)
[![License](https://img.shields.io/github/license/slimcdk/go-eloverblik)](LICENSE)

A comprehensive Go client library and CLI tool for the Danish energy data platform [Eloverblik](https://eloverblik.dk/). Access electricity consumption data, metering points, charges, and more through both the Customer API and Third-Party API.

## Features

- **Complete API Coverage**: Supports both Customer and Third-Party APIs
- **Data Export**: Export timeseries, masterdata, and charges in CSV or JSON format
- **Well-Tested**: 73% test coverage with comprehensive unit tests
- **Easy to Use**: Simple CLI interface and intuitive Go library
- **Flexible**: Automatic token refresh and error handling
- **Multi-Platform**: Cross-compiled binaries for Linux, macOS, and Windows

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
  - [CLI Usage](#cli-usage)
  - [Library Usage](#library-usage)
- [API Coverage](#api-coverage)
- [CLI Reference](#cli-reference)
- [Library Reference](#library-reference)
- [Examples](#examples)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Using Go Install

```bash
go install github.com/slimcdk/go-eloverblik/cmd/elob@latest
```

### Download Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/slimcdk/go-eloverblik/releases).

### Build from Source

```bash
git clone https://github.com/slimcdk/go-eloverblik.git
cd go-eloverblik
go build -o elob ./cmd/elob
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

# Get your metering points
elob --token=$ELO_TOKEN customer installations

# Get time series data
elob --token=$ELO_TOKEN customer timeseries 571313155411053087 \
  --from=2024-01-01 --to=2024-01-31

# Export data as JSON
elob --token=$ELO_TOKEN customer export-charges 571313155411053087 \
  --format=json

# Get charges information
elob --token=$ELO_TOKEN customer charges 571313155411053087
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

    // Get time series data
    from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

    timeseries, err := client.GetTimeSeries(
        []string{"571313155411053087"},
        from,
        to,
        eloverblik.Hour,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Process the data
    for _, ts := range timeseries {
        flattened := ts.Flatten()
        for _, point := range flattened {
            fmt.Printf("%s: %.3f kWh\n", point.Timestamp, point.Value)
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
| `/api/isalive` | `thirdparty alive` | `IsAlive()` | Health check |

## CLI Reference

### Global Flags

```
--token string   Eloverblik API refresh token (required, can use env: ELO_TOKEN)
```

### Customer Commands

```bash
# Installation Management
elob customer installations                    # List all metering points
elob customer details <metering-id>...         # Get detailed information

# Relations
elob customer add-relation <metering-id>...    # Add relation by ID
elob customer add-relation-by-code <id> <code> # Add relation by web code
elob customer delete-relation <metering-id>    # Remove relation

# Data Retrieval
elob customer timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --aggregation=Hour \                         # Actual, Quarter, Hour, Day, Month, Year
  --flatten                                    # Simplify output

elob customer charges <metering-id>...         # Get charges and tariffs

# Data Export (CSV or JSON)
elob customer export-timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --format=json                                # csv (default) or json

elob customer export-masterdata <metering-id>... \
  --format=json

elob customer export-charges <metering-id>... \
  --format=json

# Health Check
elob customer alive                            # Check API status
```

### Third-Party Commands

```bash
# Authorization Management
elob thirdparty authorizations                 # List all authorizations

# Metering Points
elob thirdparty metering-points <scope> <identifier>
  # Scope: authorizationid, customercvr, customerkey

elob thirdparty metering-point-ids <scope> <identifier>
  # Get IDs only (faster)

# Data Retrieval
elob thirdparty details <metering-id>...
elob thirdparty timeseries <metering-id>... \
  --from=YYYY-MM-DD \
  --to=YYYY-MM-DD \
  --aggregation=Hour

elob thirdparty charges <metering-id>...

# Health Check
elob thirdparty alive
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

### Aggregation Levels

```go
eloverblik.Actual   // Raw meter readings (15-min intervals)
eloverblik.Quarter  // 15-minute aggregation
eloverblik.Hour     // Hourly aggregation
eloverblik.Day      // Daily aggregation
eloverblik.Month    // Monthly aggregation
eloverblik.Year     // Yearly aggregation
```

### Authorization Scopes (Third-Party API)

```go
eloverblik.AuthorizationIDScope  // Scope by authorization ID
eloverblik.CustomerCVRScope      // Scope by customer CVR number
eloverblik.CustomerKeyScope      // Scope by customer key
```

## Examples

### Export Data to CSV File

```bash
elob --token=$ELO_TOKEN customer export-timeseries 571313155411053087 \
  --from=2024-01-01 --to=2024-12-31 > consumption_2024.csv
```

### Export Data to JSON File

```bash
elob --token=$ELO_TOKEN customer export-charges 571313155411053087 \
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
    // Flatten nested structure to simple timestamp/value pairs
    points := ts.Flatten()

    for _, point := range points {
        fmt.Printf("%s: %.3f %s (Quality: %s)\n",
            point.Timestamp.Format(time.RFC3339),
            point.Value,
            point.Unit,
            point.Quality,
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
        eloverblik.CustomerKeyScope,
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

### Error Handling

```go
details, err := client.GetMeteringPointDetails(meteringPoints)
if err != nil {
    log.Fatal(err)
}

for _, detail := range details {
    if !detail.Success {
        // Handle API-level errors
        switch detail.ErrorCode {
        case 10001:
            log.Printf("Authentication failed")
        case 10002:
            log.Printf("Authorization failed")
        default:
            log.Printf("Error %d: %s", detail.ErrorCode, detail.ErrorText)
        }
        continue
    }

    // Process successful response
    processDetail(detail.Result)
}
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./v1/...
go test -v ./cmd/...
```

### Running Linter

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run --timeout=5m
```

### Building

```bash
# Build for current platform
go build -o elob ./cmd/elob

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o elob-linux-amd64 ./cmd/elob
GOOS=darwin GOARCH=arm64 go build -o elob-darwin-arm64 ./cmd/elob
GOOS=windows GOARCH=amd64 go build -o elob-windows-amd64.exe ./cmd/elob
```

## Project Structure

```
.
├── cmd/                    # CLI command implementations
│   ├── elob/               # Application entry point
│   │   └── main.go
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
└── go.sum                  # Dependency checksums
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
