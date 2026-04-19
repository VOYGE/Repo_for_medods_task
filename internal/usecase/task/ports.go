package task

import (
	"context"
	"time"

	"github.com/google/uuid"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter taskdomain.ListTasksFilter) ([]taskdomain.Task, error)

	OccurrenceExists(ctx context.Context, seriesID uuid.UUID, occurrenceDay time.Time) (bool, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter taskdomain.ListTasksFilter) ([]taskdomain.Task, error)

	Materialize(ctx context.Context, templateID int64, from, to time.Time) (int, error)
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status

	Recurrence *taskdomain.RecurrenceRule

	// MaterializeHorizonDays controls initial generation window for recurrence templates (UTC days from today).
	MaterializeHorizonDays int
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status

	Recurrence *taskdomain.RecurrenceRule
}
