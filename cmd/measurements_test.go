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
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func TestCsvToJSON(t *testing.T) {
	t.Run("converts simple CSV to JSON", func(t *testing.T) {
		// CSV with BOM (U+FEFF) at the start
		csvData := "\uFEFFMålepunktsID;Fra_dato;Til_dato;Mængde\n571313155411053087;01-02-2026 00:00:00;01-02-2026 01:00:00;0,198\n571313155411053087;01-02-2026 01:00:00;01-02-2026 02:00:00;0,196"
		stream := nopCloser{strings.NewReader(csvData)}

		// Capture stdout
		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := csvToJSON(stream)
		assert.NoError(t, err)

		// Parse the JSON output
		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		assert.NoError(t, err)
		assert.Len(t, records, 2)

		// Check first record (BOM will be in the first header)
		assert.Equal(t, "571313155411053087", records[0]["\uFEFFMålepunktsID"])
		assert.Equal(t, "01-02-2026 00:00:00", records[0]["Fra_dato"])
		assert.Equal(t, "01-02-2026 01:00:00", records[0]["Til_dato"])
		assert.Equal(t, "0,198", records[0]["Mængde"])

		// Check second record
		assert.Equal(t, "571313155411053087", records[1]["\uFEFFMålepunktsID"])
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
		assert.NoError(t, err)

		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		assert.NoError(t, err)
		assert.Len(t, records, 0)
	})

	t.Run("handles CSV with special characters", func(t *testing.T) {
		csvData := "Navn;Beskrivelse\nNet abo C;Net abo C forbrug flex - stikledning\nTSO - System;Abonnement TSO"
		stream := nopCloser{strings.NewReader(csvData)}

		var buf bytes.Buffer
		oldStdout := output
		output = &buf
		defer func() { output = oldStdout }()

		err := csvToJSON(stream)
		assert.NoError(t, err)

		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		assert.NoError(t, err)
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
		assert.NoError(t, err)

		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		assert.NoError(t, err)
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
		assert.NoError(t, err)

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
		assert.NoError(t, err)

		// Should output JSON
		var records []map[string]string
		err = json.Unmarshal(buf.Bytes(), &records)
		assert.NoError(t, err)
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
		assert.NoError(t, err)

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
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id")
	})

	t.Run("rejects non-numeric IDs", func(t *testing.T) {
		args := []string{"57131315541105308a"}
		err := meteringPointArgs(nil, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id")
	})

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

func TestParseDate(t *testing.T) {
	testCases := []struct {
		input     string
		expectErr bool
		checkFunc func(t *testing.T, got time.Time)
	}{
		{
			input: "2026-02-28",
			checkFunc: func(t *testing.T, got time.Time) {
				expected := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
				assert.Equal(t, expected, got)
			},
		},
		{
			input: "now",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now(), got, time.Second)
			},
		},
		{
			input: "now-1d",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now().AddDate(0, 0, -1), got, time.Second)
			},
		},
		{
			input: "now-2w",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now().AddDate(0, 0, -14), got, time.Second)
			},
		},
		{
			input: "now-3m",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now().AddDate(0, -3, 0), got, time.Second)
			},
		},
		{
			input: "now-4y",
			checkFunc: func(t *testing.T, got time.Time) {
				assert.WithinDuration(t, time.Now().AddDate(-4, 0, 0), got, time.Second)
			},
		},
		{
			input:     "invalid-date",
			expectErr: true,
		},
		{
			input:     "now-5z",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseDate(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tc.checkFunc(t, got)
			}
		})
	}
}

type MockClient struct {
	eloverblik.Client
	GetMeteringPointDetailsFunc func(meteringPointIDs []string) ([]eloverblik.MeteringPointDetail, error)
	GetTimeSeriesFunc           func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error)
	ExportTimeSeriesFunc        func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) (io.ReadCloser, error)
	ExportMasterdataFunc        func(meteringPointIDs []string) (io.ReadCloser, error)
}

func (m *MockClient) GetMeteringPointDetails(meteringPointIDs []string) ([]eloverblik.MeteringPointDetail, error) {
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

func TestDetailsCmd(t *testing.T) {
	mock := &MockClient{
		GetMeteringPointDetailsFunc: func(meteringPointIDs []string) ([]eloverblik.MeteringPointDetail, error) {
			assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
			return []eloverblik.MeteringPointDetail{{Success: true}}, nil
		},
	}
	clientInstance = mock

	out, err := execute(t, "details", "571313174002485069")
	assert.NoError(t, err)
	assert.Contains(t, out, `"success": true`)
}

func TestTimeseriesCmd(t *testing.T) {
	mock := &MockClient{
		GetTimeSeriesFunc: func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) ([]eloverblik.TimeSeries, error) {
			assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
			return []eloverblik.TimeSeries{}, nil
		},
	}
	clientInstance = mock

	_, err := execute(t, "timeseries", "571313174002485069", "--from", "2026-01-01")
	assert.NoError(t, err)

	// Test with period
	_, err = execute(t, "timeseries", "571313174002485069", "--period", "last_week")
	assert.NoError(t, err)

	// Test mutually exclusive flags
	_, err = execute(t, "timeseries", "571313174002485069", "--period", "last_week", "--from", "2026-01-01")
	assert.Error(t, err)
}

func TestExportTimeseriesCmd(t *testing.T) {
	// Mock the customer API for export commands
	mockCustomer := &MockClient{
		ExportTimeSeriesFunc: func(meteringPointIDs []string, from, to time.Time, aggregation eloverblik.Aggregation) (io.ReadCloser, error) {
			assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
			return io.NopCloser(strings.NewReader("header;value\n2026-01-01;1.23")), nil
		},
	}
	clientInstance = mockCustomer

	// Redirect output for export command
	oldOutput := output
	var buf bytes.Buffer
	output = &buf
	defer func() { output = oldOutput }()

	_, err := execute(t, "export-timeseries", "571313174002485069", "--from", "2026-01-01")
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "2026-01-01;1.23")
}

func TestExportMasterdataCmd(t *testing.T) {
	mockCustomer := &MockClient{
		ExportMasterdataFunc: func(meteringPointIDs []string) (io.ReadCloser, error) {
			assert.Equal(t, []string{"571313174002485069"}, meteringPointIDs)
			return io.NopCloser(strings.NewReader("id;address\n571313174002485069;Some Address")), nil
		},
	}
	clientInstance = mockCustomer

	oldOutput := output
	var buf bytes.Buffer
	output = &buf
	defer func() { output = oldOutput }()

	_, err := execute(t, "export-masterdata", "571313174002485069")
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "571313174002485069;Some Address")
}
