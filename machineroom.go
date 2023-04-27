package machineroom

import (
	"github.com/ripienaar/machine-room/internal/cli"
	"github.com/ripienaar/machine-room/options"
)

// New creates a new machine room agent instance based on options
func New(o options.Options) (*cli.CLI, error) {
	return cli.New(o)
}
