package app

import (
	"database/sql"

	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/db"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/worktree"
)

// OpenWorktreeService opens the project database and returns a worktree service bound to it.
func (a *App) OpenWorktreeService(dbPath string) (*worktree.Service, *sql.DB, error) {
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}

	return worktree.NewService(sqlDB), sqlDB, nil
}
