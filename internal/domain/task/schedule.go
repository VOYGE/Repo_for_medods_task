package task

import (
	"sort"
	"time"
)

const dateLayout = "2006-01-02"

// OccurrenceDatesUTC returns calendar dates in [from, to] (inclusive) that match the rule.
// Dates are interpreted in UTC (date-only semantics).
func OccurrenceDatesUTC(rule RecurrenceRule, from, to time.Time) ([]time.Time, error) {
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	fromDay := truncateToUTCDate(from)
	toDay := truncateToUTCDate(to)
	if toDay.Before(fromDay) {
		return nil, nil
	}

	switch rule.Kind {
	case RecurrenceKindDailyInterval:
		return dailyIntervalDates(*rule.EveryNDays, *rule.AnchorDate, fromDay, toDay), nil
	case RecurrenceKindMonthlyDay:
		return monthlyDayDates(*rule.DayOfMonth, fromDay, toDay), nil
	case RecurrenceKindSpecificDates:
		return specificDates(rule.Dates, fromDay, toDay), nil
	case RecurrenceKindDayParity:
		return parityDates(*rule.Parity, fromDay, toDay), nil
	default:
		return nil, ErrInvalidRecurrence
	}
}

func truncateToUTCDate(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func parseDate(s string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, s, time.UTC)
}

func dailyIntervalDates(every int, anchor string, fromDay, toDay time.Time) []time.Time {
	start, err := parseDate(anchor)
	if err != nil {
		return nil
	}
	start = truncateToUTCDate(start)

	d := start
	for d.Before(fromDay) {
		d = d.AddDate(0, 0, every)
	}

	var out []time.Time
	for !d.After(toDay) {
		out = append(out, d)
		d = d.AddDate(0, 0, every)
	}

	return out
}

func monthlyDayDates(dayOfMonth int, fromDay, toDay time.Time) []time.Time {
	var out []time.Time

	cy, cm := fromDay.Year(), fromDay.Month()
	endY, endM := toDay.Year(), toDay.Month()

	for {
		last := lastDayOfMonth(cy, cm)
		actual := dayOfMonth
		if actual > last.Day() {
			if cy == endY && cm == endM {
				break
			}

			cy, cm = advanceMonth(cy, cm)

			continue
		}

		candidate := time.Date(cy, cm, actual, 0, 0, 0, 0, time.UTC)
		if !candidate.Before(fromDay) && !candidate.After(toDay) {
			out = append(out, candidate)
		}

		if cy == endY && cm == endM {
			break
		}

		cy, cm = advanceMonth(cy, cm)
	}

	return out
}

func advanceMonth(y int, m time.Month) (int, time.Month) {
	if m == time.December {
		return y + 1, time.January
	}

	return y, m + 1
}

func lastDayOfMonth(y int, m time.Month) time.Time {
	firstNext := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
	if m == time.December {
		firstNext = time.Date(y+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	}

	return firstNext.AddDate(0, 0, -1)
}

func specificDates(dates []string, fromDay, toDay time.Time) []time.Time {
	var out []time.Time

	for _, s := range dates {
		d, err := parseDate(s)
		if err != nil {
			continue
		}

		d = truncateToUTCDate(d)
		if !d.Before(fromDay) && !d.After(toDay) {
			out = append(out, d)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Before(out[j])
	})

	return dedupeSortedDates(out)
}

func parityDates(parity string, fromDay, toDay time.Time) []time.Time {
	var out []time.Time

	wantEven := parity == ParityEven

	for d := fromDay; !d.After(toDay); d = d.AddDate(0, 0, 1) {
		dayNum := d.Day()
		isEven := dayNum%2 == 0
		if wantEven && isEven || !wantEven && !isEven {
			out = append(out, d)
		}
	}

	return out
}

func dedupeSortedDates(in []time.Time) []time.Time {
	if len(in) == 0 {
		return in
	}

	out := []time.Time{in[0]}
	for i := 1; i < len(in); i++ {
		if truncateToUTCDate(in[i]).Equal(truncateToUTCDate(out[len(out)-1])) {
			continue
		}

		out = append(out, in[i])
	}

	return out
}
