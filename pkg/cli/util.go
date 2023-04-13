package cli

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/ripienaar/machine-room/pkg/options"
	"github.com/sirupsen/logrus"
)

// CommonConfigure parses the configuration file, prepares logging etc and should be called early in any action
func (c *CLI) CommonConfigure() (*options.Options, *logrus.Entry, error) {
	var err error

	c.cfgFile, err = filepath.Abs(c.cfgFile)
	if err != nil {
		return nil, nil, err
	}

	parent := filepath.Dir(c.cfgFile)
	if c.opts.ProvisioningJWTFile == "" {
		c.opts.ProvisioningJWTFile = filepath.Join(parent, "provisioning.jwt")
	}

	if c.opts.FactsFile == "" {
		c.opts.FactsFile = filepath.Join(parent, "instance.json")
	}

	build.ProvisionJWTFile = c.opts.ProvisioningJWTFile

	log := logrus.New()
	switch {
	case strings.ToLower(c.logfile) == "discard":
		log.SetOutput(io.Discard)

	case c.logfile != "":
		log.Formatter = &logrus.JSONFormatter{}

		file, err := os.OpenFile(c.logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return nil, nil, fmt.Errorf("could not set up logging: %s", err)
		}

		log.SetOutput(file)
	}
	c.log = logrus.NewEntry(log)

	switch c.loglevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	case "fatal":
		log.SetLevel(logrus.FatalLevel)
	default:
		log.SetLevel(logrus.WarnLevel)
	}

	if c.debug {
		log.SetLevel(logrus.DebugLevel)
	}

	go c.interruptWatcher()

	return c.opts, c.log, nil
}

func (c *CLI) validateOptions() error {
	if c.opts.Help == "" {
		c.opts.Help = defaultHelp
	}
	if c.opts.Version == "" {
		c.opts.Version = version
	}
	if c.opts.Name == "" {
		c.opts.Name = defaultName
	}
	if c.opts.FactsRefreshInterval < time.Minute {
		c.opts.FactsRefreshInterval = 10 * time.Minute
	}
	if c.opts.CommandPath == "" {
		c.opts.CommandPath = os.Args[0]
	}
	if c.opts.MachineSigningKey == "" {
		return fmt.Errorf("autonomous agent signing key is required")
	}
	pk, err := hex.DecodeString(c.opts.MachineSigningKey)
	if err != nil {
		return fmt.Errorf("invalid autonomous agent signing key: %v", err)
	}
	if len(pk) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid autonomous agent signing key: incorrect length")
	}

	return nil
}

func (c *CLI) forceQuit() {
	<-time.After(10 * time.Second)

	c.log.Errorf("Forcing shut-down after 10 second grace window")

	os.Exit(1)
}

func (c *CLI) interruptWatcher() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				go c.forceQuit()

				c.log.Warnf("Shutting down on interrupt")

				c.cancel()
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}