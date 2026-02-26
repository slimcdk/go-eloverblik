package eloverblik

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDatesFromPeriod(t *testing.T) {
	now := time.Now()
	tests := []struct {
		period    Period
		expectErr bool
		from      time.Time
		to        time.Time
	}{
		{
			period:    Yesterday,
			expectErr: false,
			from:      time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1),
			to:        time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(-1 * time.Nanosecond),
		},
		{
			period:    ThisWeek,
			expectErr: false,
			from:      time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -int(now.Weekday())),
			to:        now,
		},
		{
			period:    LastWeek,
			expectErr: false,
			from:      time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -int(now.Weekday())-7),
			to:        time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -int(now.Weekday())).Add(-1 * time.Nanosecond),
		},
		{
			period:    ThisMonth,
			expectErr: false,
			from:      time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
			to:        now,
		},
		{
			period:    LastMonth,
			expectErr: false,
			from:      time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0),
			to:        time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Add(-1 * time.Nanosecond),
		},
		{
			period:    ThisYear,
			expectErr: false,
			from:      time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()),
			to:        now,
		},
		{
			period:    LastYear,
			expectErr: false,
			from:      time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).AddDate(-1, 0, 0),
			to:        time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Add(-1 * time.Nanosecond),
		},
		{
			period:    "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.period), func(t *testing.T) {
			from, to, err := getDatesFromPeriod(tt.period, now)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.WithinDuration(t, tt.from, from, time.Second)
				assert.WithinDuration(t, tt.to, to, time.Second)
			}
		})
	}
}
