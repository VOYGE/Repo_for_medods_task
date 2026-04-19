ALTER TABLE tasks
	ADD COLUMN IF NOT EXISTS series_id UUID,
	ADD COLUMN IF NOT EXISTS template_id BIGINT REFERENCES tasks (id) ON DELETE CASCADE,
	ADD COLUMN IF NOT EXISTS occurrence_date DATE,
	ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT FALSE,
	ADD COLUMN IF NOT EXISTS recurrence_kind TEXT,
	ADD COLUMN IF NOT EXISTS recurrence_config JSONB;

CREATE INDEX IF NOT EXISTS idx_tasks_series_id ON tasks (series_id);
CREATE INDEX IF NOT EXISTS idx_tasks_occurrence_date ON tasks (occurrence_date);
CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks (template_id);
CREATE INDEX IF NOT EXISTS idx_tasks_is_template ON tasks (is_template);

CREATE UNIQUE INDEX IF NOT EXISTS ux_tasks_series_occurrence
	ON tasks (series_id, occurrence_date)
	WHERE
		occurrence_date IS NOT NULL;
