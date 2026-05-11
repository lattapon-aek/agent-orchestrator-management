package app

import "github.com/lattapon-aek/Agents-Orchestfator-Management/internal/project"

// App holds top-level application dependencies as the CLI grows.
type App struct {
	Projects *project.Service
}

// New creates a new application container with default wiring.
func New() *App {
	return &App{
		Projects: project.NewService(),
	}
}
