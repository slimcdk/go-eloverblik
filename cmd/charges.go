package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/spf13/cobra"
)

var customerChargesCmd = &cobra.Command{
	Use:   "charges <metering-id> [metering-id ...]",
	Short: "Get charges (subscriptions, fees, tariffs) for one or more metering points",
	Long: `Get the charges of 1 to 10 metering points: their subscriptions, fees and tariffs, with
prices. Only the charges that are valid now or take effect later are returned, never past
prices, so they cannot price consumption that already happened.

Calls POST /meteringpoints/meteringpoint/getcharges on the Customer API.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Output: a JSON array with one object per metering point:
  {"result": {"meteringPointId",
              "subscriptions": [{"priceId", "name", "description", "owner",
                                 "validFromDate", "validToDate", "periodType", "price",
                                 "quantity"}],
              "fees": [same shape as subscriptions],
              "tariffs": [{"priceId", "name", "description", "owner", "validFromDate",
                           "validToDate", "periodType",
                           "prices": [{"position", "price"}]}]},
   "success", "errorCode", "errorText", "id", "stackTrace"}
A metering point that failed has "success": false and the reason in "errorCode" and
"errorText", while the command still succeeds. A date the API leaves empty, such as a
missing "validToDate", is null.`,
	Example: `  go-eloverblik customer charges 571313000000000001 --token "$ELO_TOKEN"
  go-eloverblik customer charges 571313000000000001 571313000000000002 --token "$ELO_TOKEN" \
    | jq '.[] | select(.success) | .result.tariffs[]? | {name, validFromDate, prices}'`,
	Args: meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("charges can only be used with the 'customer' subcommand"))
		}
		charges, err := customerAPI.GetCustomerCharges(args)
		cobra.CheckErr(err)
		bytes, err := json.Marshal(charges)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

var thirdpartyChargesCmd = &cobra.Command{
	Use:   "charges <metering-id> [metering-id ...]",
	Short: "Get charges (subscriptions, tariffs) for one or more metering points",
	Long: `Get the charges of 1 to 10 metering points: their subscriptions and tariffs, with prices.
The Third-Party API returns no fees. Only the charges that are valid now or take effect
later are returned, never past prices, so they cannot price consumption that already
happened.

Calls POST /meteringpoint/getcharges on the Third-Party API.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Output: a JSON array with one object per metering point:
  {"result": {"meteringPointId",
              "subscriptions": [{"priceId", "name", "description", "owner",
                                 "validFromDate", "validToDate", "periodType", "price",
                                 "quantity"}],
              "tariffs": [{"priceId", "name", "description", "owner", "validFromDate",
                           "validToDate", "periodType",
                           "prices": [{"position", "price"}]}]},
   "success", "errorCode", "errorText", "id", "stackTrace"}
A metering point that failed has "success": false and the reason in "errorCode" and
"errorText", while the command still succeeds. A date the API leaves empty, such as a
missing "validToDate", is null.`,
	Example: `  go-eloverblik thirdparty charges 571313000000000001 --token "$ELO_TOKEN"
  go-eloverblik thirdparty charges 571313000000000001 571313000000000002 --token "$ELO_TOKEN" \
    | jq '.[] | select(.success) | .result.tariffs[]? | {name, validFromDate, prices}'`,
	Args: meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		thirdpartyAPI, ok := clientInstance.(eloverblik.ThirdParty)
		if !ok {
			cobra.CheckErr(fmt.Errorf("charges can only be used with the 'thirdparty' subcommand"))
		}
		charges, err := thirdpartyAPI.GetThirdPartyCharges(args)
		cobra.CheckErr(err)
		bytes, err := json.Marshal(charges)
		cobra.CheckErr(err)
		_, err = output.Write(bytes)
		cobra.CheckErr(err)
	},
}

var exportChargesCmd = &cobra.Command{
	Use:   "export-charges <metering-id> [metering-id ...]",
	Short: "Export charges (customer API only)",
	Long: `Export the charges of 1 to 10 metering points as the CSV file Eloverblik generates.

Calls POST /meteringpoints/charges/export on the Customer API.

Arguments: 1 to 10 metering point IDs, each exactly 18 digits.

Output with --format csv (the default): the API's CSV, unchanged: separated by
semicolons, starting with a UTF-8 byte order mark, with Danish column names.
Output with --format json: the rows as a JSON array of objects keyed by the CSV header,
every value a string, without the white space around it. The byte order mark is
dropped, and a CSV without rows gives [].
Any other --format gives the CSV.`,
	Example: `  go-eloverblik customer export-charges 571313000000000001 --token "$ELO_TOKEN" > charges.csv
  go-eloverblik customer export-charges 571313000000000001 571313000000000002 --format json --token "$ELO_TOKEN"`,
	Args: meteringPointArgs,
	Run: func(cmd *cobra.Command, args []string) {
		format, _ := cmd.Flags().GetString("format")

		customerAPI, ok := clientInstance.(eloverblik.Customer)
		if !ok {
			cobra.CheckErr(fmt.Errorf("export-charges can only be used with the 'customer' subcommand"))
		}

		stream, err := customerAPI.ExportCharges(args)
		cobra.CheckErr(err)

		err = outputStream(stream, format)
		cobra.CheckErr(err)
	},
}

func init() {
	customerCmd.AddCommand(customerChargesCmd)
	exportChargesCmd.Flags().String("format", "csv", "output format: csv (the API's CSV, unchanged) or json (an array of objects keyed by the CSV header)")
	customerCmd.AddCommand(exportChargesCmd)
	thirdpartyCmd.AddCommand(thirdpartyChargesCmd)
}
