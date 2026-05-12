package step

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Record is the persisted workflow step model for Milestone 3.
type Record struct {
	ID           string
	ProjectID    string
	TaskID       string
	StepType     string
	Title        string
	Status       string
	RoleName     string
	AgentName    string
	Dependencies []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Repository persists durable step state.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a step repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a step record.
func (r *Repository) Upsert(record Record) error {
	_, err := r.db.Exec(`
INSERT INTO steps (
	id,
	project_id,
	task_id,
	step_type,
	title,
	status,
	role_name,
	agent_name,
	dependencies
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	project_id = excluded.project_id,
	task_id = excluded.task_id,
	step_type = excluded.step_type,
	title = excluded.title,
	status = excluded.status,
	role_name = excluded.role_name,
	agent_name = excluded.agent_name,
	dependencies = excluded.dependencies,
	updated_at = CURRENT_TIMESTAMP
`,
		record.ID,
		record.ProjectID,
		record.TaskID,
		record.StepType,
		record.Title,
		record.Status,
		record.RoleName,
		record.AgentName,
		joinDependencies(record.Dependencies),
	)
	if err != nil {
		return fmt.Errorf("upsert step %q: %w", record.ID, err)
	}

	return nil
}

// GetByID returns one step record by durable step ID.
func (r *Repository) GetByID(id string) (*Record, error) {
	row := r.db.QueryRow(`
SELECT
	id,
	project_id,
	task_id,
	step_type,
	title,
	status,
	role_name,
	agent_name,
	dependencies,
	created_at,
	updated_at
FROM steps
WHERE id = ?
`,
		id,
	)

	record, err := scanRecord(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get step %q: %w", id, err)
	}

	return record, nil
}

// ListByTaskID returns step records for one task ordered by creation time.
func (r *Repository) ListByTaskID(taskID string) ([]Record, error) {
	rows, err := r.db.Query(`
SELECT
	id,
	project_id,
	task_id,
	step_type,
	title,
	status,
	role_name,
	agent_name,
	dependencies,
	created_at,
	updated_at
FROM steps
WHERE task_id = ?
ORDER BY created_at, id
`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list steps for task %q: %w", taskID, err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan step row: %w", err)
		}
		records = append(records, *record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate step rows: %w", err)
	}

	return records, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(scanner rowScanner) (*Record, error) {
	var record Record
	var dependencies string
	if err := scanner.Scan(
		&record.ID,
		&record.ProjectID,
		&record.TaskID,
		&record.StepType,
		&record.Title,
		&record.Status,
		&record.RoleName,
		&record.AgentName,
		&dependencies,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, err
	}

	record.Dependencies = splitDependencies(dependencies)
	return &record, nil
}

func joinDependencies(values []string) string {
	if len(values) == 0 {
		return ""
	}

	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}

	return strings.Join(cleaned, ",")
}

func splitDependencies(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}

	return result
}
