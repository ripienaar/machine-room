package options

import (
	"context"
	"time"

	"github.com/choria-io/go-choria/plugin"
	"github.com/sirupsen/logrus"
)

const (
	ConfigKeySourceHost    = "machine_room.source.host"
	ConfigKeySourceNatsJwt = "machine_room.source.nats_jwt"
	ConfigKeyRole          = "machine_room.role"
	ConfigKeySite          = "machine_room.site"
)

// FactsGenerator gathers facts
type FactsGenerator func(ctx context.Context, cfg Options, log *logrus.Entry) (map[string]any, error)

// Options holds configuration and runtime derived paths, members marked RO are set during CommonConfigure(), setting them has no effect
type Options struct {
	// Name is the name reported in --help and other output from the command line
	Name string `json:"name"`
	// Contact will be shown during --help
	Contact string `json:"contact"`
	// Help will be shown during --help as the main command help
	Help string `json:"help"`
	// Version will be reported in --version and elsewhere
	Version string `json:"version"`
	// MachineSigningKey hex encoded ed25519 key used to sign autonomous agents
	MachineSigningKey string `json:"machine_signing_key"`

	// optional below

	// Plugins are additional plugins like autonomous agents to add to the build
	Plugins map[string]plugin.Pluggable `json:"-"`
	// FactsRefreshInterval sets an interval to refresh facts on, 10 minutes by default and cannot be less than 1 minute
	FactsRefreshInterval time.Duration `json:"facts_refresh_interval"`
	// AdditionalFacts will be called during fact generation and the result will be shallow merged with the standard facts
	AdditionalFacts FactsGenerator `json:"-"`
	// NoStandardFacts disables gathering all standard facts
	NoStandardFacts bool `json:"no_standard_facts,omitempty"`
	// NoMemoryFacts disables built-in memory fact gathering
	NoMemoryFacts bool `json:"no_memory_facts,omitempty"`
	// NoSwapFacts disables built-in swap facts gathering
	NoSwapFacts bool `json:"no_swap_facts,omitempty"`
	// NoCPUFacts disables built-in cpu facts gathering
	NoCPUFacts bool `json:"no_cpu_facts,omitempty"`
	// NoDiskFacts disables built-in disk facts gathering
	NoDiskFacts bool `json:"no_disk_facts,omitempty"`
	// NoHostFacts disables built-in host facts gathering
	NoHostFacts bool `json:"no_host_facts,omitempty"`
	// NoNetworkFacts disables built-in network interface facts gathering
	NoNetworkFacts bool `json:"no_network_facts,omitempty"`

	// ConfigurationDirectory is the directory the configuration file is stored in (RO)
	ConfigurationDirectory string `json:"configuration_directory"`
	// MachinesDirectory is where autonomous agents are stored (RO)
	MachinesDirectory string `json:"machines_directory"`
	// ProvisioningJWTFile is the path to provisioning jwt file, defaults to provisioning.jwt in the options dir (RO)
	ProvisioningJWTFile string `json:"provisioning_jwt_file"`
	// FactsFile is the path to the facts file which default to instance.json in the options dir (RO)
	FactsFile string `json:"facts_file"`
	// ServerSeedFile is the path to the server seed file that will exist after provisioning (RO)
	ServerSeedFile string `json:"server_seed_file"`
	// ServerJWTFile is the path to the server jwt file that will exist after provisioning (RO)
	ServerJWTFile string `json:"server_jwt_file"`
	// ServerStatusFile is where the server will regularly write its status (RO)
	ServerStatusFile string `json:"server_status_file"`
	// ServerSubmissionDirectory is the directory holding the submission spool (RO)
	ServerSubmissionDirectory string `json:"server_submission_directory"`
	// ServerSubmissionSpoolSize is the maximum size of the submission spool (RO)
	ServerSubmissionSpoolSize int `json:"server_submission_spool_size"`
	// CommandPath is the path to the command being run, defaults to argv[0] (RO)
	CommandPath string `json:"command_path"`
	// ServerStorageDirectory the directory where state is stored (RO)
	ServerStorageDirectory string `json:"server_storage_directory"`
	// NatsNeySeedFile is a path to a nkey seed created at start
	NatsNeySeedFile string `json:"nats_ney_seed_file"`
	// NatsCredentialsFile is a path to the nats credentials file holding data received during provisioning
	NatsCredentialsFile string `json:"nats_credentials_file"`
	// StartTime the time the process started (RO)
	StartTime time.Time `json:"start_time"`
}
