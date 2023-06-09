package cli

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/fisk"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/nats-io/nats.go"
	"github.com/ripienaar/machine-room/internal/broker"
	"github.com/ripienaar/machine-room/internal/server"
	"github.com/ripienaar/machine-room/options"
	"github.com/sirupsen/logrus"
)

func (c *CLI) runCommand(pc *fisk.ParseContext) error {
	_, _, err := c.CommonConfigure()
	if err != nil {
		return err
	}

	var cfg *config.Config

	if choria.FileExist(c.cfgFile) {
		cfg, err = config.NewConfig(c.cfgFile)
		if err != nil {
			return err
		}

		c.isLeader = cfg.Option(options.ConfigKeyRole, "follower") == "leader"
	}

	c.log = c.log.WithFields(logrus.Fields{"leader": c.isLeader})

	c.log.Warnf("Starting %s version %s with config file %s", c.opts.Name, c.opts.Version, c.cfgFile)

	err = c.createServerNKey()
	if err != nil {
		c.log.Errorf("Could not create nkey: %v", err)
	}

	// makes sure we have some facts to send during provisioning
	err = c.factsCommand(pc)
	if err != nil {
		c.log.Errorf("Could not write initial facts: %v", err)
	}

	wg := sync.WaitGroup{}

	var inproc nats.InProcessConnProvider
	if c.isLeader {
		b, err := broker.New(c.opts, c.cfgFile, &build.Info{}, c.log)
		if err != nil {
			return err
		}

		err = b.Start(c.ctx, &wg)
		if err != nil {
			return err
		}

		inproc = b.InProcessConnProvider()

		err = b.StartReplication(c.ctx, &wg)
		if err != nil {
			return err
		}
	}

	err = c.startServer(c.ctx, &wg, inproc)
	if err != nil {
		return fmt.Errorf("machine room server failed: %v", err)
	}

	wg.Wait()

	return nil
}

func (c *CLI) startServer(ctx context.Context, wg *sync.WaitGroup, inproc nats.InProcessConnProvider) error {
	srv, err := server.New(c.opts, c.cfgFile, inproc, c.log)
	if err != nil {
		return err
	}

	err = srv.Start(ctx, wg)
	if err != nil {
		return err
	}

	return nil
}
