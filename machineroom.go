package machineroom

import (
	"context"

	"github.com/choria-io/fisk"
	"github.com/ripienaar/machine-room/internal/cli"
	"github.com/ripienaar/machine-room/options"
	"github.com/sirupsen/logrus"
)

// Instance is an instance of the Choria Machine Room Agent
type Instance interface {
	// Run starts running the command line
	Run(ctx context.Context) error
	// Application allows adding additional commands to the CLI application that will be built
	Application() *fisk.Application
	// CommonConfigure performs basic setup that a command added using Application() might need
	CommonConfigure() (*options.Options, *logrus.Entry, error)
}

// New creates a new machine room agent instance based on options
func New(o options.Options) (Instance, error) {
	return cli.New(o)
}
