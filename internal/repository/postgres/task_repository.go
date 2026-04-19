package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (
			title,
			description,
			status,
			created_at,
			updated_at,
			series_id,
			template_id,
			occurrence_date,
			is_template,
			recurrence_kind,
			recurrence_config
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11
		)
		RETURNING
			id,
			title,
			description,
			status,
			created_at,
			updated_at,
			series_id,
			template_id,
			occurrence_date,
			is_template,
			recurrence_kind,
			recurrence_config
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		task.CreatedAt,
		task.UpdatedAt,
		uuidPtrToPg(task.SeriesID),
		int64PtrToPg(task.TemplateID),
		datePtrToPg(task.OccurrenceDate),
		task.IsTemplate,
		recurrenceKindToPg(task.RecurrenceKind),
		recurrenceConfigToPg(task.Recurrence),
	)

	return scanFullTask(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT
			id,
			title,
			description,
			status,
			created_at,
			updated_at,
			series_id,
			template_id,
			occurrence_date,
			is_template,
			recurrence_kind,
			recurrence_config
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)

	found, err := scanFullTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET
			title = $1,
			description = $2,
			status = $3,
			updated_at = $4,
			recurrence_kind = $5,
			recurrence_config = $6
		WHERE id = $7
		RETURNING
			id,
			title,
			description,
			status,
			created_at,
			updated_at,
			series_id,
			template_id,
			occurrence_date,
			is_template,
			recurrence_kind,
			recurrence_config
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		task.UpdatedAt,
		recurrenceKindToPg(task.RecurrenceKind),
		recurrenceConfigToPg(task.Recurrence),
		task.ID,
	)

	updated, err := scanFullTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context, filter taskdomain.ListTasksFilter) ([]taskdomain.Task, error) {
	const query = `
		SELECT
			id,
			title,
			description,
			status,
			created_at,
			updated_at,
			series_id,
			template_id,
			occurrence_date,
			is_template,
			recurrence_kind,
			recurrence_config
		FROM tasks
		WHERE
			($1::boolean OR NOT is_template)
			AND (
				($2::date IS NULL AND $3::date IS NULL)
				OR occurrence_date IS NULL
				OR (
					($2::date IS NULL OR occurrence_date >= $2::date)
					AND ($3::date IS NULL OR occurrence_date <= $3::date)
				)
			)
		ORDER BY occurrence_date DESC NULLS LAST, id DESC
	`

	from := pgtype.Date{}
	to := pgtype.Date{}

	if filter.OccurrenceFrom != nil {
		t := filter.OccurrenceFrom.UTC()
		from = pgtype.Date{
			Time:  time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
			Valid: true,
		}
	}

	if filter.OccurrenceTo != nil {
		t := filter.OccurrenceTo.UTC()
		to = pgtype.Date{
			Time:  time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
			Valid: true,
		}
	}

	rows, err := r.pool.Query(ctx, query, filter.IncludeTemplates, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)

	for rows.Next() {
		task, err := scanFullTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) OccurrenceExists(ctx context.Context, seriesID uuid.UUID, occurrenceDay time.Time) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM tasks
			WHERE series_id = $1
				AND occurrence_date = $2::date
		)
	`

	var exists bool

	t := occurrenceDay.UTC()

	err := r.pool.QueryRow(
		ctx,
		query,
		seriesID,
		pgtype.Date{
			Time:  time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanFullTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task           taskdomain.Task
		status         string
		series         pgtype.UUID
		templateID     pgtype.Int8
		occurrenceDate pgtype.Date
		recurrenceKind pgtype.Text
		recurrenceRaw  []byte
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&series,
		&templateID,
		&occurrenceDate,
		&task.IsTemplate,
		&recurrenceKind,
		&recurrenceRaw,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	if series.Valid {
		id := uuid.UUID(series.Bytes)
		task.SeriesID = &id
	}

	if templateID.Valid {
		v := templateID.Int64
		task.TemplateID = &v
	}

	if occurrenceDate.Valid {
		t := occurrenceDate.Time.UTC()
		midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		task.OccurrenceDate = &midnight
	}

	if recurrenceKind.Valid {
		k := taskdomain.RecurrenceKind(recurrenceKind.String)
		task.RecurrenceKind = &k
	}

	if len(recurrenceRaw) > 0 {
		var rule taskdomain.RecurrenceRule

		if err := json.Unmarshal(recurrenceRaw, &rule); err != nil {
			return nil, err
		}

		task.Recurrence = &rule
	}

	return &task, nil
}

func uuidPtrToPg(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}

	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

func int64PtrToPg(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{
		Int64: *v,
		Valid: true,
	}
}

func datePtrToPg(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}

	return pgtype.Date{
		Time:  t.UTC(),
		Valid: true,
	}
}

func recurrenceKindToPg(kind *taskdomain.RecurrenceKind) pgtype.Text {
	if kind == nil {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: string(*kind),
		Valid:  true,
	}
}

func recurrenceConfigToPg(rule *taskdomain.RecurrenceRule) interface{} {
	if rule == nil {
		return nil
	}

	raw, err := json.Marshal(rule)
	if err != nil {
		return nil
	}

	return raw
}
