package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

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
