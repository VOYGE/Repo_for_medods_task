package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	taskdomain "example.com/taskservice/internal/domain/task"
)

const defaultMaterializeHorizonDays = 30

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	if input.Recurrence != nil {
		if err := input.Recurrence.Validate(); err != nil {
			return nil, err
		}

		days := input.MaterializeHorizonDays
		if days <= 0 {
			days = defaultMaterializeHorizonDays
		}

		return s.createRecurringTemplate(ctx, normalized, input.Recurrence, days)
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
	}

	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) createRecurringTemplate(
	ctx context.Context,
	normalized CreateInput,
	rule *taskdomain.RecurrenceRule,
	horizonDays int,
) (*taskdomain.Task, error) {
	sid := uuid.New()
	kind := rule.Kind

	model := &taskdomain.Task{
		Title:          normalized.Title,
		Description:    normalized.Description,
		Status:         normalized.Status,
		SeriesID:       &sid,
		IsTemplate:     true,
		RecurrenceKind: &kind,
		Recurrence:     rule,
	}

	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	from := truncateUTCDate(s.now())
	to := from.AddDate(0, 0, horizonDays)

	if _, err := s.materializeOccurrences(ctx, created, from, to); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Title = normalized.Title
	existing.Description = normalized.Description
	existing.Status = normalized.Status
	existing.UpdatedAt = s.now()

	if existing.IsTemplate && input.Recurrence != nil {
		if err := input.Recurrence.Validate(); err != nil {
			return nil, err
		}

		kind := input.Recurrence.Kind
		existing.RecurrenceKind = &kind
		existing.Recurrence = input.Recurrence
	}

	return s.repo.Update(ctx, existing)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, filter taskdomain.ListTasksFilter) ([]taskdomain.Task, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Materialize(ctx context.Context, templateID int64, from, to time.Time) (int, error) {
	if templateID <= 0 {
		return 0, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	fromDay := truncateUTCDate(from)
	toDay := truncateUTCDate(to)

	if toDay.Before(fromDay) {
		return 0, fmt.Errorf("%w: from must be on or before to", ErrInvalidInput)
	}

	template, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return 0, err
	}

	if !template.IsTemplate {
		return 0, ErrNotATemplate
	}

	return s.materializeOccurrences(ctx, template, fromDay, toDay)
}

func (s *Service) materializeOccurrences(
	ctx context.Context,
	template *taskdomain.Task,
	fromDay, toDay time.Time,
) (int, error) {
	if template.Recurrence == nil {
		return 0, fmt.Errorf("%w: template has no recurrence configuration", ErrInvalidInput)
	}

	if template.SeriesID == nil {
		return 0, fmt.Errorf("%w: template has no series id", ErrInvalidInput)
	}

	dates, err := taskdomain.OccurrenceDatesUTC(*template.Recurrence, fromDay, toDay)
	if err != nil {
		return 0, err
	}

	created := 0
	now := s.now()

	for _, d := range dates {
		exists, err := s.repo.OccurrenceExists(ctx, *template.SeriesID, d)
		if err != nil {
			return created, err
		}

		if exists {
			continue
		}

		tid := template.ID
		midnight := truncateUTCDate(d)

		occ := &taskdomain.Task{
			Title:          template.Title,
			Description:    template.Description,
			Status:         taskdomain.StatusNew,
			CreatedAt:      now,
			UpdatedAt:      now,
			SeriesID:       template.SeriesID,
			TemplateID:     &tid,
			OccurrenceDate: &midnight,
			IsTemplate:     false,
		}

		if _, err := s.repo.Create(ctx, occ); err != nil {
			return created, err
		}

		created++
	}

	return created, nil
}

func truncateUTCDate(t time.Time) time.Time {
	u := t.UTC()

	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}
