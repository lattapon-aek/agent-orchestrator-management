package worktree

import (
	"database/sql"
	"fmt"
	"time"
)

// Record is the persisted task-to-worktree mapping for Milestone 5.
type Record struct {
	TaskID       string
	ProjectID    string
	Status       string
	BaseBranch   string
	BranchName   string
	WorktreePath string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Repository persists durable worktree mappings.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a worktree repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a worktree mapping.
func (r *Repository) Upsert(record Record) error {
	_, err := r.db.Exec(`
INSERT INTO worktrees (
	task_id,
	project_id,
	status,
	base_branch,
	branch_name,
	worktree_path
)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET
	project_id = excluded.project_id,
	status = excluded.status,
	base_branch = excluded.base_branch,
	branch_name = excluded.branch_name,
	worktree_path = excluded.worktree_path,
	updated_at = CURRENT_TIMESTAMP
`,
		record.TaskID,
		record.ProjectID,
		record.Status,
		record.BaseBranch,
		record.BranchName,
		record.WorktreePath,
	)
	if err != nil {
		return fmt.Errorf("upsert worktree for task %q: %w", record.TaskID, err)
	}

	return nil
}

// GetByTaskID returns one worktree mapping by task ID.
func (r *Repository) GetByTaskID(taskID string) (*Record, error) {
	row := r.db.QueryRow(`
SELECT
	task_id,
	project_id,
	status,
	base_branch,
	branch_name,
	worktree_path,
	created_at,
	updated_at
FROM worktrees
WHERE task_id = ?
`,
		taskID,
	)

	var record Record
	if err := row.Scan(
		&record.TaskID,
		&record.ProjectID,
		&record.Status,
		&record.BaseBranch,
		&record.BranchName,
		&record.WorktreePath,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get worktree for task %q: %w", taskID, err)
	}

	return &record, nil
}

// ListByProjectID returns worktree mappings for one project ordered by update time.
func (r *Repository) ListByProjectID(projectID string) ([]Record, error) {
	rows, err := r.db.Query(`
SELECT
	task_id,
	project_id,
	status,
	base_branch,
	branch_name,
	worktree_path,
	created_at,
	updated_at
FROM worktrees
WHERE project_id = ?
ORDER BY updated_at DESC, created_at DESC, task_id DESC
`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list worktrees for project %q: %w", projectID, err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var record Record
		if err := rows.Scan(
			&record.TaskID,
			&record.ProjectID,
			&record.Status,
			&record.BaseBranch,
			&record.BranchName,
			&record.WorktreePath,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan worktree row: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate worktree rows: %w", err)
	}

	return records, nil
}
