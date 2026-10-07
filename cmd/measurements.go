package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// output is where this package's commands write the JSON or CSV they print (configurable
// for testing). alive is the exception: it prints its line of text to os.Stdout directly.
// cobra's own help and completion commands do not use it either.
var output io.Writer = os.Stdout

// warningOutput receives what a command reports about a call that still succeeded, such
// as a metering point that failed on its own (configurable for testing). It defaults to
// stderr so stdout stays clean, parseable JSON.
var warningOutput io.Writer = os.Stderr

// clock is what parseDate and today read the current time from (configurable for testing).
var clock = time.Now

// today is the date the --to flags default to: the current date in Copenhagen, the zone
// parseDate reads it back in, whatever the host's own zone. Around midnight the host's own
// date can be a day behind or ahead of it.
func today() string {
	return clock().In(copenhagen).Format(time.DateOnly)
}

// copenhagen is the zone parseDate reads dates in. The v1 package embeds time/tzdata, so
// it loads on every platform, Windows and minimal container images included.
var copenhagen = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Copenhagen")
	if err != nil {
		panic(fmt.Errorf("load time zone Europe/Copenhagen: %w", err))
	}
	return loc
}()

func meteringPointArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
		return err
	}
	if err := cobra.MaximumNArgs(10)(cmd, args); err != nil {
		return err
	}
	for i, id := range args {
		if !isMeteringPointID(id) {
			return fmt.Errorf("provided metering id (number %d) looks like an invalid id: %s", i, id)
		}
	}
	return nil
}

// isMeteringPointID reports whether id has the shape of a metering point ID: exactly 18
// ASCII digits. It checks the text instead of parsing a number: 18 digits do not fit in
// the int of a 32-bit platform such as linux/arm, where parsing rejected every valid ID,
// and a parser would accept a sign as well.
func isMeteringPointID(id string) bool {
	return len(id) == 18 && !strings.ContainsFunc(id, func(r rune) bool { return r < '0' || r > '9' })
}

// csvToJSON converts a CSV stream to a JSON array with one object per row, keyed by the
// header. Eloverblik's CSV starts with a UTF-8 byte order mark, which is dropped, so the
// first key is the header's own name; a CSV with a header and no rows gives [].
func csvToJSON(stream io.ReadCloser) error {
	defer func() { _ = stream.Close() }()

	// Drop the byte order mark before the CSV reader sees it: left in the first field, it
	// would also keep a quoted header from being read as quoted.
	body := bufio.NewReader(stream)
	mark, _, err := body.ReadRune()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}
	if mark != '\uFEFF' {
		_ = body.UnreadRune()
	}

	reader := csv.NewReader(body)
	reader.Comma = ';' // Eloverblik CSV uses semicolon delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields per record

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	// Not nil: no rows encode as [], not null
	records := []map[string]string{}

	// Read all data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		// The exports send every MålepunktsID with a tab before it. No value keeps white
		// space on either side
		row := make(map[string]string)
		for i, value := range record {
			if i < len(headers) {
				row[headers[i]] = strings.TrimSpace(value)
			}
		}
		records = append(records, row)
	}

	// Output as JSON
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

// parseDate reads a date given on the command line. Eloverblik counts in Danish days, so
// every form is read in Copenhagen time, whatever the host's own zone:
//   - YYYY-MM-DD: midnight at the start of that day in Copenhagen
//   - now: the current time
//   - now-30d, now-4w, now-2m, now-1y: the current time, that many days, weeks, months or
//     years back on the Copenhagen calendar
//
// The result matters for charge-links, which sends timestamps: 2026-07-01 goes out as
// 2026-07-01T00:00:00+02:00. timeseries and export-timeseries send only the Copenhagen
// date of the result.
func parseDate(dateStr string) (time.Time, error) {
	now := clock().In(copenhagen)
	if dateStr == "now" {
		return now, nil
	}

	// Standard date format
	if t, err := time.ParseInLocation(time.DateOnly, dateStr, copenhagen); err == nil {
		return t, nil
	}

	// Relative date format (e.g., now-30d)
	re := regexp.MustCompile(`^now-(\d+)([dwmy])$`)
	matches := re.FindStringSubmatch(strings.ToLower(dateStr))

	if len(matches) == 3 {
		value, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		switch unit {
		case "d":
			return now.AddDate(0, 0, -value), nil
		case "w":
			return now.AddDate(0, 0, -value*7), nil
		case "m":
			return now.AddDate(0, -value, 0), nil
		case "y":
			return now.AddDate(-value, 0, 0), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format: '%s'. Use YYYY-MM-DD, now, or now-Xd/w/m/y", dateStr)
}

// outputStream outputs the stream as either CSV or JSON based on format flag
func outputStream(stream io.ReadCloser, format string) error {
	if format == "json" {
		return csvToJSON(stream)
	}
	// Default CSV output
	defer func() { _ = stream.Close() }()
	_, err := io.Copy(output, stream)
	return err
}

func newDetailsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "details <metering-id> [metering-id ...]",
		Short: "Get metering point details",
		Long: `Get the master data of 1 to 10 metering points: type, grid operator, address, meter,
balance supplier, contact addresses and child metering points.

Calls POST /meteringpoints/meteringpoint/getdetails on the Customer API, or
POST /meteringpoint/getdetails on the Third-Party API.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Output: a JSON array with one object per metering point:
  {"result": {"meteringPointId", "typeOfMP", "energyTimeSeriesMeasureUnit",
              "gridOperatorName", "meteringGridAreaIdentification", "physicalStatusOfMP",
              "meterNumber", "streetName", "buildingNumber", "postcode", "cityName",
              "balanceSupplierName", "contactAddresses": [...],
              "childMeteringPoints": [...], ...},
   "success", "errorCode", "errorText", "id", "stackTrace"}
A metering point that failed has "success": false and the reason in "errorCode" and
"errorText", while the command still succeeds. The same metering point can come back once
per access period. Energinet lists several fields as retired or unavailable for now, e.g.
"settlementMethod" and "consumerStartDate": expect them empty, a date as null. The balance
supplier fields are not shared with a third party.`,
		Example: `  go-eloverblik customer details 571313000000000001 --token "$ELO_TOKEN"
  go-eloverblik thirdparty details 571313000000000001 571313000000000002 --token "$ELO_TOKEN"

  # The metering points that failed on their own
  go-eloverblik customer details 571313000000000001 571313000000000002 --token "$ELO_TOKEN" \
    | jq '.[] | select(.success == false) | {id, errorCode, errorText}'`,
		Args: meteringPointArgs,
		Run: func(cmd *cobra.Command, args []string) {
			details, err := clientInstance.GetMeteringPointDetails(args)
			cobra.CheckErr(err)
			bytes, err := json.Marshal(details)
			cobra.CheckErr(err)
			_, err = output.Write(bytes)
			cobra.CheckErr(err)
		},
	}
}

func newTimeseriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeseries <metering-id> [metering-id ...]",
		Short: "Get time series for one or more metering points",
		Long: `Get the metered quantities (time series) of 1 to 10 metering points in the half-open range
[from, to), at the given aggregation.

Calls POST /meterdata/gettimeseries/{from}/{to}/{aggregation} on the API of the parent
command, with from and to as Copenhagen calendar dates (YYYY-MM-DD).

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Dates: give either --period, or --from with an optional --to, not both.
  --from    The first day, included: YYYY-MM-DD, now, or now-Nd, now-Nw, now-Nm or now-Ny,
            N days, weeks, months or years back on the Copenhagen calendar.
  --to      The day after the last one, excluded, in the same forms. It defaults to today,
            so the range ends with yesterday. --from 2026-09-01 --to 2026-10-01 is all of
            September.
  --period  yesterday, this_week, last_week, this_month, last_month, this_year or
            last_year. Weeks start on Monday. A this_* period ends with yesterday, and is
            an error on its first day (a Monday, the 1st, 1 January), which has no
            complete day yet.
From and to on the same date is rejected (API error 30002), and a range longer than 730
days too (30014).

--aggregation is sent as given: Actual, Quarter, Hour (the default), Day, Month or Year.

Output without --flatten: the API's document, a JSON array with one object per metering
point:
  {"MyEnergyData_MarketDocument": {"TimeSeries": [{"mRID", "businessType", "curveType",
     "measurement_Unit.name", "Period": [{"resolution", "timeInterval": {"start", "end"},
     "point": [{"position", "out_Quantity.quantity", "out_Quantity.quality"}]}]}], ...},
   "success", "errorCode", "errorText", "id", "stackTrace"}
"position" and "out_Quantity.quantity" are JSON strings, e.g. "0.198". A metering point
that failed on its own has "success": false, the reason in "errorCode" and "errorText",
and no time series, while the command still succeeds.

Output with --flatten: a JSON object keyed by metering point ID, each holding a list of
  {"from", "to", "measurement", "quality", "unit", "curvetype", "businesstype",
   "resolution"}
with "from" and "to" as RFC 3339 times in Copenhagen time and "measurement" as a number.
A metering point that failed on its own is left out and reported as a warning on stderr.

The same metering point can come back once per access period: without --flatten as
several objects, with --flatten as one list holding the points of all of them.`,
		Example: `  # All of September, one value per day
  go-eloverblik customer timeseries 571313000000000001 --from 2026-09-01 --to 2026-10-01 --aggregation Day --token "$ELO_TOKEN"

  # Last month, hourly, as one flat list per metering point
  go-eloverblik customer timeseries 571313000000000001 571313000000000002 --period last_month --flatten --token "$ELO_TOKEN"

  # The last 30 days up to and including yesterday
  go-eloverblik thirdparty timeseries 571313000000000001 --from now-30d --aggregation Day --flatten --token "$ELO_TOKEN"`,
		Args: meteringPointArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			period, _ := cmd.Flags().GetString("period")

			// Check for mutual exclusivity and requirements
			if period != "" {
				if cmd.Flags().Changed("from") || cmd.Flags().Changed("to") {
					return errors.New("--period cannot be used with --from or --to")
				}
			} else {
				if !cmd.Flags().Changed("from") {
					return errors.New("either --period or --from is required")
				}
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			period, _ := cmd.Flags().GetString("period")
			fromFlag, _ := cmd.Flags().GetString("from")
			toFlag, _ := cmd.Flags().GetString("to")
			aggregation, _ := cmd.Flags().GetString("aggregation")
			flatten, _ := cmd.Flags().GetBool("flatten")

			var from, to time.Time
			var err error

			if period != "" {
				from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.Period(period))
				cobra.CheckErr(err)
			} else {
				from, err = parseDate(fromFlag)
				cobra.CheckErr(err)
				to, err = parseDate(toFlag)
				cobra.CheckErr(err)
			}

			tss, err := clientInstance.GetTimeSeries(args, from, to, eloverblik.Aggregation(aggregation))
			cobra.CheckErr(err)

			if !flatten {
				bytes, err := json.Marshal(tss)
				cobra.CheckErr(err)
				_, err = output.Write(bytes)
				cobra.CheckErr(err)
			} else {
				flattened := make(map[string][]eloverblik.FlatTimeSeriesPoint, len(args))
				for _, ts := range tss {
					// A metering point that failed on its own has no market document to
					// flatten. Report it and keep the others.
					if err := ts.Err(); err != nil {
						_, _ = fmt.Fprintf(warningOutput, "warning: %v\n", err)
						continue
					}
					// The API can report a metering point once per access period, so the
					// same ID may come back more than once.
					flattened[ts.ID] = append(flattened[ts.ID], ts.Flatten()...)
				}
				bytes, err := json.Marshal(flattened)
				cobra.CheckErr(err)
				_, err = output.Write(bytes)
				cobra.CheckErr(err)
			}
		},
	}
	cmd.Flags().String("from", "", "start date, inclusive (YYYY-MM-DD, now, now-30d/w/m/y); required unless --period is given")
	cmd.Flags().String("to", today(), "end date, exclusive (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
	cmd.Flags().String("period", "", "named range instead of --from and --to: yesterday, this_week, last_week, this_month, last_month, this_year or last_year")
	cmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation level (Actual, Quarter, Hour, Day, Month, Year)")
	cmd.Flags().Bool("flatten", false, "print one flat list of readings per metering point instead of the API's nested document")
	return cmd
}

func newExportTimeseriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-timeseries <metering-id> [metering-id ...]",
		Short: "Export time series as CSV or JSON (customer API only)",
		Long: `Export the time series of 1 to 10 metering points in the half-open range [from, to) as the
CSV file Eloverblik generates.

Calls POST /meterdata/timeseries/export/{from}/{to}/{aggregation} on the Customer API,
with from and to as Copenhagen calendar dates (YYYY-MM-DD).

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

--from, --to, --period and --aggregation work as for timeseries (see
"go-eloverblik customer timeseries --help"): give either --period, or --from with an
optional --to; --to is excluded and defaults to today, so the range ends with yesterday.
From and to on the same date is rejected (API error 30002), and a range longer than 730
days too (30014).

Output with --format csv (the default): the API's CSV, unchanged: separated by
semicolons, starting with a UTF-8 byte order mark, with Danish column names.
Output with --format json: the rows as a JSON array of objects keyed by the CSV header,
every value a string, without the white space around it. The byte order mark is
dropped, and a CSV without rows gives [].
Any other --format gives the CSV.`,
		Example: `  go-eloverblik customer export-timeseries 571313000000000001 --from 2026-09-01 --to 2026-10-01 --token "$ELO_TOKEN" > september.csv
  go-eloverblik customer export-timeseries 571313000000000001 571313000000000002 --period last_month --aggregation Day --format json --token "$ELO_TOKEN"`,
		Args: meteringPointArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			period, _ := cmd.Flags().GetString("period")

			// Check for mutual exclusivity and requirements
			if period != "" {
				if cmd.Flags().Changed("from") || cmd.Flags().Changed("to") {
					return errors.New("--period cannot be used with --from or --to")
				}
			} else {
				if !cmd.Flags().Changed("from") {
					return errors.New("either --period or --from is required")
				}
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			period, _ := cmd.Flags().GetString("period")
			fromFlag, _ := cmd.Flags().GetString("from")
			toFlag, _ := cmd.Flags().GetString("to")
			aggregation, _ := cmd.Flags().GetString("aggregation")
			format, _ := cmd.Flags().GetString("format")

			var from, to time.Time
			var err error

			if period != "" {
				from, to, err = eloverblik.GetDatesFromPeriod(eloverblik.Period(period))
				cobra.CheckErr(err)
			} else {
				from, err = parseDate(fromFlag)
				cobra.CheckErr(err)
				to, err = parseDate(toFlag)
				cobra.CheckErr(err)
			}

			customerAPI, ok := clientInstance.(eloverblik.Customer)
			if !ok {
				cobra.CheckErr(fmt.Errorf("export-timeseries can only be used with the 'customer' subcommand"))
			}

			stream, err := customerAPI.ExportTimeSeries(args, from, to, eloverblik.Aggregation(aggregation))
			cobra.CheckErr(err)

			err = outputStream(stream, format)
			cobra.CheckErr(err)
		},
	}
	cmd.Flags().String("from", "", "start date, inclusive (YYYY-MM-DD, now, now-30d/w/m/y); required unless --period is given")
	cmd.Flags().String("to", today(), "end date, exclusive (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
	cmd.Flags().String("period", "", "named range instead of --from and --to: yesterday, this_week, last_week, this_month, last_month, this_year or last_year")
	cmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation level (Actual, Quarter, Hour, Day, Month, Year)")
	cmd.Flags().String("format", "csv", "output format: csv (the API's CSV, unchanged) or json (an array of objects keyed by the CSV header)")
	return cmd
}

func newExportMasterdataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-masterdata <metering-id> [metering-id ...]",
		Short: "Export metering point masterdata (customer API only)",
		Long: `Export the master data of 1 to 10 metering points as the CSV file Eloverblik generates.
For master data as JSON from the API, use details.

Calls POST /meteringpoints/masterdata/export on the Customer API.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Output with --format csv (the default): the API's CSV, unchanged: separated by
semicolons, starting with a UTF-8 byte order mark, with Danish column names.
Output with --format json: the rows as a JSON array of objects keyed by the CSV header,
every value a string, without the white space around it. The byte order mark is
dropped, and a CSV without rows gives [].
Any other --format gives the CSV.`,
		Example: `  go-eloverblik customer export-masterdata 571313000000000001 --token "$ELO_TOKEN" > masterdata.csv
  go-eloverblik customer export-masterdata 571313000000000001 571313000000000002 --format json --token "$ELO_TOKEN"`,
		Args: meteringPointArgs,
		Run: func(cmd *cobra.Command, args []string) {
			format, _ := cmd.Flags().GetString("format")

			customerAPI, ok := clientInstance.(eloverblik.Customer)
			if !ok {
				cobra.CheckErr(fmt.Errorf("export-masterdata can only be used with the 'customer' subcommand"))
			}

			stream, err := customerAPI.ExportMasterdata(args)
			cobra.CheckErr(err)

			err = outputStream(stream, format)
			cobra.CheckErr(err)
		},
	}
	cmd.Flags().String("format", "csv", "output format: csv (the API's CSV, unchanged) or json (an array of objects keyed by the CSV header)")
	return cmd
}

func init() {
	customerCmd.AddCommand(newDetailsCmd())
	customerCmd.AddCommand(newTimeseriesCmd())
	customerCmd.AddCommand(newExportTimeseriesCmd())
	customerCmd.AddCommand(newExportMasterdataCmd())

	thirdpartyCmd.AddCommand(newDetailsCmd())
	thirdpartyCmd.AddCommand(newTimeseriesCmd())
}
