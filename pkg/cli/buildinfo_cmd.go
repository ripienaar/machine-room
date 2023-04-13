package cli

import (
	"encoding/json"
	"fmt"

	"github.com/choria-io/fisk"
	"github.com/choria-io/go-choria/build"
)

func (c *CLI) buildInfoCommand(_ *fisk.ParseContext) error {
	bi := build.Info{}

	nfo := map[string]any{
		"providers": map[string]any{
			"agent":    bi.AgentProviders(),
			"watchers": bi.MachineWatchers(),
			"data":     bi.DataProviders(),
		},
	}

	j, err := json.MarshalIndent(&nfo, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))

	return nil
}
