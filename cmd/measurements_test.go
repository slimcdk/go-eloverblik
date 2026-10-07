package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func TestCsvToJSON(t *testing.T) {
	t.Run("converts simple CSV to JSON", func(t *testing.T) {
		// CSV with BOM (U+FEFF) at the start, as Eloverblik sends it
		csvData := "\uFEFFMålepunktsID;Fra_dato;Til_dato;Mængde\n571313155411053087;01-02-2026 00:00:00;01-02-2026 01:00:00;0,198\n571313155411053087;01-02-2026 01:00:00;01-02-2026 02:00:00;0,196"
		stream := nopCloser{strings.NewReader(csvData)}

		// Capture stdout
		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := csvToJSON(stream)
		require.NoError(t, err)

		// Parse the JSON output
		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		require.NoError(t, err)
		assert.Len(t, records, 2)

		// Check first record (the BOM is not part of the first header)
		assert.Equal(t, "571313155411053087", records[0]["MålepunktsID"])
		assert.Equal(t, "01-02-2026 00:00:00", records[0]["Fra_dato"])
		assert.Equal(t, "01-02-2026 01:00:00", records[0]["Til_dato"])
		assert.Equal(t, "0,198", records[0]["Mængde"])

		// Check second record
		assert.Equal(t, "571313155411053087", records[1]["MålepunktsID"])
		assert.Equal(t, "0,196", records[1]["Mængde"])
	})

	t.Run("handles empty CSV", func(t *testing.T) {
		csvData := "Header1;Header2;Header3"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := csvToJSON(stream)
		require.NoError(t, err)

		// An empty array, not null: a consumer can iterate the result without a nil check
		assert.JSONEq(t, `[]`, buf.String())
	})

	t.Run("handles CSV with special characters", func(t *testing.T) {
		csvData := "Navn;Beskrivelse\nNet abo C;Net abo C forbrug flex - stikledning\nTSO - System;Abonnement TSO"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := csvToJSON(stream)
		require.NoError(t, err)

		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		require.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, "Net abo C", records[0]["Navn"])
		assert.Equal(t, "Net abo C forbrug flex - stikledning", records[0]["Beskrivelse"])
	})

	t.Run("handles CSV with mismatched columns", func(t *testing.T) {
		csvData := "Col1;Col2;Col3\nA;B;C\nD;E"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := csvToJSON(stream)
		require.NoError(t, err)

		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		require.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, "A", records[0]["Col1"])
		assert.Equal(t, "C", records[0]["Col3"])
		assert.Equal(t, "D", records[1]["Col1"])
		assert.Equal(t, "E", records[1]["Col2"])
		// Col3 should not exist in second record
		_, exists := records[1]["Col3"]
		assert.False(t, exists)
	})
}

func TestOutputStream(t *testing.T) {
	t.Run("outputs CSV format by default", func(t *testing.T) {
		csvData := "Header1;Header2\nValue1;Value2"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := outputStream(stream, "csv")
		require.NoError(t, err)

		// Should output raw CSV
		assert.Equal(t, csvData, buf.String())
	})

	t.Run("outputs JSON format when specified", func(t *testing.T) {
		csvData := "Name;Age\nJohn;30\nJane;25"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := outputStream(stream, "json")
		require.NoError(t, err)

		// Should output JSON
		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		require.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, "John", records[0]["Name"])
		assert.Equal(t, "30", records[0]["Age"])
	})

	t.Run("handles unknown format as CSV", func(t *testing.T) {
		csvData := "A;B\n1;2"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := outputStream(stream, "unknown")
		require.NoError(t, err)

		// Should default to CSV output
		assert.Equal(t, csvData, buf.String())
	})
}

func TestMeteringPointArgs(t *testing.T) {
	t.Run("accepts valid metering point IDs", func(t *testing.T) {
		args := []string{"571313155411053087"}
		err := meteringPointArgs(nil, args)
		assert.NoError(t, err)
	})

	t.Run("accepts multiple valid IDs", func(t *testing.T) {
		args := []string{"571313155411053087", "571313155411782079"}
		err := meteringPointArgs(nil, args)
		assert.NoError(t, err)
	})

	t.Run("rejects IDs with wrong length", func(t *testing.T) {
		args := []string{"12345"}
		err := meteringPointArgs(nil, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id")
	})

	t.Run("rejects non-numeric IDs", func(t *testing.T) {
		args := []string{"57131315541105308a"}
		err := meteringPointArgs(nil, args)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id")
	})

	// An ID is 18 digits, not a number: a sign must not pass for one.
	for _, id := range []string{"+57131315541105308", "-57131315541105308"} {
		t.Run("rejects signed ID "+id, func(t *testing.T) {
			err := meteringPointArgs(nil, []string{id})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid id")
		})
	}

	t.Run("requires at least one ID", func(t *testing.T) {
		args := []string{}
		err := meteringPointArgs(nil, args)
		assert.Error(t, err)
	})

	t.Run("rejects more than 10 IDs", func(t *testing.T) {
		args := make([]string, 11)
		for i := range args {
			args[i] = "571313155411053087"
		}
		err := meteringPointArgs(nil, args)
		assert.Error(t, err)
	})
}

// copenhagenForTest loads Europe/Copenhagen, the zone the CLI reads dates in.
func copenhagenForTest(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Copenhagen")
	require.NoError(t, err)
	return loc
}

// stubClock makes parseDate read instant as the current time for the rest of the test.
func stubClock(t *testing.T, instant time.Time) {
	t.Helper()
	old := clock
	clock = func() time.Time { return instant }
	t.Cleanup(func() { clock = old })
}

// zoned renders a time with its offset and zone abbreviation, so an assertion compares
// both the instant and the zone it is read in.
func zoned(tm time.Time) string {
	return tm.Format("2006-01-02T15:04:05Z07:00 MST")
}

// TestParseDate covers the dates the CLI accepts. A date is a Danish calendar day, so
// YYYY-MM-DD is midnight in Copenhagen, and now-Nd/w/m/y count back on the Copenhagen
// calendar from the current time, whatever the host's own zone. The clock is pinned to
// 00:30 on 30 March 2026 in Copenhagen: still 29 March in UTC, and the first night of
// summer time, so a day back is 23 hours.
func TestParseDate(t *testing.T) {
	cph := copenhagenForTest(t)
	stubClock(t, time.Date(2026, 3, 29, 22, 30, 0, 0, time.UTC))

	valid := []struct {
		input string
		want  time.Time
	}{
		{"2026-02-28", time.Date(2026, 2, 28, 0, 0, 0, 0, cph)},
		{"2026-07-01", time.Date(2026, 7, 1, 0, 0, 0, 0, cph)},
		{"now", time.Date(2026, 3, 30, 0, 30, 0, 0, cph)},
		{"now-1d", time.Date(2026, 3, 29, 0, 30, 0, 0, cph)},
		{"now-2w", time.Date(2026, 3, 16, 0, 30, 0, 0, cph)},
		{"now-3m", time.Date(2025, 12, 30, 0, 30, 0, 0, cph)},
		{"now-4y", time.Date(2022, 3, 30, 0, 30, 0, 0, cph)},
	}
	for _, tc := range valid {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseDate(tc.input)
			require.NoError(t, err)
			assert.Equal(t, zoned(tc.want), zoned(got))
		})
	}

	for _, input := range []string{"invalid-date", "now-5z", "2026-02-30"} {
		t.Run(input, func(t *testing.T) {
			_, err := parseDate(input)
			assert.Error(t, err)
		})
	}
}

type MockClient struct {
	eloverblik.Client
	GetMeteringPointDetailsFunc func(meteringPointIDs []string) ([]eloverblik.MeteringPointDetailsResponse, error)
	GetTimeSeriesFunc           func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error)
	ExportTimeSeriesFunc        func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) (io.ReadCloser, error)
	ExportMasterdataFunc        func(meteringPointIDs []string) (io.ReadCloser, error)

	GetChargeLinksWithChargesFunc func(meteringPointIDs []string, from, to time.Time) (*eloverblik.ChargeLinksWithChargesResponse, error)
}

type MockCustomerClient struct {
	MockClient
}

func (m *MockCustomerClient) GetCustomerCharges(meteringPointIDs []string) ([]eloverblik.CustomerChargeResponse, error) {
	return nil, nil
}
func (m *MockCustomerClient) AddRelationByID(meteringPointIDs []string) ([]eloverblik.StringResponse, error) {
	return nil, nil
}
func (m *MockCustomerClient) AddRelationByWebAccessCode(meteringPointID, webAccessCode string) (string, error) {
	return "", nil
}
func (m *MockCustomerClient) DeleteRelation(meteringPointID string) (bool, error) {
	return false, nil
}
func (m *MockCustomerClient) GetMeteringPoints(includeAll bool) ([]eloverblik.MeteringPoints, error) {
	return nil, nil
}
func (m *MockCustomerClient) ExportCharges(meteringPointIDs []string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *MockClient) GetMeteringPointDetails(meteringPointIDs []string) ([]eloverblik.MeteringPointDetailsResponse, error) {
	if m.GetMeteringPointDetailsFunc != nil {
		return m.GetMeteringPointDetailsFunc(meteringPointIDs)
	}
	return nil, nil
}

func (m *MockClient) GetTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error) {
	if m.GetTimeSeriesFunc != nil {
		return m.GetTimeSeriesFunc(meteringPointIDs, from, to, aggregation)
	}
	return nil, nil
}

func (m *MockClient) ExportTimeSeries(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) (io.ReadCloser, error) {
	if m.ExportTimeSeriesFunc != nil {
		return m.ExportTimeSeriesFunc(meteringPointIDs, from, to, aggregation)
	}
	return nil, nil
}

func (m *MockClient) ExportMasterdata(meteringPointIDs []string) (io.ReadCloser, error) {
	if m.ExportMasterdataFunc != nil {
		return m.ExportMasterdataFunc(meteringPointIDs)
	}
	return nil, nil
}

func (m *MockClient) GetChargeLinksWithCharges(meteringPointIDs []string, from, to time.Time) (*eloverblik.ChargeLinksWithChargesResponse, error) {
	if m.GetChargeLinksWithChargesFunc != nil {
		return m.GetChargeLinksWithChargesFunc(meteringPointIDs, from, to)
	}
	return &eloverblik.ChargeLinksWithChargesResponse{}, nil
}

func TestDetailsCmd(t *testing.T) {
	mock := &MockClient{
		GetMeteringPointDetailsFunc: func(meteringPointIDs []string) ([]eloverblik.MeteringPointDetailsResponse, error) {
			assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
			return []eloverblik.MeteringPointDetailsResponse{{
				Success: true,
			}}, nil
		},
	}
	clientInstance = mock
	defer func() { clientInstance = nil }()

	oldOutput := output
	var buf bytes.Buffer
	output = &buf
	defer func() { output = oldOutput }()

	_, err := execute(t, "customer", "details", "571313174002485069", "--token", "dummy")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"success":true`)
}

func TestExportTimeseriesCmd(t *testing.T) {
	// Mock the customer API for export commands
	mockCustomer := &MockCustomerClient{}
	mockCustomer.ExportTimeSeriesFunc = func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) (io.ReadCloser, error) {
		assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
		return io.NopCloser(strings.NewReader("header;value\n2026-01-01;1.23")), nil
	}
	clientInstance = mockCustomer
	defer func() { clientInstance = nil }()

	// Redirect output for export command
	oldOutput := output
	var buf bytes.Buffer
	output = &buf
	defer func() { output = oldOutput }()

	_, err := execute(t, "customer", "export-timeseries", "571313174002485069", "--from", "2026-01-01", "--token", "dummy")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "2026-01-01;1.23")
}

func TestExportMasterdataCmd(t *testing.T) {
	mockCustomer := &MockCustomerClient{}
	mockCustomer.ExportMasterdataFunc = func(meteringPointIDs []string) (io.ReadCloser, error) {
		assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
		return io.NopCloser(strings.NewReader("id;address\n571313174002485069;Some Address")), nil
	}
	clientInstance = mockCustomer
	defer func() { clientInstance = nil }()

	oldOutput := output
	var buf bytes.Buffer
	output = &buf
	defer func() { output = oldOutput }()

	_, err := execute(t, "customer", "export-masterdata", "571313174002485069", "--token", "dummy")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "571313174002485069;Some Address")
}

// TestExportCmdsFormatJSON covers --format json on the export commands. Eloverblik's CSV
// starts with a UTF-8 byte order mark, which ended up in the first key of every row, and
// a CSV with a header and no rows printed null instead of an empty array.
func TestExportCmdsFormatJSON(t *testing.T) {
	cases := []struct {
		name string
		csv  string
		want string
	}{
		{
			name: "drops the byte order mark from the first key",
			csv:  "\uFEFFMålepunktsID;Mængde\n571313174002485069;0,198",
			want: `[{"MålepunktsID":"571313174002485069","Mængde":"0,198"}]`,
		},
		{
			name: "drops the byte order mark before a quoted header",
			csv:  "\uFEFF\"MålepunktsID\";\"Mængde\"\n571313174002485069;0,198",
			want: `[{"MålepunktsID":"571313174002485069","Mængde":"0,198"}]`,
		},
		{
			name: "prints an empty array for a header without rows",
			csv:  "\uFEFFMålepunktsID;Mængde\n",
			want: `[]`,
		},
	}

	for _, command := range []string{"export-timeseries", "export-masterdata"} {
		for _, tc := range cases {
			t.Run(command+" "+tc.name, func(t *testing.T) {
				mock := &MockCustomerClient{}
				mock.ExportTimeSeriesFunc = func([]string, time.Time, time.Time, eloverblik.Aggregation) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(tc.csv)), nil
				}
				mock.ExportMasterdataFunc = func([]string) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader(tc.csv)), nil
				}
				clientInstance = mock
				defer func() { clientInstance = nil }()

				oldOutput := output
				var buf bytes.Buffer
				output = &buf
				defer func() { output = oldOutput }()

				args := []string{"customer", command, "571313174002485069", "--format", "json", "--token", "dummy"}
				if command == "export-timeseries" {
					args = append(args, "--from", "2026-09-01")
				}
				_, err := execute(t, args...)

				require.NoError(t, err)
				assert.JSONEq(t, tc.want, buf.String())
			})
		}
	}
}

func TestTimeseriesCmd(t *testing.T) {
	mock := &MockClient{
		GetTimeSeriesFunc: func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error) {
			assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
			return []eloverblik.TimeSeries{}, nil
		},
	}
	clientInstance = mock
	defer func() { clientInstance = nil }()

	oldOutput := output
	var buf bytes.Buffer
	output = &buf
	defer func() { output = oldOutput }()

	_, err := execute(t, "customer", "timeseries", "571313174002485069", "--from", "2026-01-01", "--token", "dummy")
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, buf.String())

	// Test with period
	buf.Reset()
	_, err = execute(t, "customer", "timeseries", "571313174002485069", "--period", "last_week", "--token", "dummy")
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, buf.String())

	// Test mutually exclusive flags
	buf.Reset()
	_, err = execute(t, "customer", "timeseries", "571313174002485069", "--period", "last_week", "--from", "2026-01-01", "--token", "dummy")
	assert.Error(t, err)
}

// timeSeriesResult is a successful time series result for one metering point, holding a
// single daily reading.
func timeSeriesResult(meteringPointID string, day time.Time, quantity float64) eloverblik.TimeSeries {
	return eloverblik.TimeSeries{
		MyEnergyDataMarketDocument: eloverblik.MyEnergyDataMarketDocumentResponse{
			TimeSeries: []eloverblik.TimeSeriesTimeSeriesResponse{{
				MRID:                meteringPointID,
				MeasurementUnitName: "KWH",
				Periods: []eloverblik.PeriodResponse{{
					Resolution:   "PT1D",
					TimeInterval: eloverblik.TimeInterval{Start: day, End: day.AddDate(0, 0, 1)},
					Points:       []eloverblik.PointResponse{{Position: 1, OutQuantityQuantity: quantity, OutQuantityQuality: "A04"}},
				}},
			}},
		},
		Success: true, ErrorCode: 10000, ID: meteringPointID,
	}
}

// TestTimeseriesCmdFlattenFailedMeteringPoint covers a response in which one metering point
// failed on its own. Its result carries no market document, which crashed --flatten with an
// index out of range; it must be reported as a warning instead, and the rest kept.
func TestTimeseriesCmdFlattenFailedMeteringPoint(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	mock := &MockClient{
		GetTimeSeriesFunc: func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error) {
			return []eloverblik.TimeSeries{
				timeSeriesResult("571313174002485069", day, 7.5),
				{
					Success:   false,
					ErrorCode: 30018,
					ErrorText: "MeteringPointDataNotAvailableForTheRequestedPeriod",
					ID:        "571313174002485070",
				},
			}, nil
		},
	}
	clientInstance = mock
	defer func() { clientInstance = nil }()

	oldOutput, oldWarnings := output, warningOutput
	var stdout, stderr bytes.Buffer
	output, warningOutput = &stdout, &stderr
	defer func() { output, warningOutput = oldOutput, oldWarnings }()

	_, err := execute(t, "customer", "timeseries", "571313174002485069", "571313174002485070",
		"--from", "2026-09-01", "--to", "2026-09-02", "--aggregation", "Day", "--flatten", "--token", "dummy")

	assert.NoError(t, err)

	var flattened map[string][]eloverblik.FlatTimeSeriesPoint
	assert.NoError(t, json.Unmarshal(stdout.Bytes(), &flattened))
	assert.Len(t, flattened["571313174002485069"], 1)
	assert.NotContains(t, flattened, "571313174002485070")
	assert.Contains(t, stderr.String(), "metering point 571313174002485070: 30018 MeteringPointDataNotAvailableForTheRequestedPeriod")
}

// TestTimeseriesCmdFlattenRepeatedMeteringPoint covers a response that holds the same
// metering point twice, which the API does when it reports several access periods for it.
// --flatten keys its output by metering point, so the periods must add up, not overwrite.
func TestTimeseriesCmdFlattenRepeatedMeteringPoint(t *testing.T) {
	first := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	second := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	mock := &MockClient{
		GetTimeSeriesFunc: func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error) {
			return []eloverblik.TimeSeries{
				timeSeriesResult("571313174002485069", first, 7.5),
				timeSeriesResult("571313174002485069", second, 8.5),
			}, nil
		},
	}
	clientInstance = mock
	defer func() { clientInstance = nil }()

	oldOutput := output
	var stdout bytes.Buffer
	output = &stdout
	defer func() { output = oldOutput }()

	_, err := execute(t, "customer", "timeseries", "571313174002485069",
		"--from", "2026-09-01", "--to", "2026-09-03", "--aggregation", "Day", "--flatten", "--token", "dummy")

	assert.NoError(t, err)

	var flattened map[string][]eloverblik.FlatTimeSeriesPoint
	assert.NoError(t, json.Unmarshal(stdout.Bytes(), &flattened))
	if assert.Len(t, flattened["571313174002485069"], 2) {
		assert.InDelta(t, 7.5, flattened["571313174002485069"][0].Measurement, 1e-9)
		assert.InDelta(t, 8.5, flattened["571313174002485069"][1].Measurement, 1e-9)
	}
}

// TestTimeseriesCmdsSendCalendarDates proves that reading a date as midnight in Copenhagen
// rather than in UTC changes nothing for timeseries and export-timeseries. The client sends
// them the Copenhagen calendar date of from and to, and midnight in UTC falls on the same
// Copenhagen date as midnight in Copenhagen, in winter and in summer time alike.
func TestTimeseriesCmdsSendCalendarDates(t *testing.T) {
	cph := copenhagenForTest(t)
	// sentDate is the date the client puts in the URL: GetTimeSeries and ExportTimeSeries
	// send the Copenhagen calendar date and drop the time of day.
	sentDate := func(tm time.Time) string { return tm.In(cph).Format(time.DateOnly) }

	for _, command := range []string{"timeseries", "export-timeseries"} {
		t.Run(command, func(t *testing.T) {
			var from, to time.Time
			mock := &MockCustomerClient{}
			mock.GetTimeSeriesFunc = func(_ []string, f, tt time.Time, _ eloverblik.Aggregation) ([]eloverblik.TimeSeries, error) {
				from, to = f, tt
				return []eloverblik.TimeSeries{}, nil
			}
			mock.ExportTimeSeriesFunc = func(_ []string, f, tt time.Time, _ eloverblik.Aggregation) (io.ReadCloser, error) {
				from, to = f, tt
				return io.NopCloser(strings.NewReader("")), nil
			}
			clientInstance = mock
			defer func() { clientInstance = nil }()

			oldOutput := output
			output = io.Discard
			defer func() { output = oldOutput }()

			_, err := execute(t, "customer", command, "571313174002485069",
				"--from", "2026-01-01", "--to", "2026-07-01", "--token", "dummy")

			require.NoError(t, err)
			assert.Equal(t, "2026-01-01", sentDate(from))
			assert.Equal(t, "2026-07-01", sentDate(to))
		})
	}
}
