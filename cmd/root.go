package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	eloverblik "github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// clientInstance will hold the instantiated client (either Customer or ThirdParty)
var clientInstance eloverblik.Client

// headerOutput is the destination for the HTTP response headers printed with
// --print-response-headers (configurable for testing). It defaults to stderr so
// stdout stays clean, parseable JSON.
var headerOutput io.Writer = os.Stderr

var rootCmd = &cobra.Command{
	Use:   "go-eloverblik",
	Short: "A CLI for the Danish Eloverblik platform",
	Long: `A CLI for the Danish Eloverblik platform: electricity metering data from Energinet's
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
    --from 2026-09-01 --to 2026-10-01 is all of September. export-timeseries is the
    exception: the export includes --to too. For timeseries and export-timeseries, from
    and to on the same date is rejected (API error 30002).
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
https://github.com/slimcdk/go-eloverblik/blob/master/llms.md`,
	Example: `  export ELO_TOKEN='<refresh token from eloverblik.dk>'
  go-eloverblik token --token "$ELO_TOKEN"
  go-eloverblik customer installations --token "$ELO_TOKEN"
  go-eloverblik customer timeseries 571313000000000001 --from 2026-09-01 --to 2026-10-01 --aggregation Day --token "$ELO_TOKEN"
  go-eloverblik thirdparty authorizations --token "$ELO_TOKEN"
  go-eloverblik customer timeseries --help`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// refreshToken returns the refresh token given with --token, or an error when none was
// given. The commands that use the token call it instead of having cobra require the
// flag, which cobra would then demand of every command, help and completion included.
func refreshToken(cmd *cobra.Command) (string, error) {
	token, err := cmd.Root().PersistentFlags().GetString("token")
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", errors.New(`required flag "token" not set`)
	}
	return token, nil
}

// helpFunc returns the help for the whole command tree. A help function set on the root
// is inherited by every command below it, so it has to pick: the root lists each group's
// commands under the group's name, and every other command gets cobra's default help,
// which is what tells how to call it: the usage line with its arguments, the Long text,
// the examples, its flags and the global flags such as --token.
func helpFunc(defaultHelp func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.HasParent() {
			defaultHelp(cmd, args)
			return
		}
		rootHelpFunc(cmd, args)
	}
}

// rootHelpFunc prints the root help: the Long text, which carries the rules an agent with
// only the binary needs, the usage line and the examples, then the commands of each group
// listed under the group's name, and the root's own flags. It is cobra's order, with the
// grouped listing in place of cobra's flat one.
func rootHelpFunc(cmd *cobra.Command, _ []string) {
	w := cmd.OutOrStdout()
	printf := func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, format, args...)
	}

	about := cmd.Long
	if about == "" {
		about = cmd.Short
	}
	printf("%s\n\nUsage:\n  %s [command]\n", about, cmd.CommandPath())
	if cmd.HasExample() {
		printf("\nExamples:\n%s\n", cmd.Example)
	}
	printf("\nAvailable Commands:\n")
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.Name() == "help" {
			continue
		}
		if sub.HasAvailableSubCommands() {
			printf("\n  %s\n", sub.Name())
			for _, subsub := range sub.Commands() {
				if !subsub.IsAvailableCommand() {
					continue
				}
				printf("    %-24s %s\n", subsub.Name(), subsub.Short)
			}
		} else {
			printf("  %-26s %s\n", sub.Name(), sub.Short)
		}
	}
	if flags := cmd.LocalFlags().FlagUsages(); flags != "" {
		printf("\nFlags:\n%s", flags)
	}
	printf("\nUse \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
}

func init() {
	// Setting Version gives the root command cobra's --version and -v flags.
	info, ok := debug.ReadBuildInfo()
	rootCmd.Version = cliVersion(version, info, ok)

	// --token is not marked required: the commands that use it check it with refreshToken.
	rootCmd.PersistentFlags().String("token", "", "Eloverblik refresh token, created at eloverblik.dk (required by the customer, thirdparty and token commands)")
	rootCmd.PersistentFlags().Bool("print-response-headers", false, "Print HTTP response headers from the Eloverblik API to stderr")
	rootCmd.SetHelpFunc(helpFunc(rootCmd.HelpFunc()))
}
