package cli

import (
	"encoding/json"
	"os"

	"github.com/choria-io/fisk"
	"github.com/ripienaar/machine-room/internal/facts"
)

func (c *CLI) factsCommand(_ *fisk.ParseContext) error {
	_, log, err := c.CommonConfigure()
	if err != nil {
		return err
	}

	data, err := facts.Generate(c.ctx, *c.opts, log)
	if err != nil {
		return err
	}

	j, err := json.Marshal(data)
	if err != nil {
		return err
	}

	log.Infof("Writing facts to %v", c.opts.FactsFile)

	return os.WriteFile(c.opts.FactsFile, j, 0600)
}
