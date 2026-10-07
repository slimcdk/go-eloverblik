package eloverblik

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// periodCase is one expected range. The bounds are RFC 3339 timestamps, which carry the
// UTC offset, so a bound at the right wall clock in the wrong zone fails as well.
type periodCase struct {
	name   string
	now    time.Time
	period Period
	from   string
	to     string
}

func (c periodCase) run(t *testing.T) {
	t.Helper()
	t.Run(c.name, func(t *testing.T) {
		from, to, err := getDatesFromPeriod(c.period, c.now)
		require.NoError(t, err)
		assert.Equal(t, c.from, from.Format(time.RFC3339), "from")
		assert.Equal(t, c.to, to.Format(time.RFC3339), "to")
	})
}

func TestGetDatesFromPeriod(t *testing.T) {
	// Wednesday 18 March 2026, in winter time (CET, +01:00).
	now := time.Date(2026, 3, 18, 14, 30, 0, 0, cph)

	for _, c := range []periodCase{
		{period: Yesterday, from: "2026-03-17T00:00:00+01:00", to: "2026-03-18T00:00:00+01:00"},
		{period: ThisWeek, from: "2026-03-16T00:00:00+01:00", to: "2026-03-18T14:30:00+01:00"},
		{period: LastWeek, from: "2026-03-09T00:00:00+01:00", to: "2026-03-16T00:00:00+01:00"},
		{period: ThisMonth, from: "2026-03-01T00:00:00+01:00", to: "2026-03-18T14:30:00+01:00"},
		{period: LastMonth, from: "2026-02-01T00:00:00+01:00", to: "2026-03-01T00:00:00+01:00"},
		{period: ThisYear, from: "2026-01-01T00:00:00+01:00", to: "2026-03-18T14:30:00+01:00"},
		{period: LastYear, from: "2025-01-01T00:00:00+01:00", to: "2026-01-01T00:00:00+01:00"},
		{period: "This_Week", from: "2026-03-16T00:00:00+01:00", to: "2026-03-18T14:30:00+01:00"},
	} {
		c.name, c.now = string(c.period), now
		c.run(t)
	}

	t.Run("invalid", func(t *testing.T) {
		from, to, err := getDatesFromPeriod("invalid", now)
		require.Error(t, err)
		assert.True(t, from.IsZero())
		assert.True(t, to.IsZero())
	})
}

// TestGetDatesFromPeriodWeeksStartOnMonday pins the week to ISO 8601 and the Danish
// calendar: Monday to Sunday, so a Sunday is the last day of its week, not the first.
func TestGetDatesFromPeriodWeeksStartOnMonday(t *testing.T) {
	sunday := time.Date(2026, 3, 22, 20, 0, 0, 0, cph)
	saturday := time.Date(2026, 3, 21, 9, 0, 0, 0, cph)
	monday := time.Date(2026, 3, 16, 9, 0, 0, 0, cph)
	newYear := time.Date(2026, 1, 1, 12, 0, 0, 0, cph) // a Thursday

	for _, c := range []periodCase{
		{name: "this_week on a Sunday", now: sunday, period: ThisWeek,
			from: "2026-03-16T00:00:00+01:00", to: "2026-03-22T20:00:00+01:00"},
		{name: "last_week on a Sunday", now: sunday, period: LastWeek,
			from: "2026-03-09T00:00:00+01:00", to: "2026-03-16T00:00:00+01:00"},
		{name: "this_week on a Saturday", now: saturday, period: ThisWeek,
			from: "2026-03-16T00:00:00+01:00", to: "2026-03-21T09:00:00+01:00"},
		{name: "last_week on a Monday", now: monday, period: LastWeek,
			from: "2026-03-09T00:00:00+01:00", to: "2026-03-16T00:00:00+01:00"},
		{name: "this_week across new year", now: newYear, period: ThisWeek,
			from: "2025-12-29T00:00:00+01:00", to: "2026-01-01T12:00:00+01:00"},
		{name: "last_week across new year", now: newYear, period: LastWeek,
			from: "2025-12-22T00:00:00+01:00", to: "2025-12-29T00:00:00+01:00"},
	} {
		c.run(t)
	}
}

// TestGetDatesFromPeriodIsHalfOpen guards every period against the two ways the API
// rejects or truncates a range. The API reads the range as [dateFrom, dateTo) on a
// date granularity: an equal pair is rejected with error 30002, and a to that lands
// inside the period silently drops the period's last day.
func TestGetDatesFromPeriodIsHalfOpen(t *testing.T) {
	// A Wednesday, so that no this_* period starts today and the week periods do not
	// straddle a month boundary.
	now := time.Date(2026, 3, 18, 14, 30, 0, 0, cph)

	periods := []Period{Yesterday, ThisWeek, LastWeek, ThisMonth, LastMonth, ThisYear, LastYear}

	// The last day each period must still cover, i.e. the day before the exclusive to.
	lastDay := map[Period]string{
		Yesterday: "2026-03-17",
		LastWeek:  "2026-03-15", // the Sunday that ends last week
		LastMonth: "2026-02-28",
		LastYear:  "2025-12-31",
	}

	for _, period := range periods {
		t.Run(string(period), func(t *testing.T) {
			from, to, err := getDatesFromPeriod(period, now)
			require.NoError(t, err)

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
