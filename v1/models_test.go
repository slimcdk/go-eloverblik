package eloverblik

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFlexibleTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectZero  bool
	}{
		{
			name:       "valid RFC3339 timestamp",
			input:      `"2024-01-15T10:30:00Z"`,
			expectZero: false,
		},
		{
			name:       "empty string",
			input:      `""`,
			expectZero: true,
		},
		{
			name:       "null value",
			input:      `null`,
			expectZero: true,
		},
		{
			name:        "invalid timestamp format",
			input:       `"not-a-date"`,
			expectError: true,
		},
		{
			name:       "RFC3339 with timezone",
			input:      `"2024-01-15T10:30:00+01:00"`,
			expectZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FlexibleTime
			err := json.Unmarshal([]byte(tt.input), &ft)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectZero {
					assert.True(t, ft.Time.IsZero())
				} else {
					assert.False(t, ft.Time.IsZero())
				}
			}
		})
	}
}

func TestFlexibleTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    FlexibleTime
		expected string
	}{
		{
			name:     "zero time",
			input:    FlexibleTime{Time: time.Time{}},
			expected: `null`,
		},
		{
			name:     "valid time",
			input:    FlexibleTime{Time: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
			expected: `"2024-01-15T10:30:00Z"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestFlexibleTime_InStruct(t *testing.T) {
	type TestStruct struct {
		Date FlexibleTime `json:"date"`
		Name string       `json:"name"`
	}

	t.Run("unmarshal struct with empty date string", func(t *testing.T) {
		jsonData := `{"date": "", "name": "test"}`
		var ts TestStruct
		err := json.Unmarshal([]byte(jsonData), &ts)
		assert.NoError(t, err)
		assert.True(t, ts.Date.Time.IsZero())
		assert.Equal(t, "test", ts.Name)
	})

	t.Run("unmarshal struct with valid date", func(t *testing.T) {
		jsonData := `{"date": "2024-01-15T10:30:00Z", "name": "test"}`
		var ts TestStruct
		err := json.Unmarshal([]byte(jsonData), &ts)
		assert.NoError(t, err)
		assert.False(t, ts.Date.Time.IsZero())
		assert.Equal(t, 2024, ts.Date.Year())
		assert.Equal(t, "test", ts.Name)
	})

	t.Run("marshal struct with zero date", func(t *testing.T) {
		ts := TestStruct{
			Date: FlexibleTime{Time: time.Time{}},
			Name: "test",
		}
		result, err := json.Marshal(ts)
		assert.NoError(t, err)
		assert.Contains(t, string(result), `"date":null`)
		assert.Contains(t, string(result), `"name":"test"`)
	})
}
