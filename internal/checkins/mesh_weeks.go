package checkins

import "time"

// CountWeeklyNetWeeksInclusive returns how many WeeklyNet week buckets fall between
// the week-start of trackFrom and the week-start of through, inclusive,
// using tz for calendar boundaries (same as WeekStartForWeekday).
func CountWeeklyNetWeeksInclusive(trackFrom, through time.Time, tz string, dayOfWeek time.Weekday) int {
	from := WeekStartForWeekday(trackFrom, tz, dayOfWeek)
	to := WeekStartForWeekday(through, tz, dayOfWeek)
	if to.Before(from) {
		return 0
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc)
	days := int(toDay.Sub(fromDay).Hours() / 24)
	return days/7 + 1
}
