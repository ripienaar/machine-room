package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/providers/provtarget"
	"github.com/choria-io/go-choria/server"
	"github.com/nats-io/nats.go"
	"github.com/ripienaar/machine-room/pkg/internal/autoagents/factsrefresh"
	"github.com/ripienaar/machine-room/pkg/options"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg    *config.Config
	bi     *build.Info
	fw     *choria.Framework
	inproc nats.InProcessConnProvider
	log    *logrus.Entry
}

func New(opts *options.Options, configFile string, inproc nats.InProcessConnProvider, log *logrus.Entry) (*Server, error) {
	if configFile == "" {
		return nil, fmt.Errorf("configuration file is required")
	}

	var err error
	srv := &Server{
		bi:  &build.Info{},
		log: log.WithField("machine_room", "server"),
	}

	//srv.log.Logger.SetLevel(logrus.DebugLevel)

	srv.bi.SetProvisionJWTFile(opts.ProvisioningJWTFile)
	srv.bi.SetProvisionUsingVersion2(false)
	srv.bi.EnableProvisionModeAsDefault()
	build.Version = opts.Version // TODO: wrap in bi

	switch {
	case choria.FileExist(configFile):
		srv.cfg, err = config.NewSystemConfig(configFile, true)
		if err != nil {
			return nil, fmt.Errorf("could not parse configuration: %s", err)
		}
		srv.cfg.CustomLogger = srv.log.Logger

		if srv.shouldProvision() {
			provtarget.Configure(context.Background(), srv.cfg, srv.log.WithField("component", "provtarget"))

			log.Warnf("Switching to provisioning configuration due to build defaults and configuration settings")
			srv.cfg, err = srv.provisionConfig(configFile, srv.bi)
			if err != nil {
				return nil, err
			}
		} else {
			cfgDir := filepath.Dir(configFile)
			// auto agents are always on
			srv.cfg.Choria.MachineSourceDir = filepath.Join(cfgDir, "machine")
			srv.cfg.Choria.MachinesSignerPublicKey = opts.MachineSigningKey

			// standard status file always
			srv.cfg.Choria.StatusFilePath = "/var/lib/choria/machine-room/status.json"

			// message submit for auto agents etc
			srv.cfg.Choria.SubmissionSpoolMaxSize = 5000
			srv.cfg.Choria.SubmissionSpool = "/var/lib/choria/machine-room/submission"

			// some settings we need to not forget in provisioning helper
			srv.cfg.Choria.UseSRVRecords = false
			srv.cfg.RegisterInterval = 300
			srv.cfg.RegistrationSplay = true
			srv.cfg.FactSourceFile = opts.FactsFile
			srv.cfg.Choria.InventoryContentRegistrationTarget = "choria.broadcast.agent.registration"
			srv.cfg.Registration = []string{"inventory_content"}
			srv.cfg.Collectives = []string{"choria"}
			srv.cfg.MainCollective = "choria"

			srv.cfg.Choria.SecurityProvider = "choria"
			srv.cfg.Choria.ChoriaSecurityTokenFile = filepath.Join(cfgDir, "server.jwt")
			srv.cfg.Choria.ChoriaSecuritySeedFile = filepath.Join(cfgDir, "server.seed")

			os.MkdirAll(srv.cfg.Choria.MachineSourceDir, 0700)

			err = factsrefresh.Register(opts, configFile)
			if err != nil {
				srv.log.Errorf("Could not register facts refresh autonomous agent: %v", err)
			}
		}

	default:
		srv.cfg, err = srv.provisionConfig(configFile, srv.bi)
		if err != nil {
			return nil, err
		}
		srv.cfg.CustomLogger = srv.log.Logger
		provtarget.Configure(context.Background(), srv.cfg, srv.log.WithField("component", "provtarget"))

		log.Warnf("Switching to provisioning configuration due to build defaults and missing %s", configFile)
	}

	srv.cfg.ApplyBuildSettings(srv.bi)

	srv.fw, err = choria.NewWithConfig(srv.cfg)
	if err != nil {
		return nil, err
	}

	if inproc != nil {
		srv.fw.SetInProcessConnProvider(inproc)
	}

	return srv, nil
}

func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) error {
	s.fw.ConfigureProvisioning(ctx)
	instance, err := server.NewInstance(s.fw)
	if err != nil {
		return fmt.Errorf("could not create Choria Machine Room Server instance: %s", err)
	}

	wg.Add(1)
	go func() {
		err := instance.Run(ctx, wg)
		if err != nil {
			s.log.Errorf("Server instance failed to start: %v", err)
		}
	}()

	return nil
}

func (s *Server) IsProvisioning() bool {
	return s.fw.ProvisionMode()
}

func (s *Server) shouldProvision() bool {
	should := true
	if s.cfg.HasOption("plugin.choria.server.provision") {
		should = s.cfg.Choria.Provision
	}

	return should
}

func (s *Server) provisionConfig(f string, bi *build.Info) (*config.Config, error) {
	if !choria.FileExist(bi.ProvisionJWTFile()) {
		return nil, fmt.Errorf("provisioming token not found in %s", bi.ProvisionJWTFile())
	}

	cfg, err := config.NewDefaultSystemConfig(true)
	if err != nil {
		return nil, fmt.Errorf("could not create default configuration for provisioning: %s", err)
	}

	cfg.ConfigFile = f

	// set this to avoid calling into puppet on non puppet machines
	// later ConfigureProvisioning() will do all the right things
	cfg.Choria.SecurityProvider = "file"

	// in provision mode we do not yet have certs and stuff so we disable these checks
	cfg.DisableSecurityProviderVerify = true

	cfg.Choria.UseSRVRecords = false

	return cfg, nil
}
