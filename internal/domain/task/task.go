package task

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	SeriesID       *uuid.UUID       `json:"series_id,omitempty"`
	TemplateID     *int64           `json:"template_id,omitempty"`
	OccurrenceDate *time.Time       `json:"occurrence_date,omitempty"` // UTC midnight for calendar day
	IsTemplate     bool             `json:"is_template"`
	RecurrenceKind *RecurrenceKind  `json:"recurrence_kind,omitempty"`
	Recurrence     *RecurrenceRule `json:"recurrence,omitempty"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}
