package task

import (
	"database/sql"
	"fmt"
	"time"
)

// Record is the persisted workflow task model for Milestone 3.
type Record struct {
	ID             string
	ProjectID      string
	Title          string
	Mode           string
	Status         string
	PreferredRole  string
	PreferredAgent string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Repository persists durable task state.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a task repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a task record.
func (r *Repository) Upsert(record Record) error {
	_, err := r.db.Exec(`
INSERT INTO tasks (
	id,
	project_id,
	title,
	mode,
	status,
	preferred_role,
	preferred_agent
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	project_id = excluded.project_id,
	title = excluded.title,
	mode = excluded.mode,
	status = excluded.status,
	preferred_role = excluded.preferred_role,
	preferred_agent = excluded.preferred_agent,
	updated_at = CURRENT_TIMESTAMP
`,
		record.ID,
		record.ProjectID,
		record.Title,
		record.Mode,
		record.Status,
		record.PreferredRole,
		record.PreferredAgent,
	)
	if err != nil {
		return fmt.Errorf("upsert task %q: %w", record.ID, err)
	}

	return nil
}

// GetByID returns one task record by durable task ID.
func (r *Repository) GetByID(id string) (*Record, error) {
	row := r.db.QueryRow(`
SELECT
	id,
	project_id,
	title,
	mode,
	status,
	preferred_role,
	preferred_agent,
	created_at,
	updated_at
FROM tasks
WHERE id = ?
`,
		id,
	)

	record, err := scanRecord(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get task %q: %w", id, err)
	}

	return record, nil
}

// CountByProjectID returns the durable task count for one project.
func (r *Repository) CountByProjectID(projectID string) (int, error) {
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(1) FROM tasks WHERE project_id = ?`, projectID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tasks for project %q: %w", projectID, err)
	}

	return count, nil
}

// ListByProjectID returns task records for one project ordered by update time.
func (r *Repository) ListByProjectID(projectID string) ([]Record, error) {
	rows, err := r.db.Query(`
SELECT
	id,
	project_id,
	title,
	mode,
	status,
	preferred_role,
	preferred_agent,
	created_at,
	updated_at
FROM tasks
WHERE project_id = ?
ORDER BY updated_at DESC, created_at DESC, id DESC
`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks for project %q: %w", projectID, err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		records = append(records, *record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}

	return records, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecord(scanner rowScanner) (*Record, error) {
	var record Record
	if err := scanner.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Title,
		&record.Mode,
		&record.Status,
		&record.PreferredRole,
		&record.PreferredAgent,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &record, nil
}
