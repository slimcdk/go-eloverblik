# AGENTS.md

Instructions for coding agents working on this repository. Agents that use the library or the
CLI from other code never see this file: they get `go doc` (pkg.go.dev),
`go-eloverblik --help` and `llms.md`. Whatever a caller must know therefore belongs in the
godoc of the exported identifier and in the command's help text, not only in README.md.

## Map

- `v1/`: the library, package `eloverblik`, imported as `github.com/slimcdk/go-eloverblik/v1`.
- `cmd/`: the cobra CLI. `main.go` at the module root only calls `cmd.Execute()`.
- `docs/`: Energinet's API documentation. Only the OpenAPI documents and the PDF are copied
  as published: the guides in `docs/eloverblik-guides/` were rendered and converted to
  Markdown, and the JSON response skeletons come from Appendix B of the technical
  description's 2020 edition. Never edit these files; they are Energinet's material, not
  covered by the MIT license. `docs/README.md` is ours: it names the source of each file, how
  it was obtained, and how to refresh the ones that can be refreshed by hand.

## Checks

Run all of these before every commit. They must pass, and golangci-lint must report 0 issues.

```sh
go mod verify
gofmt -l $(git ls-files '*.go')               # must print nothing
go vet ./...
go test -race ./...
env -u GOROOT -u ZONEINFO go test -count=1 -trimpath ./...
GOARCH=386 go test ./...
GOTOOLCHAIN=go1.26.0 go vet ./... && GOTOOLCHAIN=go1.26.0 go test ./...   # the oldest Go
golangci-lint run --timeout=5m
go mod tidy -diff                            # must print nothing
actionlint .github/workflows/*.yml           # when a workflow changes
```

How CI runs them (`.github/workflows/test.yml`; before a release, `release.yml` runs
`go mod verify`, `go mod tidy -diff`, `go test -race ./...`, the 386 run and the ARMv6 run
again as its gate):

- `test.yml` runs for pushes to and pull requests into master, main and develop. On any
  other branch, such as one stacked on another pull request, start it by hand:
  `gh workflow run test.yml --ref <branch>`.
- `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...` on ubuntu, windows and
  macos runners, every step in bash, followed by the `-trimpath` run above on all three.
- The `-trimpath` run with `GOROOT` and `ZONEINFO` unset leaves the tests, like a release
  binary, no Go installation to fall back on for zoneinfo. On Windows, which has no system
  zoneinfo, Europe/Copenhagen then comes only from the `time/tzdata` import in
  `v1/constvars.go`. On Linux, CI also runs the release build in a busybox container without
  network or zoneinfo; see that step.
- `go.mod` has two Go lines. `go 1.26.0` is the oldest Go a caller can build the module
  with: the older of the two releases the Go team supports. Raise it only when that drops
  out of support or the code needs something newer, and never for a test alone. The
  `toolchain` line is the Go that CI, `go-version-file` and the release build with; move it
  to each new release. A job with `GOTOOLCHAIN=local` on Go 1.26.0 vets and tests the
  module, so a feature the go line does not allow fails there; move its version with the
  go line.
- `GOARCH=386` and the ARMv6 run below test the suite as 32-bit code, where `int` is 32 bits.
  Metering point IDs have 18 digits: never parse one into an `int`. Neither run uses `-race`,
  which supports neither architecture.
- The shipped linux/arm build, on the ARMv6 CPU of a Raspberry Pi Zero or Pi 1:

  ```sh
  GOARCH=arm GOARM=6 QEMU_CPU=arm1176 go test -exec qemu-arm ./...
  ```

  Locally this needs the `qemu-user` package (Debian/Ubuntu), which provides `qemu-arm`.
  `-exec` runs each test binary through it, so no binfmt registration is involved.
- golangci-lint runs through `golangci/golangci-lint-action` with `version: v2.14.0` and
  `--timeout=5m`. The config is in the v2 format, which v1 cannot read. Locally:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`.
  It runs govet (every analyzer but shadow and fieldalignment) and reports gofmt and
  goimports differences, which is why `gofmt` is not a CI step of its own and `go vet` is
  one only in the oldest-Go job. The config sets the `live` build tag, so it also
  type-checks and lints `v1/live_test.go`, which only that job vets too. Linting runs no
  test.
  `actionlint` is not run by CI at all.
- The Build job cross-compiles every shipped target with `go build -v -o /dev/null .`:
  linux, darwin and windows on amd64 and arm64, plus linux/arm with `GOARM=6`.
- `.github/workflows/security.yml` runs CodeQL and govulncheck v1.8.0,
  `govulncheck -show verbose ./...` (locally:
  `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 -show verbose ./...`). It sets up Go
  `1.27` instead of reading `go.mod`, and Dependabot does not see the govulncheck pin: move
  both by hand when the toolchain line in `go.mod` moves.

## Conventions

- Test first: a change in behaviour starts with a test that fails without it.
- Tests never call api.eloverblik.dk, except the [live tests](#live-tests), and there is
  no test host: `NewCustomer` and `NewThirdParty` always target `https://api.eloverblik.dk`
  (`Mode` and `TestMode` have no effect). In `v1/`, activate httpmock on the client's own
  HTTP client before its first request, `httpmock.ActivateNonDefault(c.resty.GetClient())`,
  so an unregistered request fails instead of going out; or point `c.resty.SetBaseURL` at
  an `httptest` server, as `v1/options_test.go` does. In `cmd/`, set `clientInstance` to a
  fake (`MockClient` in `cmd/measurements_test.go`): the `customer` and `thirdparty`
  commands build a real client whenever `clientInstance` is nil and `--token` is set, and
  `token --data-access` calls `/token`.
- No scheduled or automated traffic to Energinet: no workflow or script, and no test but
  the live tests, may call an Eloverblik or Energinet host, and only a person runs the live
  tests (`go test -tags live`), by hand. golangci-lint sets the `live` tag only to compile
  and lint them. The files in `docs/` are refreshed by hand, as `docs/README.md` describes.
- When sources disagree, trust them in this order: the live API (the live tests check the
  client against it), then the OpenAPI documents (`docs/swagger-eloverblik-*.json`), then
  the PDF technical description, which predates DataHub 3.0. Where the live API is known
  to differ from the documents, as with the time series resolutions, README.md and llms.md
  say so.
- Every behaviour change gets an entry in CHANGELOG.md, in the section of the next release
  (the topmost `## [x.y.z]` whose `vx.y.z` tag does not exist yet; open one above the others
  when the topmost is already tagged), and is documented in README.md and llms.md in the
  same change. The CHANGELOG section is published as the release notes. README.md
  ("Help Output") and llms.md ("Help Output (`go-eloverblik --help`)") each hold a copy of
  `go run . --help`; nothing checks them, so regenerate both when a command or flag changes.
- Do not remove an exported identifier within v1: deprecate it (`// Deprecated:` in its
  godoc) and leave the removal to v2. List in CHANGELOG.md whatever stops compiling, such as
  a method added to an interface.
- Commit messages are conventional: `type(scope): summary`, lower case. The types in use are
  `feat`, `fix`, `docs`, `test`, `refactor`, `ci` and `build`, with the scope `cli` for
  changes in `cmd/`. The body says what was wrong and why the change is the right fix.
- Every artifact is in English: code, comments, docs, commit messages. Danish text from
  Energinet, such as the CSV export headers or the wording of the guides in
  `docs/eloverblik-guides/`, is quoted verbatim and never translated, as `v1/meters.go`
  quotes "udgået".

## Live tests

`v1/live_test.go` checks the client against the production API with the refresh tokens
of the person running it. Only the `live` build tag compiles it in, and each test skips
unless its token is set, so CI and a plain `go test` never run it:

```sh
export ELO_CUSTOMER_TOKEN='...'    # a Customer API refresh token, and/or
export ELO_THIRDPARTY_TOKEN='...'  # a Third-Party API refresh token
go test -tags live -count=1 -v -run Live ./v1/
```

- A coding agent never runs them. It asks the maintainer to, and to share the output.
- They check the shape of every answer, never its values. A call that fails, a metering
  point ID or GLN without its format and check digit, an empty address field, a week of
  time series that is all zeros, or a field in a response that the client would drop
  fails the test. A field the client decodes that a response leaves out is only logged,
  because the API leaves out empty fields. The field comparison is `jsonFieldDrift` in
  `v1/responseshape_test.go`, which the ordinary tests cover.
- They read only metering points that should have data for last week: the customer test
  skips those the user has moved out of, the third-party test those whose access does
  not cover the week, and both read time series only for connected metering points.
- They only read. Never add a call that changes something, such as `AddRelationByID`.
- Their output must hold no customer data: log counts, codes, resolutions and field
  names, never a value from a response. Report errors through `describeShapeError` in
  `v1/responseshape_test.go`, which leaves out the values, URLs and IDs an error can
  quote; resty's own log is silenced for the same reason.
- A run sends at most about 15 requests per token, one of them to `/token`, and stops
  when no data access token could be fetched. The API allows 2 `/token` calls a minute
  per IP, so wait a minute between runs. `-count=1` keeps `go test` from replaying a
  cached pass.
- When the code or the docs change what they claim about the live API, change the check
  that verifies the claim with them.

## Release

Pushing a `v*` tag runs `.github/workflows/release.yml`: the test gate, then GoReleaser
(`release --clean`, configured in `.goreleaser.yaml`), which builds the archives and
checksums and creates the GitHub release. A tag with a pre-release suffix, such as
`v1.4.0-rc.1`, is published as a pre-release. The release build sets the version
`--version` prints with `-X github.com/slimcdk/go-eloverblik/cmd.version=v{{ .Version }}`;
the linker ignores `-X` for a symbol that does not exist, so check `--version` on a
snapshot build whenever that variable moves.

The release notes are the CHANGELOG.md section whose heading is exactly
`## [<tag without v>]`; GoReleaser appends the commits since the previous tag, grouped by
type. A tag without such a section still releases, with a warning and no notes of its own.
Preview the notes for a version before tagging:

```sh
awk -v heading="## [1.4.0]" '$0 == heading { inside = 1; next } inside && /^## \[/ { exit } inside { print }' CHANGELOG.md
```

Starting the workflow by hand is a dry run: a snapshot build that publishes nothing. Locally,
`goreleaser check` validates the config, and
`goreleaser build --snapshot --clean --single-target` builds for the current platform.
