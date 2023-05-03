package cli

import (
	"github.com/choria-io/fisk"
	"github.com/ripienaar/machine-room/internal/server"
)

func (c *CLI) factsCommand(_ *fisk.ParseContext) error {
	_, log, err := c.CommonConfigure()
	if err != nil {
		return err
	}

	return server.SaveFacts(c.ctx, *c.opts, log)
}
