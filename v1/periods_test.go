package eloverblik

import (
	"testing"
	"time"
)

func TestGetDatesFromPeriod(t *testing.T) {
	// A fixed "now" for predictable test results: Sunday, March 15, 2026, 10:30:00
	mockNow := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	testCases := []struct {
		period       Period
		expectedFrom time.Time
		expectedTo   time.Time
		expectErr    bool
	}{
		{
			period:       Yesterday,
			expectedFrom: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
			expectedTo:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
		{
			period:       ThisWeek,
			expectedFrom: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), // It's Sunday, so week starts today
			expectedTo:   mockNow,
		},
		{
			period:       LastWeek,
			expectedFrom: time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC),                        // Previous Sunday
			expectedTo:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond), // End of Saturday
		},
		{
			period:       ThisMonth,
			expectedFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedTo:   mockNow,
		},
		{
			period:       LastMonth,
			expectedFrom: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			expectedTo:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
		{
			period:       ThisYear,
			expectedFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedTo:   mockNow,
		},
		{
			period:       LastYear,
			expectedFrom: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedTo:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
		{
			period:    "invalid_period",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.period), func(t *testing.T) {
			from, to, err := getDatesFromPeriod(tc.period, mockNow)

			if (err != nil) != tc.expectErr {
				t.Fatalf("expected error: %v, got: %v", tc.expectErr, err)
			}

			if !tc.expectErr {
				if !from.Equal(tc.expectedFrom) {
					t.Errorf("From date mismatch: got %v, want %v", from, tc.expectedFrom)
				}
				if !to.Equal(tc.expectedTo) {
					t.Errorf("To date mismatch: got %v, want %v", to, tc.expectedTo)
				}
			}
		})
	}
}
