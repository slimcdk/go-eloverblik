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

// output is where every command writes its result (configurable for testing). alive is the
// exception: it prints to os.Stdout directly.
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

		row := make(map[string]string)
		for i, value := range record {
			if i < len(headers) {
				row[headers[i]] = value
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
		Args:  meteringPointArgs,
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
		Args:  meteringPointArgs,
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
	cmd.Flags().String("from", "", "start date (YYYY-MM-DD, now, now-30d/w/m/y)")
	cmd.Flags().String("to", today(), "end date, exclusive (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
	cmd.Flags().String("period", "", "predefined period (yesterday, last_week, etc.)")
	cmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation level (Actual, Quarter, Hour, Day, Month, Year)")
	cmd.Flags().Bool("flatten", false, "simplify the data series")
	return cmd
}

func newExportTimeseriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-timeseries <metering-id> [metering-id ...]",
		Short: "Export time series as CSV or JSON (customer API only)",
		Args:  meteringPointArgs,
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
	cmd.Flags().String("from", "", "start date (YYYY-MM-DD, now, now-30d/w/m/y)")
	cmd.Flags().String("to", today(), "end date, exclusive (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
	cmd.Flags().String("period", "", "predefined period (yesterday, last_week, etc.)")
	cmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation level (Actual, Quarter, Hour, Day, Month, Year)")
	cmd.Flags().String("format", "csv", "output format (csv, json)")
	return cmd
}

func newExportMasterdataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-masterdata <metering-id> [metering-id ...]",
		Short: "Export metering point masterdata (customer API only)",
		Args:  meteringPointArgs,
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
	cmd.Flags().String("format", "csv", "output format (csv, json)")
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
