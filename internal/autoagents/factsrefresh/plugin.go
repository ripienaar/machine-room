package factsrefresh

import (
	"fmt"

	"github.com/choria-io/go-choria/aagent/machine"
	mp "github.com/choria-io/go-choria/aagent/plugin"
	"github.com/choria-io/go-choria/aagent/watchers"
	"github.com/choria-io/go-choria/plugin"
	"github.com/ripienaar/machine-room/options"
)

func Register(opts *options.Options, cfgFile string) error {
	if opts.CommandPath == "" {
		return fmt.Errorf("no command path set in options")
	}

	m := &machine.Machine{
		MachineName:    "facts_refresh",
		MachineVersion: opts.Version,
		InitialState:   "GATHER",
		Transitions: []*machine.Transition{
			{
				Name:        "MAINTENANCE",
				From:        []string{"GATHER"},
				Destination: "MAINTENANCE",
			},
			{
				Name:        "RESUME",
				From:        []string{"MAINTENANCE"},
				Destination: "GATHER",
			},
		},
		WatcherDefs: []*watchers.WatcherDef{
			{
				Name:       "update_facts",
				Type:       "exec",
				Interval:   opts.FactsRefreshInterval.String(),
				StateMatch: []string{"GATHER"},
				Properties: map[string]any{
					"command":              fmt.Sprintf("%s facts --config %s", opts.CommandPath, cfgFile),
					"timeout":              "1m",
					"gather_initial_state": "true",
				},
			},
		},
	}

	return plugin.Register("facts_refresh", mp.NewMachinePlugin("facts_refresh", m))
}
