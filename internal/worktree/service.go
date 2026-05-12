package worktree

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultStatus = "Planned"

// CreateParams describes the minimum input needed to create a planned worktree mapping.
type CreateParams struct {
	ProjectID     string
	TaskID        string
	TaskTitle     string
	RepoPath      string
	DefaultBranch string
}

// Service owns worktree mapping behavior for Milestone 5.
type Service struct {
	repo     *Repository
	lookPath func(string) (string, error)
	runGit   func(repoPath string, args ...string) ([]byte, error)
	stat     func(string) (os.FileInfo, error)
	mkdirAll func(string, os.FileMode) error
}

// NewService creates a worktree service backed by the provided database.
func NewService(db *sql.DB) *Service {
	return &Service{
		repo:     NewRepository(db),
		lookPath: exec.LookPath,
		runGit: func(repoPath string, args ...string) ([]byte, error) {
			cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
			return cmd.CombinedOutput()
		},
		stat:     os.Stat,
		mkdirAll: os.MkdirAll,
	}
}

// CreatePlanned inserts or updates the planned worktree mapping for one task.
func (s *Service) CreatePlanned(params CreateParams) (*Record, error) {
	projectID := strings.TrimSpace(params.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id is required")
	}
	taskID := strings.TrimSpace(params.TaskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	repoPath := strings.TrimSpace(params.RepoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("repo path is required")
	}
	defaultBranch := strings.TrimSpace(params.DefaultBranch)
	if defaultBranch == "" {
		return nil, fmt.Errorf("default branch is required")
	}

	record := Record{
		TaskID:       taskID,
		ProjectID:    projectID,
		Status:       defaultStatus,
		BaseBranch:   defaultBranch,
		BranchName:   plannedBranchName(taskID, params.TaskTitle),
		WorktreePath: plannedWorktreePath(repoPath, taskID, params.TaskTitle),
	}

	if err := s.repo.Upsert(record); err != nil {
		return nil, err
	}

	return s.repo.GetByTaskID(taskID)
}

// GetByTask returns one worktree mapping by task ID.
func (s *Service) GetByTask(taskID string) (*Record, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	return s.repo.GetByTaskID(taskID)
}

// ListByProject returns all worktree mappings for one project.
func (s *Service) ListByProject(projectID string) ([]Record, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id is required")
	}

	return s.repo.ListByProjectID(projectID)
}

// EnsureProvisioned upgrades a planned mapping to Ready when the repo supports git worktrees.
func (s *Service) EnsureProvisioned(taskID, repoPath string) (*Record, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("repo path is required")
	}

	record, err := s.repo.GetByTaskID(taskID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("worktree for task %q not found", taskID)
	}

	if record.Status == "Ready" {
		return record, nil
	}

	if _, err := s.lookPath("git"); err != nil {
		return record, nil
	}

	if _, err := s.runGit(repoPath, "rev-parse", "--is-inside-work-tree"); err != nil {
		return record, nil
	}

	if err := s.mkdirAll(filepath.Dir(record.WorktreePath), 0o755); err != nil {
		return nil, fmt.Errorf("create worktree parent dir: %w", err)
	}

	if _, err := s.stat(record.WorktreePath); err == nil {
		record.Status = "Ready"
		if err := s.repo.Upsert(*record); err != nil {
			return nil, err
		}
		return s.repo.GetByTaskID(taskID)
	}

	if output, err := s.runGit(repoPath, "worktree", "add", "-b", record.BranchName, record.WorktreePath, record.BaseBranch); err != nil {
		return nil, fmt.Errorf("provision worktree for task %q: %s", taskID, strings.TrimSpace(string(output)))
	}

	record.Status = "Ready"
	if err := s.repo.Upsert(*record); err != nil {
		return nil, err
	}

	return s.repo.GetByTaskID(taskID)
}

func plannedBranchName(taskID, taskTitle string) string {
	return "aom/" + sanitizeSegment(taskID) + "-" + sanitizeSegment(taskTitle)
}

func plannedWorktreePath(repoPath, taskID, taskTitle string) string {
	dirName := sanitizeSegment(taskID) + "-" + sanitizeSegment(taskTitle)
	return filepath.Join(repoPath, ".aom", "worktrees", dirName)
}

func sanitizeSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")

	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "task"
	}
	return result
}
