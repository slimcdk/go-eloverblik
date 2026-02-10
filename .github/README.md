# GitHub Actions CI/CD

This repository uses GitHub Actions for continuous integration and delivery.

## Workflows

### Test Workflow (`.github/workflows/test.yml`)

Automatically runs on:
- Push to `master`, `main`, or `develop` branches
- Pull requests targeting `master`, `main`, or `develop` branches

#### Jobs

**1. Test Job**
- Runs unit tests with race detection
- Tests against multiple Go versions (1.21, 1.22, 1.23)
- Generates code coverage reports
- Uploads coverage to Codecov (optional)
- **Coverage:** ~65.5% overall
  - `cmd` package: 37.0%
  - `v1` package: 67.1%

**2. Lint Job**
- Runs golangci-lint for code quality checks
- Configured via `.golangci.yml`
- Checks for:
  - Unchecked errors
  - Code simplification opportunities
  - Security issues (gosec)
  - Formatting (gofmt, goimports)
  - Common misspellings
  - Unused code

**3. Build Job**
- Cross-compiles binaries for multiple platforms:
  - Linux (amd64, arm64)
  - macOS/Darwin (amd64, arm64)
  - Windows (amd64)
- Uploads build artifacts (retained for 7 days)

## Local Testing

Run the same checks locally before pushing:

```bash
# Run tests with race detection and coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage report
go tool cover -func=coverage.out

# Run linter (requires golangci-lint)
golangci-lint run --timeout=5m

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -v -o dist/elob-linux-amd64 .
```

## Coverage Reports

To view detailed coverage in your browser:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Adding Status Badges

Add these badges to your main README.md:

```markdown
[![Tests](https://github.com/slimcdk/go-eloverblik/workflows/Tests/badge.svg)](https://github.com/slimcdk/go-eloverblik/actions?query=workflow%3ATests)
[![Go Report Card](https://goreportcard.com/badge/github.com/slimcdk/go-eloverblik)](https://goreportcard.com/report/github.com/slimcdk/go-eloverblik)
[![codecov](https://codecov.io/gh/slimcdk/go-eloverblik/branch/master/graph/badge.svg)](https://codecov.io/gh/slimcdk/go-eloverblik)
```

## Codecov Integration (Optional)

To enable Codecov coverage reporting:

1. Sign up at [codecov.io](https://codecov.io)
2. Add your repository
3. No token needed for public repositories
4. For private repos, add `CODECOV_TOKEN` to repository secrets

## Linter Configuration

The project uses `.golangci.yml` for linter configuration. Key settings:

- **Enabled linters:** errcheck, gosimple, govet, ineffassign, staticcheck, unused, gofmt, goimports, misspell, revive, gosec
- **Timeout:** 5 minutes
- **Test files:** Some checks are relaxed for `*_test.go` files

## Troubleshooting

### Tests fail locally but pass in CI
- Ensure you're using the correct Go version
- Run `go mod download && go mod verify`
- Check for race conditions with `-race` flag

### Linter fails
- Run `golangci-lint run` locally
- Check `.golangci.yml` for disabled rules
- Some errors can be auto-fixed with `golangci-lint run --fix`

### Build fails for specific platform
- Check GOOS/GOARCH compatibility
- Ensure no platform-specific code without build tags
- Test cross-compilation locally:
  ```bash
  GOOS=windows GOARCH=amd64 go build .
  ```
