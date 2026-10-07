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
		assert.Equal(t, "Europe/Copenhagen", from.Location().String(), "from")
		assert.Equal(t, "Europe/Copenhagen", to.Location().String(), "to")
	})
}

func loadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
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

// TestGetDatesFromPeriodUsesCopenhagenTime gives now in zones east and west of
// Copenhagen, mostly at moments when their date differs from the Copenhagen date. The
// periods must follow the Copenhagen date, and the bounds must be Copenhagen midnights.
func TestGetDatesFromPeriodUsesCopenhagenTime(t *testing.T) {
	tokyo := loadLocation(t, "Asia/Tokyo")
	losAngeles := loadLocation(t, "America/Los_Angeles")

	// Wednesday 20:00 in Tokyo is Wednesday 12:00 in Copenhagen: the same date, but
	// Tokyo's midnights are 16:00 the day before in Copenhagen.
	tokyoEvening := time.Date(2026, 3, 18, 20, 0, 0, 0, tokyo)
	// Monday 07:00 in Tokyo is Sunday 23:00 in Copenhagen: still last week's Sunday.
	tokyoMonday := time.Date(2026, 3, 23, 7, 0, 0, 0, tokyo)
	// 1 April 06:00 in Tokyo is 31 March 23:00 in Copenhagen: still March.
	tokyoFirstOfApril := time.Date(2026, 4, 1, 6, 0, 0, 0, tokyo)
	// Sunday 17:00 in Los Angeles is Monday 01:00 in Copenhagen: a new week.
	losAngelesSunday := time.Date(2026, 3, 22, 17, 0, 0, 0, losAngeles)
	// New Year's Eve 16:30 in Los Angeles is 1 January 01:30 in Copenhagen: a new year.
	losAngelesNewYearsEve := time.Date(2025, 12, 31, 16, 30, 0, 0, losAngeles)

	for _, c := range []periodCase{
		{name: "Tokyo evening yesterday", now: tokyoEvening, period: Yesterday,
			from: "2026-03-17T00:00:00+01:00", to: "2026-03-18T00:00:00+01:00"},
		{name: "Tokyo evening last_year", now: tokyoEvening, period: LastYear,
			from: "2025-01-01T00:00:00+01:00", to: "2026-01-01T00:00:00+01:00"},
		{name: "Tokyo Monday yesterday", now: tokyoMonday, period: Yesterday,
			from: "2026-03-21T00:00:00+01:00", to: "2026-03-22T00:00:00+01:00"},
		{name: "Tokyo Monday this_week", now: tokyoMonday, period: ThisWeek,
			from: "2026-03-16T00:00:00+01:00", to: "2026-03-22T23:00:00+01:00"},
		{name: "Tokyo Monday last_week", now: tokyoMonday, period: LastWeek,
			from: "2026-03-09T00:00:00+01:00", to: "2026-03-16T00:00:00+01:00"},
		{name: "Tokyo 1 April this_month", now: tokyoFirstOfApril, period: ThisMonth,
			from: "2026-03-01T00:00:00+01:00", to: "2026-03-31T23:00:00+02:00"},
		{name: "Tokyo 1 April last_month", now: tokyoFirstOfApril, period: LastMonth,
			from: "2026-02-01T00:00:00+01:00", to: "2026-03-01T00:00:00+01:00"},
		{name: "Tokyo 1 April this_year", now: tokyoFirstOfApril, period: ThisYear,
			from: "2026-01-01T00:00:00+01:00", to: "2026-03-31T23:00:00+02:00"},
		{name: "Los Angeles Sunday yesterday", now: losAngelesSunday, period: Yesterday,
			from: "2026-03-22T00:00:00+01:00", to: "2026-03-23T00:00:00+01:00"},
		{name: "Los Angeles Sunday last_week", now: losAngelesSunday, period: LastWeek,
			from: "2026-03-16T00:00:00+01:00", to: "2026-03-23T00:00:00+01:00"},
		{name: "Los Angeles New Year's Eve yesterday", now: losAngelesNewYearsEve, period: Yesterday,
			from: "2025-12-31T00:00:00+01:00", to: "2026-01-01T00:00:00+01:00"},
		{name: "Los Angeles New Year's Eve this_week", now: losAngelesNewYearsEve, period: ThisWeek,
			from: "2025-12-29T00:00:00+01:00", to: "2026-01-01T01:30:00+01:00"},
		{name: "Los Angeles New Year's Eve last_month", now: losAngelesNewYearsEve, period: LastMonth,
			from: "2025-12-01T00:00:00+01:00", to: "2026-01-01T00:00:00+01:00"},
		{name: "Los Angeles New Year's Eve last_year", now: losAngelesNewYearsEve, period: LastYear,
			from: "2025-01-01T00:00:00+01:00", to: "2026-01-01T00:00:00+01:00"},
	} {
		c.run(t)
	}
}

// TestGetDatesFromPeriodAcrossDaylightSavingTime covers the days Copenhagen changes
// between CET (+01:00) and CEST (+02:00): in 2026 the clocks go forward at 02:00 on
// 29 March, a 23-hour day, and back at 03:00 on 25 October, a 25-hour day. Every bound
// is still midnight, with the offset in force on its own date. now is given in UTC so
// the conversion to Copenhagen time is exercised too.
func TestGetDatesFromPeriodAcrossDaylightSavingTime(t *testing.T) {
	// Monday 30 March 12:00 CEST, the day after the clocks went forward.
	afterSpringForward := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)
	// Sunday 29 March 03:30 CEST, an hour after the clocks went forward.
	springForward := time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC)
	// Monday 26 October 10:00 CET, the day after the clocks went back.
	afterFallBack := time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)
	// Sunday 25 October 02:30 occurs twice: first in CEST, then again in CET.
	fallBackFirstPass := time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC)
	fallBackSecondPass := time.Date(2026, 10, 25, 1, 30, 0, 0, time.UTC)
	midApril := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	midNovember := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

	for _, c := range []periodCase{
		{name: "yesterday is the 23-hour day", now: afterSpringForward, period: Yesterday,
			from: "2026-03-29T00:00:00+01:00", to: "2026-03-30T00:00:00+02:00"},
		{name: "last_week ends after spring forward", now: afterSpringForward, period: LastWeek,
			from: "2026-03-23T00:00:00+01:00", to: "2026-03-30T00:00:00+02:00"},
		{name: "this_month spans spring forward", now: afterSpringForward, period: ThisMonth,
			from: "2026-03-01T00:00:00+01:00", to: "2026-03-30T12:00:00+02:00"},
		{name: "this_week on the spring forward day", now: springForward, period: ThisWeek,
			from: "2026-03-23T00:00:00+01:00", to: "2026-03-29T03:30:00+02:00"},
		{name: "yesterday on the spring forward day", now: springForward, period: Yesterday,
			from: "2026-03-28T00:00:00+01:00", to: "2026-03-29T00:00:00+01:00"},
		{name: "yesterday is the 25-hour day", now: afterFallBack, period: Yesterday,
			from: "2026-10-25T00:00:00+02:00", to: "2026-10-26T00:00:00+01:00"},
		{name: "last_week ends after fall back", now: afterFallBack, period: LastWeek,
			from: "2026-10-19T00:00:00+02:00", to: "2026-10-26T00:00:00+01:00"},
		{name: "this_week at 02:30 CEST on the fall back day", now: fallBackFirstPass, period: ThisWeek,
			from: "2026-10-19T00:00:00+02:00", to: "2026-10-25T02:30:00+02:00"},
		{name: "this_week at 02:30 CET on the fall back day", now: fallBackSecondPass, period: ThisWeek,
			from: "2026-10-19T00:00:00+02:00", to: "2026-10-25T02:30:00+01:00"},
		{name: "last_month ends in summer time", now: midApril, period: LastMonth,
			from: "2026-03-01T00:00:00+01:00", to: "2026-04-01T00:00:00+02:00"},
		{name: "last_month ends in winter time", now: midNovember, period: LastMonth,
			from: "2026-10-01T00:00:00+02:00", to: "2026-11-01T00:00:00+01:00"},
	} {
		c.run(t)
	}
}

// TestGetDatesFromPeriodReturnsCopenhagenTime goes through the exported function, which
// reads the clock in the host's zone (time.Local). Whatever that zone is, the bounds of
// the complete periods are midnights in Copenhagen. Run it with TZ=Asia/Tokyo or
// TZ=America/Los_Angeles to try a host east or west of Copenhagen.
func TestGetDatesFromPeriodReturnsCopenhagenTime(t *testing.T) {
	for _, period := range []Period{Yesterday, LastWeek, LastMonth, LastYear} {
		t.Run(string(period), func(t *testing.T) {
			from, to, err := GetDatesFromPeriod(period)
			require.NoError(t, err)
			for _, bound := range []time.Time{from, to} {
				assert.Equal(t, "Europe/Copenhagen", bound.Location().String())
				assert.Equal(t, "00:00:00", bound.Format(time.TimeOnly))
			}
		})
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
