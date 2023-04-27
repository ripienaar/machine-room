package cli

import (
	archivewatcher "github.com/choria-io/go-choria/aagent/watchers/archivewatcher"
	"github.com/choria-io/go-choria/aagent/watchers/execwatcher"
	"github.com/choria-io/go-choria/aagent/watchers/filewatcher"
	"github.com/choria-io/go-choria/aagent/watchers/kvwatcher"
	"github.com/choria-io/go-choria/aagent/watchers/nagioswatcher"
	"github.com/choria-io/go-choria/aagent/watchers/pluginswatcher"
	"github.com/choria-io/go-choria/aagent/watchers/schedulewatcher"
	"github.com/choria-io/go-choria/aagent/watchers/timerwatcher"
	"github.com/choria-io/go-choria/plugin"
	golangrpc "github.com/choria-io/go-choria/providers/agent/mcorpc/golang"
	provisioner "github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/ripienaar/machine-room/internal/autoagents/machinesmanager"
)

func init() {
	// TODO: do this after start allowing options to add/remove some

	err := plugin.Register("choria_provision", provisioner.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("golangmco", golangrpc.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("archive_watcher", archivewatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("exec_watcher", execwatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("file_watcher", filewatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("kv_watcher", kvwatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("nagios_watcher", nagioswatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("plugins_watcher", pluginswatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("schedule_watcher", schedulewatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("timer_watcher", timerwatcher.ChoriaPlugin())
	if err != nil {
		panic(err)
	}

	err = plugin.Register("plugins_manager_machine", machines_manager.ChoriaPlugin())
	if err != nil {
		panic(err)
	}
}
