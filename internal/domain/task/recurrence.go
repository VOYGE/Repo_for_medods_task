package task

import (
	"fmt"
	"time"
)

type RecurrenceKind string

const (
	RecurrenceKindDailyInterval RecurrenceKind = "daily_interval"
	RecurrenceKindMonthlyDay    RecurrenceKind = "monthly_day"
	RecurrenceKindSpecificDates RecurrenceKind = "specific_dates"
	RecurrenceKindDayParity     RecurrenceKind = "day_parity"
)

const (
	ParityEven = "even"
	ParityOdd  = "odd"
)

// RecurrenceRule is persisted in recurrence_config (JSONB) and mirrors API shape.
type RecurrenceRule struct {
	Kind RecurrenceKind `json:"kind"`

	EveryNDays *int     `json:"every_n_days,omitempty"`
	AnchorDate *string  `json:"anchor_date,omitempty"` // YYYY-MM-DD (UTC calendar date)
	DayOfMonth *int     `json:"day_of_month,omitempty"`
	Dates      []string `json:"dates,omitempty"`
	Parity     *string  `json:"parity,omitempty"`
}

func (r RecurrenceRule) Validate() error {
	switch r.Kind {
	case RecurrenceKindDailyInterval:
		if r.EveryNDays == nil || *r.EveryNDays < 1 {
			return fmt.Errorf("%w: every_n_days must be >= 1", ErrInvalidRecurrence)
		}
		if r.AnchorDate == nil || *r.AnchorDate == "" {
			return fmt.Errorf("%w: anchor_date is required", ErrInvalidRecurrence)
		}
		if _, err := time.Parse("2006-01-02", *r.AnchorDate); err != nil {
			return fmt.Errorf("%w: anchor_date must be YYYY-MM-DD", ErrInvalidRecurrence)
		}
	case RecurrenceKindMonthlyDay:
		if r.DayOfMonth == nil || *r.DayOfMonth < 1 || *r.DayOfMonth > 30 {
			return fmt.Errorf("%w: day_of_month must be 1..30", ErrInvalidRecurrence)
		}
	case RecurrenceKindSpecificDates:
		if len(r.Dates) == 0 {
			return fmt.Errorf("%w: dates must not be empty", ErrInvalidRecurrence)
		}
		for _, d := range r.Dates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("%w: invalid date %q", ErrInvalidRecurrence, d)
			}
		}
	case RecurrenceKindDayParity:
		if r.Parity == nil {
			return fmt.Errorf("%w: parity is required", ErrInvalidRecurrence)
		}
		switch *r.Parity {
		case ParityEven, ParityOdd:
		default:
			return fmt.Errorf("%w: parity must be even or odd", ErrInvalidRecurrence)
		}
	default:
		return fmt.Errorf("%w: unknown recurrence kind", ErrInvalidRecurrence)
	}

	return nil
}
