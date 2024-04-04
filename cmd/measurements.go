package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/drewstinnett/gout/v2"
	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

// go run main.go --token=... timeseries --from 2024-01-01 [--to=2024-02-01] --aggregation=Day --flatten <metering-ids, ...> | jq
var timeseriesCmd = &cobra.Command{
	Use: "timeseries",

	Args: func(cmd *cobra.Command, args []string) error {
		// Optionally run one of the validators provided by cobra
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}

		if err := cobra.MaximumNArgs(10)(cmd, args); err != nil {
			return err
		}

		for i, meteringId := range args {
			if _, err := strconv.Atoi(meteringId); len(meteringId) != 18 || err != nil {
				return fmt.Errorf("provided metering id (number %d) looks like an invalid id: %s", i, meteringId)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cobra.CheckErr(gout.BindCobra(cmd, nil))
		fromFlag, _ := cmd.Flags().GetString("from")
		toFlag, _ := cmd.Flags().GetString("to")
		aggregation, _ := cmd.Flags().GetString("aggregation")
		flatten, _ := cmd.Flags().GetBool("flatten")

		from, fromErr := time.Parse(time.DateOnly, fromFlag)
		to, toErr := time.Parse(time.DateOnly, toFlag)
		cobra.CheckErr(errors.Join(fromErr, toErr))

		tss, err := ec.GetTimeSeries(args, from, to, eloverblik.Aggregation(aggregation))
		cobra.CheckErr(err)

		if !flatten {
			gout.MustPrint(tss)
		} else {
			flattened := make(map[string][]eloverblik.FlatTimeSeriesPoint, len(args))
			for _, ts := range tss {
				flattened[ts.ID] = ts.Flatten()
			}
			gout.MustPrint(flattened)
		}
	},
}

func init() {
	timeseriesCmd.Flags().String("from", "", "start date for timeseries")
	timeseriesCmd.Flags().String("to", time.Now().Format(time.DateOnly), "end date for request. defaults to now")
	timeseriesCmd.Flags().String("aggregation", string(eloverblik.Hour), "aggregation for timeseries")
	timeseriesCmd.Flags().Bool("flatten", false, "simplify the data series")

	rootCmd.AddCommand(timeseriesCmd)
}
