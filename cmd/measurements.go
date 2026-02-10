package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/drewstinnett/gout/v2"
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// output is the destination for export commands (configurable for testing)
var output io.Writer = os.Stdout

func meteringPointArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
		return err
	}
	if err := cobra.MaximumNArgs(10)(cmd, args); err != nil {
		return err
	}
	for i, id := range args {
		if _, err := strconv.Atoi(id); len(id) != 18 || err != nil {
			return fmt.Errorf("provided metering id (number %d) looks like an invalid id: %s", i, id)
		}
	}
	return nil
}

// csvToJSON converts a CSV stream to JSON format
func csvToJSON(stream io.ReadCloser) error {
	defer stream.Close()

	reader := csv.NewReader(stream)
	reader.Comma = ';' // Eloverblik CSV uses semicolon delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields per record

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var records []map[string]string

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

// outputStream outputs the stream as either CSV or JSON based on format flag
func outputStream(stream io.ReadCloser, format string) error {
	if format == "json" {
		return csvToJSON(stream)
	}
	// Default CSV output
	defer stream.Close()
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
			gout.MustPrint(details)
		},
	}
}

func newTimeseriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeseries <metering-id> [metering-id ...]",
		Short: "Get time series for one or more metering points",
		Args:  meteringPointArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fromFlag, _ := cmd.Flags().GetString("from")
			toFlag, _ := cmd.Flags().GetString("to")
			aggregation, _ := cmd.Flags().GetString("aggregation")
			flatten, _ := cmd.Flags().GetBool("flatten")

			from, fromErr := time.Parse(time.DateOnly, fromFlag)
			to, toErr := time.Parse(time.DateOnly, toFlag)
			cobra.CheckErr(errors.Join(fromErr, toErr))

			tss, err := clientInstance.GetTimeSeries(args, from, to, eloverblik.Aggregation(aggregation))
			cobra.CheckErr(err)

			if !flatten {
				gout.MustPrint(tss)
			} else {
				flattened := make(map[string][]eloverblik.FlatTimeSeriesPoint, len(args))
				for _, ts := range tss {
					id := ts.MyEnergyDataMarketDocument.TimeSeries[0].MRID
					flattened[id] = ts.Flatten()
				}
				gout.MustPrint(flattened)
			}
		},
	}
	cmd.Flags().String("from", "", "start date (YYYY-MM-DD, required)")
	cmd.Flags().String("to", time.Now().Format(time.DateOnly), "end date (YYYY-MM-DD, defaults to today)")
	cmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation level (Actual, Quarter, Hour, Day, Month, Year)")
	cmd.Flags().Bool("flatten", false, "simplify the data series")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func newExportTimeseriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-timeseries <metering-id> [metering-id ...]",
		Short: "Export time series as a raw stream (customer API only)",
		Args:  meteringPointArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fromFlag, _ := cmd.Flags().GetString("from")
			toFlag, _ := cmd.Flags().GetString("to")
			aggregation, _ := cmd.Flags().GetString("aggregation")
			format, _ := cmd.Flags().GetString("format")

			from, fromErr := time.Parse(time.DateOnly, fromFlag)
			to, toErr := time.Parse(time.DateOnly, toFlag)
			cobra.CheckErr(errors.Join(fromErr, toErr))

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
	cmd.Flags().String("from", "", "start date (YYYY-MM-DD, required)")
	cmd.Flags().String("to", time.Now().Format(time.DateOnly), "end date (YYYY-MM-DD, defaults to today)")
	cmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation level (Actual, Quarter, Hour, Day, Month, Year)")
	cmd.Flags().String("format", "csv", "output format (csv, json)")
	_ = cmd.MarkFlagRequired("from")
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
