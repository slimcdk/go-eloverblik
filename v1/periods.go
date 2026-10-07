package eloverblik

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrorPeriodHasNoCompleteDay is returned by GetDatesFromPeriod, wrapped with the
// period's name, for this_week, this_month and this_year on their first day in
// Copenhagen: a Monday, the 1st of the month and 1 January respectively. The period
// then runs from 00:00 today to now, so from and to fall on the same date, which the
// API rejects with error 30002 (ErrorToDateCanNotBeEqualToFromDate). Match it with
// errors.Is.
var ErrorPeriodHasNoCompleteDay = errors.New("period started today and has no complete day yet")

// Period names a predefined date range relative to today, which GetDatesFromPeriod
// turns into from and to bounds.
//
// Periods are computed in Copenhagen time (Europe/Copenhagen), the time the API's dates
// are in, whatever the host's time zone: today is the current date in Copenhagen, and
// every bound is 00:00 Copenhagen time, except the to of a this_* period, which is now.
// Weeks run from Monday to Sunday, as in ISO 8601 and the Danish calendar. Every range
// is half-open, [from, to), as the API reads it.
type Period string

// The supported periods, each with the [from, to) range GetDatesFromPeriod returns.
// The API reads both bounds as dates, so a this_* period covers its first day up to and
// including yesterday, and leaves out today, which is not complete.
const (
	// Yesterday is [00:00 yesterday, 00:00 today).
	Yesterday Period = "yesterday"
	// ThisWeek is [00:00 on this week's Monday, now). On a Monday it is an error that
	// wraps ErrorPeriodHasNoCompleteDay.
	ThisWeek Period = "this_week"
	// LastWeek is [00:00 on last week's Monday, 00:00 on this week's Monday): Monday to
	// Sunday.
	LastWeek Period = "last_week"
	// ThisMonth is [00:00 on the 1st of this month, now). On the 1st it is an error that
	// wraps ErrorPeriodHasNoCompleteDay.
	ThisMonth Period = "this_month"
	// LastMonth is [00:00 on the 1st of last month, 00:00 on the 1st of this month).
	LastMonth Period = "last_month"
	// ThisYear is [00:00 on 1 January this year, now). On 1 January it is an error that
	// wraps ErrorPeriodHasNoCompleteDay.
	ThisYear Period = "this_year"
	// LastYear is [00:00 on 1 January last year, 00:00 on 1 January this year).
	LastYear Period = "last_year"
)

// GetDatesFromPeriod returns the from and to bounds of a Period, for the API methods
// that take a date range, such as GetTimeSeries. The period name is matched without
// regard to case, and the range of each period is documented on its constant.
//
// It works in Copenhagen time (Europe/Copenhagen) whatever the host's time zone: today
// is the current date in Copenhagen, weeks start on Monday, and from and to are returned
// in Europe/Copenhagen.
//
// The API treats the requested range as half-open: it returns data from dateFrom up to
// but not including dateTo, and it rejects a request where the two dates are equal with
// error 30002. For yesterday, last_week, last_month and last_year the returned to is
// therefore 00:00 on the first day after the period, not the last instant of the period
// itself. For this_week, this_month and this_year it is now, so the range ends before
// today. On the first day of a this_* period from and to would fall on the same date,
// so it returns an error wrapping ErrorPeriodHasNoCompleteDay instead, such as
// "this_week: period started today and has no complete day yet".
//
// An unknown period name is an error too. On an error, from and to are zero.
func GetDatesFromPeriod(period Period) (from time.Time, to time.Time, err error) {
	return getDatesFromPeriod(period, time.Now())
}

// getDatesFromPeriod is the internal, testable implementation for calculating dates.
// now may be in any zone: the periods follow its date in Copenhagen.
func getDatesFromPeriod(period Period, now time.Time) (from time.Time, to time.Time, err error) {
	now = now.In(cph)
	year, month, day := now.Date()
	startOfToday := time.Date(year, month, day, 0, 0, 0, 0, cph)
	firstOfThisMonth := time.Date(year, month, 1, 0, 0, 0, 0, cph)
	firstOfThisYear := time.Date(year, 1, 1, 0, 0, 0, 0, cph)
	// Go numbers the weekdays from Sunday (0), but weeks start on Monday.
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	startOfThisWeek := time.Date(year, month, day-daysSinceMonday, 0, 0, 0, 0, cph)

	name := strings.ToLower(string(period))
	switch name {
	case string(Yesterday):
		from = startOfToday.AddDate(0, 0, -1)
		to = startOfToday // Exclusive: the day that follows yesterday
	case string(ThisWeek):
		from = startOfThisWeek
		to = now
	case string(LastWeek):
		from = startOfThisWeek.AddDate(0, 0, -7)
		to = startOfThisWeek // Exclusive: the Monday that follows last week
	case string(ThisMonth):
		from = firstOfThisMonth
		to = now
	case string(LastMonth):
		from = firstOfThisMonth.AddDate(0, -1, 0)
		to = firstOfThisMonth // Exclusive: the first of this month
	case string(ThisYear):
		from = firstOfThisYear
		to = now
	case string(LastYear):
		from = firstOfThisYear.AddDate(-1, 0, 0)
		to = firstOfThisYear // Exclusive: 1 January this year
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid period: '%s'", period)
	}

	// The API reads from and to as Copenhagen dates and rejects an equal pair with error
	// 30002. Only a this_* period on its first day, from 00:00 today to now, gets one.
	if from.Format(time.DateOnly) == to.Format(time.DateOnly) {
		return time.Time{}, time.Time{}, fmt.Errorf("%s: %w", name, ErrorPeriodHasNoCompleteDay)
	}
	return from, to, nil
}
