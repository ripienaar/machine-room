package broker

import (
	"context"
	"fmt"
	"sync"

	srcfg "github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/replicator"
	"github.com/ripienaar/machine-room/options"
)

// StartReplication starts to replicate our standard streams and buckets
func (b *Broker) StartReplication(ctx context.Context, wg *sync.WaitGroup) error {
	// TODO: support nats creds sent via provisioning into the config file

	b.log.Infof("Starting data replication")

	backendUrl := b.cfg.Option(options.ConfigKeySourceHost, "")
	if backendUrl == "" {
		fmt.Printf("\n%#v\n", b.cfg)
		return fmt.Errorf("replication source is not defined")
	}

	site := b.cfg.Option(options.ConfigKeySite, "")
	if site == "" {
		return fmt.Errorf("site is not defined")
	}

	rcfg := &srcfg.Config{
		ReplicatorName: site,
		StateDirectory: "/var/lib/choria/machine-room/replicator",
	}

	cc := &srcfg.ChoriaConnection{
		SeedFileName:   b.cfg.Choria.ChoriaSecuritySeedFile,
		JWTFileName:    b.cfg.Choria.ChoriaSecurityTokenFile,
		CollectiveName: "choria",
	}

	rcfg.Streams = []*srcfg.Stream{
		{
			Name:             "REGISTRATION",
			Stream:           "REGISTRATION",
			TargetStream:     "MACHINE_ROOM_NODES",
			TargetURL:        backendUrl,
			NoTargetCreate:   true,
			SourceURL:        "nats://localhost:9222",
			SourceProcess:    b.broker,
			SourceChoriaConn: cc,
		},
		{
			Name:               "SUBMIT",
			Stream:             "SUBMIT",
			TargetStream:       "MACHINE_ROOM_EVENTS",
			TargetURL:          backendUrl,
			NoTargetCreate:     true,
			SourceURL:          "nats://localhost:9222",
			SourceProcess:      b.broker,
			SourceChoriaConn:   cc,
			TargetRemoveString: "choria.submission.in.",
			TargetPrefix:       "machine_room.submit.",
		},
		{
			Name:               "CHORIA_EVENTS",
			Stream:             "CHORIA_EVENTS",
			TargetStream:       "MACHINE_ROOM_EVENTS",
			TargetURL:          backendUrl,
			NoTargetCreate:     true,
			SourceURL:          "nats://localhost:9222",
			SourceProcess:      b.broker,
			SourceChoriaConn:   cc,
			TargetRemoveString: "choria.lifecycle.",
			TargetPrefix:       "machine_room.events.lifecycle.",
		},
		{
			Name:               "CHORIA_MACHINE",
			Stream:             "CHORIA_MACHINE",
			TargetStream:       "MACHINE_ROOM_EVENTS",
			TargetURL:          backendUrl,
			NoTargetCreate:     true,
			SourceURL:          "nats://localhost:9222",
			SourceProcess:      b.broker,
			SourceChoriaConn:   cc,
			TargetRemoveString: "choria.machine.",
			TargetPrefix:       "machine_room.events.machine.",
		},
		{
			Name:             "KV_CONFIG",
			Stream:           "KV_CONFIG",
			TargetStream:     "KV_CONFIG",
			TargetURL:        "nats://localhost:9222",
			TargetProcess:    b.broker,
			TargetChoriaConn: cc,
			NoTargetCreate:   true,
			Ephemeral:        true, // copy the entire thing each time we start to be sure we have the latest config
			SourceURL:        backendUrl,
		},
	}

	err := rcfg.Validate()
	if err != nil {
		return err
	}

	for _, s := range rcfg.Streams {
		b.log.Debugf("Configuring replication for stream stream %s", s.Name)
		stream, err := replicator.NewStream(s, rcfg, b.log.WithField("stream", s.Name))
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(s *srcfg.Stream) {
			defer wg.Done()

			wg.Add(1)
			err = stream.Run(ctx, wg)
			if err != nil {
				b.log.Errorf("Could not start replicator for %s: %v", s.Name, err)
			}
		}(s)
	}

	return nil
}
