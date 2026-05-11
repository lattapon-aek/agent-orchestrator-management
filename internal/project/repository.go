package project

import (
	"database/sql"
	"fmt"
)

// Record is the persisted project model for Milestone 1.
type Record struct {
	ID            string
	Name          string
	RepoPath      string
	DefaultBranch string
}

// Repository persists project state.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a project repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a project record.
func (r *Repository) Upsert(record Record) error {
	_, err := r.db.Exec(`
INSERT INTO projects (id, name, repo_path, default_branch)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	repo_path = excluded.repo_path,
	default_branch = excluded.default_branch
`,
		record.ID,
		record.Name,
		record.RepoPath,
		record.DefaultBranch,
	)
	if err != nil {
		return fmt.Errorf("upsert project %q: %w", record.ID, err)
	}

	return nil
}

// FindByRepoPath returns the project record registered for the given repo path.
func (r *Repository) FindByRepoPath(repoPath string) (*Record, error) {
	row := r.db.QueryRow(`
SELECT id, name, repo_path, default_branch
FROM projects
WHERE repo_path = ?
`,
		repoPath,
	)

	var record Record
	if err := row.Scan(&record.ID, &record.Name, &record.RepoPath, &record.DefaultBranch); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find project by repo path %q: %w", repoPath, err)
	}

	return &record, nil
}
