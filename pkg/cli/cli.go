package cli

import (
	"context"
	"os"

	"github.com/choria-io/fisk"
	"github.com/ripienaar/machine-room/pkg/options"
	"github.com/sirupsen/logrus"
)

var (
	version     = "development"
	defaultName = "machine-room"
	defaultHelp = "Management Agent"
)

type CLI struct {
	opts *options.Options

	log *logrus.Entry
	cli *fisk.Application

	logfile  string
	loglevel string
	debug    bool
	cfgFile  string
	isLeader bool

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new command line instance based on options
func New(o options.Options) (*CLI, error) {
	app := &CLI{opts: &o}

	err := app.validateOptions()
	if err != nil {
		return nil, err
	}

	app.cli = app.newCli()

	return app, nil
}

// Application expose the command line framework allowing new commands to be added to it at compile time
func (c *CLI) Application() *fisk.Application {
	return c.cli
}

// Run parses and executes the command
func (c *CLI) Run(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	c.cli.MustParseWithUsage(os.Args[1:])

	return nil
}

func (c *CLI) newCli() *fisk.Application {
	cli := fisk.New(c.opts.Name, c.opts.Help)
	cli.Author(c.opts.Contact)
	cli.Version(c.opts.Version)
	cli.HelpFlag.Short('h')

	cli.Flag("debug", "Enables debug logging").Default("false").UnNegatableBoolVar(&c.debug)

	run := cli.Commandf("run", "Runs the management agent").Action(c.runCommand)
	run.Flag("config", "Configuration file to use").Required().StringVar(&c.cfgFile)

	// generates and saves facts, will be called from auto agents to
	// update facts on a schedule hidden as its basically a private api
	facts := cli.Commandf("facts", "Save facts about this node to a file").Action(c.factsCommand).Hidden()
	facts.Flag("config", "Configuration file to use").Required().StringVar(&c.cfgFile)

	cli.Commandf("buildinfo", "Shows build information").Action(c.buildInfoCommand).Hidden()

	return cli
}
