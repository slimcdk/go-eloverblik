package eloverblik

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDatesFromPeriod(t *testing.T) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstOfThisYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	startOfThisWeek := startOfToday.AddDate(0, 0, -int(now.Weekday()))

	tests := []struct {
		period    Period
		expectErr bool
		from      time.Time
		to        time.Time
	}{
		{
			period: Yesterday,
			from:   startOfToday.AddDate(0, 0, -1),
			to:     startOfToday,
		},
		{
			period: ThisWeek,
			from:   startOfThisWeek,
			to:     now,
		},
		{
			period: LastWeek,
			from:   startOfThisWeek.AddDate(0, 0, -7),
			to:     startOfThisWeek,
		},
		{
			period: ThisMonth,
			from:   firstOfThisMonth,
			to:     now,
		},
		{
			period: LastMonth,
			from:   firstOfThisMonth.AddDate(0, -1, 0),
			to:     firstOfThisMonth,
		},
		{
			period: ThisYear,
			from:   firstOfThisYear,
			to:     now,
		},
		{
			period: LastYear,
			from:   firstOfThisYear.AddDate(-1, 0, 0),
			to:     firstOfThisYear,
		},
		{
			period:    "invalid",
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(string(test.period), func(t *testing.T) {
			from, to, err := getDatesFromPeriod(test.period, now)
			if test.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.WithinDuration(t, test.from, from, time.Second)
			assert.WithinDuration(t, test.to, to, time.Second)
		})
	}
}

// TestGetDatesFromPeriodIsHalfOpen guards every period against the two ways the API
// rejects or truncates a range. The API reads the range as [dateFrom, dateTo) on a
// date granularity: an equal pair is rejected with error 30002, and a to that lands
// inside the period silently drops the period's last day.
func TestGetDatesFromPeriodIsHalfOpen(t *testing.T) {
	// A Monday, so that the week periods do not straddle a month boundary.
	now := time.Date(2026, 3, 16, 14, 30, 0, 0, cph)

	periods := []Period{Yesterday, ThisWeek, LastWeek, ThisMonth, LastMonth, ThisYear, LastYear}

	// The last day each period must still cover, i.e. the day before the exclusive to.
	lastDay := map[Period]string{
		Yesterday: "2026-03-15",
		LastWeek:  "2026-03-14", // the Saturday before this week
		LastMonth: "2026-02-28",
		LastYear:  "2025-12-31",
	}

	for _, period := range periods {
		t.Run(string(period), func(t *testing.T) {
			from, to, err := getDatesFromPeriod(period, now)
			assert.NoError(t, err)

			// The API formats both bounds as YYYY-MM-DD, so they must differ as dates.
			fromDate := from.Format(time.DateOnly)
			toDate := to.Format(time.DateOnly)
			assert.NotEqual(t, fromDate, toDate, "equal dates are rejected with error 30002")
			assert.True(t, to.After(from), "to must be after from")

			// The exclusive bound must sit on the day after the last day of the period,
			// otherwise the API drops that last day.
			if want, ok := lastDay[period]; ok {
				assert.Equal(t, want, to.AddDate(0, 0, -1).Format(time.DateOnly),
					"the last day of the period must still be inside the requested range")
			}
		})
	}
}
