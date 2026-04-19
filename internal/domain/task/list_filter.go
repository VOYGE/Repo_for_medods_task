package task

import "time"

// ListTasksFilter narrows task lists (e.g. calendar range, hide series templates).
type ListTasksFilter struct {
	IncludeTemplates bool
	OccurrenceFrom   *time.Time // UTC calendar day boundaries; nil = unbounded
	OccurrenceTo     *time.Time
}
