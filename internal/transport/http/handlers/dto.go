package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      taskdomain.Status `json:"status"`

	Recurrence *taskdomain.RecurrenceRule `json:"recurrence"`

	// MaterializeHorizonDays is used only when recurrence is set (defaults in the usecase if <= 0).
	MaterializeHorizonDays int `json:"materialize_horizon_days"`
}

type updateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      taskdomain.Status `json:"status"`

	Recurrence *taskdomain.RecurrenceRule `json:"recurrence"`
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	SeriesID       *string                   `json:"series_id,omitempty"`
	TemplateID     *int64                    `json:"template_id,omitempty"`
	OccurrenceDate *string                   `json:"occurrence_date,omitempty"`
	IsTemplate     bool                      `json:"is_template"`
	RecurrenceKind *taskdomain.RecurrenceKind `json:"recurrence_kind,omitempty"`
	Recurrence     *taskdomain.RecurrenceRule `json:"recurrence,omitempty"`
}

type materializeRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type materializeResponse struct {
	Created int `json:"created"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		IsTemplate:  task.IsTemplate,
		TemplateID:  task.TemplateID,
		RecurrenceKind: task.RecurrenceKind,
		Recurrence:     task.Recurrence,
	}

	if task.SeriesID != nil {
		s := task.SeriesID.String()
		dto.SeriesID = &s
	}

	if task.OccurrenceDate != nil {
		s := task.OccurrenceDate.UTC().Format("2006-01-02")
		dto.OccurrenceDate = &s
	}

	return dto
}
