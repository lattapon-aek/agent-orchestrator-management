package app

import (
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/config"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/project"
	"github.com/lattapon-aek/Agents-Orchestfator-Management/internal/provider"
)

func init() {
	for name := range provider.DefaultRegistry() {
		config.RegisterKnownRuntime(name)
		project.RegisterKnownInitRuntime(name)
	}
}
