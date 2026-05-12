package step

import (
	"database/sql"
	"fmt"
	"strings"
)

// UpdateParams describes mutable step fields in Milestone 3.
type UpdateParams struct {
	Status    string
	RoleName  string
	AgentName string
}

// Service owns step retrieval and update behavior for Milestone 3.
type Service struct {
	repo *Repository
}

// NewService creates a step service backed by the provided database.
func NewService(db *sql.DB) *Service {
	return &Service{repo: NewRepository(db)}
}

// Get returns one step by ID.
func (s *Service) Get(id string) (*Record, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("step id is required")
	}

	return s.repo.GetByID(strings.TrimSpace(id))
}

// ListByTask returns all steps for one task.
func (s *Service) ListByTask(taskID string) ([]Record, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("task id is required")
	}

	return s.repo.ListByTaskID(strings.TrimSpace(taskID))
}

// Update mutates step ownership or status with transition validation.
func (s *Service) Update(id string, params UpdateParams) (*Record, error) {
	record, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("step %q not found", strings.TrimSpace(id))
	}

	next := *record
	changed := false

	if params.RoleName != "" {
		next.RoleName = strings.TrimSpace(params.RoleName)
		changed = true
	}
	if params.AgentName != "" {
		next.AgentName = strings.TrimSpace(params.AgentName)
		changed = true
	}
	if params.Status != "" {
		status, err := normalizeStatus(params.Status)
		if err != nil {
			return nil, err
		}
		if err := validateTransition(record.Status, status); err != nil {
			return nil, err
		}
		next.Status = status
		changed = true
	}

	if !changed {
		return nil, fmt.Errorf("at least one step field must be updated")
	}
	if next.Status == "Ready" && strings.TrimSpace(next.RoleName) == "" && strings.TrimSpace(next.AgentName) == "" {
		return nil, fmt.Errorf("step %q needs a role or agent before entering Ready", next.ID)
	}

	if err := s.repo.Upsert(next); err != nil {
		return nil, err
	}

	return s.repo.GetByID(next.ID)
}

func normalizeStatus(input string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "proposed":
		return "Proposed", nil
	case "confirmed":
		return "Confirmed", nil
	case "ready":
		return "Ready", nil
	case "inprogress", "in-progress":
		return "InProgress", nil
	case "blocked":
		return "Blocked", nil
	case "needsattention", "needs-attention":
		return "NeedsAttention", nil
	case "completed":
		return "Completed", nil
	case "skipped":
		return "Skipped", nil
	case "canceled", "cancelled":
		return "Canceled", nil
	default:
		return "", fmt.Errorf("step status %q is not recognized", input)
	}
}

func validateTransition(current, next string) error {
	if current == next {
		return nil
	}

	allowed := map[string]map[string]bool{
		"Proposed": {
			"Confirmed": true,
			"Skipped":   true,
			"Canceled":  true,
		},
		"Confirmed": {
			"Ready":    true,
			"Canceled": true,
		},
		"Ready": {
			"InProgress": true,
			"Skipped":    true,
			"Canceled":   true,
		},
		"InProgress": {
			"Blocked":        true,
			"NeedsAttention": true,
			"Completed":      true,
			"Ready":          true,
		},
		"Blocked": {
			"Ready":          true,
			"NeedsAttention": true,
		},
		"NeedsAttention": {
			"Ready":      true,
			"InProgress": true,
			"Canceled":   true,
		},
	}

	if allowed[current][next] {
		return nil
	}

	return fmt.Errorf("step transition %s -> %s is not allowed", current, next)
}
