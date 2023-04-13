package options

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// FactsGenerator gathers facts
type FactsGenerator func(ctx context.Context, cfg Options, log *logrus.Entry) (map[string]any, error)

type Options struct {
	// Name is the name reported in --help and other output from the command line
	Name string
	// Contact will be shown during --help
	Contact string
	// Help will be shown during --help as the main command help
	Help string
	// Version will be reported in --version and elsewhere
	Version string
	// ProvisioningJWTFile is the path to provisioning jwt file, defaults to provisioning.jwt in the options dir
	ProvisioningJWTFile string
	// FactsFile is the path to the facts file which default to instance.json in the options dir
	FactsFile string
	// FactsRefreshInterval sets a interval to refresh facts on, 10 minutes by default and cannot be less than 1 minute
	FactsRefreshInterval time.Duration
	// AdditionalFacts will be called during fact generation and the result will be shallow merged with the standard facts
	AdditionalFacts FactsGenerator
	// CommandPath is the path to the command being run, defaults to argv[0]
	CommandPath string
	// MachineSigningKey hex encoded ed25519 key used to sign autonomous agents
	MachineSigningKey string
}
