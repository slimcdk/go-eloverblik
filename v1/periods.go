package eloverblik

import (
	"fmt"
	"strings"
	"time"
)

// Period defines a custom string type for predefined time periods.
type Period string

// Defines constants for all supported time periods.
const (
	Yesterday Period = "yesterday"
	ThisWeek  Period = "this_week"
	LastWeek  Period = "last_week"
	ThisMonth Period = "this_month"
	LastMonth Period = "last_month"
	ThisYear  Period = "this_year"
	LastYear  Period = "last_year"
)

// GetDatesFromPeriod calculates the from and to time.Time values based on a Period.
// This is useful for easily specifying common time ranges when calling API methods.
//
// The API treats the requested range as half-open: it returns data from dateFrom up to
// but not including dateTo, and it rejects a request where the two dates are equal with
// error 30002. The returned to is therefore the start of the period that follows, not
// the last instant of the period itself.
func GetDatesFromPeriod(period Period) (from time.Time, to time.Time, err error) {
	return getDatesFromPeriod(period, time.Now())
}

// getDatesFromPeriod is the internal, testable implementation for calculating dates.
func getDatesFromPeriod(period Period, now time.Time) (from time.Time, to time.Time, err error) {
	year, month, day := now.Date()
	startOfToday := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	firstOfThisMonth := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())

	switch strings.ToLower(string(period)) {
	case string(Yesterday):
		from = startOfToday.AddDate(0, 0, -1)
		to = startOfToday // Exclusive: the day that follows yesterday
	case string(ThisWeek):
		weekday := int(now.Weekday())
		from = startOfToday.AddDate(0, 0, -weekday)
		to = now
	case string(LastWeek):
		weekday := int(now.Weekday())
		startOfThisWeek := startOfToday.AddDate(0, 0, -weekday)
		from = startOfThisWeek.AddDate(0, 0, -7)
		to = startOfThisWeek
	case string(ThisMonth):
		from = firstOfThisMonth
		to = now
	case string(LastMonth):
		from = firstOfThisMonth.AddDate(0, -1, 0)
		to = firstOfThisMonth // Exclusive: the first of this month
	case string(ThisYear):
		from = time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
		to = now
	case string(LastYear):
		firstOfThisYear := time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
		from = firstOfThisYear.AddDate(-1, 0, 0)
		to = firstOfThisYear
	default:
		err = fmt.Errorf("invalid period: '%s'", period)
	}
	return
}
