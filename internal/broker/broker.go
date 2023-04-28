package broker

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/broker/adapter"
	"github.com/choria-io/go-choria/broker/network"
	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Broker struct {
	cfg    *config.Config
	bi     *build.Info
	fw     *choria.Framework
	log    *logrus.Entry
	broker *network.Server
}

func New(configFile string, bi *build.Info, log *logrus.Entry) (*Broker, error) {
	if configFile == "" {
		return nil, fmt.Errorf("configuration file is required")
	}

	var err error

	broker := &Broker{
		bi:  bi,
		log: log.WithField("machine_room", "broker"),
	}

	broker.cfg, err = config.NewSystemConfig(configFile, true)
	if err != nil {
		return nil, fmt.Errorf("could not parse configuration: %s", err)
	}

	broker.cfg.Choria.BrokerNetwork = true
	broker.cfg.CustomLogger = broker.log.Logger
	broker.cfg.DisableTLSVerify = false
	broker.cfg.Choria.ServerAnonTLS = false
	broker.cfg.Choria.UseSRVRecords = false
	broker.cfg.Choria.NetworkClientPort = 9222
	broker.cfg.Choria.NetworkSystemUsername = "system"

	// disable some broker things, we dont want people to edit the config file and enable stuff
	broker.cfg.Choria.NetworkMappings = []string{}
	broker.cfg.Choria.NetworkDenyServers = false
	broker.cfg.Choria.NetworkClientTLSForce = false
	broker.cfg.Choria.NetworkAllowedClientHosts = []string{}
	broker.cfg.Choria.NetworkGatewayPort = 0
	broker.cfg.Choria.NetworkLeafPort = 0
	broker.cfg.Choria.NetworkWebSocketPort = 0
	broker.cfg.Choria.NetworkPeerPort = 0
	broker.cfg.Choria.BrokerAdapters = []string{}

	// forcing here disables the delay in stream creation at first start
	broker.cfg.Choria.NetworkEventStoreReplicas = 1
	broker.cfg.Choria.NetworkLeaderElectionReplicas = 1
	broker.cfg.Choria.NetworkMachineStoreReplicas = 1
	broker.cfg.Choria.NetworkStreamAdvisoryReplicas = 1

	// always be running jetstream
	broker.cfg.Choria.NetworkStreamStore = "/var/lib/choria/machine-room"

	err = broker.saveCert()
	if err != nil {
		return nil, fmt.Errorf("TLS setup failed: %v", err)
	}

	broker.fw, err = choria.NewWithConfig(broker.cfg)
	if err != nil {
		return nil, err
	}

	broker.cfg.Choria.BrokerAdapters = []string{"registration"}
	mw, err := broker.fw.MiddlewareServers()
	if err == nil {
		broker.cfg.SetOption("plugin.choria.adapter.registration.stream.servers", strings.Join(mw.Strings(), ","))
		broker.cfg.SetOption("plugin.choria.adapter.registration.type", "choria_streams")
		broker.cfg.SetOption("plugin.choria.adapter.registration.stream.topic", "machine_room.nodes.%s")
		broker.cfg.SetOption("plugin.choria.adapter.registration.stream.workers", "3")
		broker.cfg.SetOption("plugin.choria.adapter.registration.ingest.topic", "choria.broadcast.agent.registration")
		broker.cfg.SetOption("plugin.choria.adapter.registration.ingest.protocol", "request")
		broker.cfg.SetOption("plugin.choria.adapter.registration.ingest.workers", "3")
	}

	return broker, nil
}

// creates a single use CA just enough to start the broker and allow local
// access, does not allow RPC using the cert thats created
//
// also does not store the CA key so no new certs can be made, remade at every
// start for the moment.
func (b *Broker) saveCert() error {
	// TODO: if these are valid keep them, eventually we might have more than one process using them so need to not rip it out under that

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"Synadia Inc."},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"Synadia Inc."},
			Country:      []string{"US"},
			CommonName:   b.cfg.Identity,
		},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		DNSNames:    []string{"localhost", b.cfg.Identity},
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	b.cfg.Choria.SecurityProvider = "choria"
	b.cfg.Choria.ChoriaSecuritySeedFile = filepath.Join(filepath.Dir(b.cfg.ConfigFile), "server.seed")
	b.cfg.Choria.ChoriaSecurityCA = filepath.Join(filepath.Dir(b.cfg.ConfigFile), "ca.pem")
	b.cfg.Choria.ChoriaSecurityCertificate = filepath.Join(filepath.Dir(b.cfg.ConfigFile), "cert.pem")
	b.cfg.Choria.ChoriaSecurityKey = filepath.Join(filepath.Dir(b.cfg.ConfigFile), "key.pem")

	err = os.WriteFile(b.cfg.Choria.ChoriaSecurityCA, caPEM.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("could not write certificate: %v", err)
	}

	err = os.WriteFile(b.cfg.Choria.ChoriaSecurityCertificate, certPEM.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("could not write certificate: %v", err)
	}

	err = os.WriteFile(b.cfg.Choria.ChoriaSecurityKey, certPrivKeyPEM.Bytes(), 0400)
	if err != nil {
		return fmt.Errorf("could not write certificate: %v", err)
	}

	return nil
}

func (b *Broker) InProcessConnProvider() nats.InProcessConnProvider {
	return b.broker
}

func (b *Broker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	b.log.Warnf("Choria Machine Room Broker version %s starting with config %s", b.bi.Version(), b.cfg.ConfigFile)
	broker, err := network.NewServer(b.fw, b.bi, b.log.Level == logrus.DebugLevel)
	if err != nil {
		return err
	}

	b.broker = broker
	b.fw.SetInProcessConnProvider(broker)

	wg.Add(1)
	go b.broker.Start(ctx, wg)

	for {
		b.log.Infof("Waiting for broker to be started")
		if broker.Started() {
			break
		}
		err = backoff.Default.Sleep(ctx, 500*time.Millisecond)
		if err != nil {
			return err
		}
	}

	go b.setupStreams(ctx)

	if len(b.cfg.Choria.BrokerAdapters) > 0 {
		b.log.Infof("Starting data adapters: %s", strings.Join(b.cfg.Choria.BrokerAdapters, ", "))
		err = adapter.RunAdapters(ctx, b.fw, wg)
		if err != nil {
			b.log.Errorf("Could not start adapters: %v", err)
		}
	}

	return nil
}

func (b *Broker) createDesiredStateBucket(ctx context.Context, nc *nats.Conn) error {
	js, err := nc.JetStream(nats.Context(ctx))
	if err != nil {
		return err
	}

	_, err = js.KeyValue("CONFIG")
	if err == nil {
		return nil
	}

	if errors.Is(err, nats.ErrBucketNotFound) {
		_, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: "CONFIG", History: 5, Storage: nats.FileStorage})
		if err != nil {
			return err
		}
		b.log.Infof("Creating CONFIG bucket")
	} else if err != nil {
		return err
	}

	return nil
}

func (b *Broker) createRegistrationStream(ctx context.Context, nc *nats.Conn) error {
	js, err := nc.JetStream(nats.Context(ctx))
	if err != nil {
		b.log.Errorf("Could not connect to Machine Room JetStream: %v", err)
		return err
	}

	_, err = js.StreamInfo("REGISTRATION")
	if errors.Is(err, nats.ErrStreamNotFound) {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:              "REGISTRATION",
			Subjects:          []string{"machine_room.nodes.>"},
			MaxAge:            24 * time.Hour,
			MaxMsgsPerSubject: 5,
			Storage:           nats.FileStorage,
		})
		if err != nil {
			return err
		}
		b.log.Infof("Created REGISTRATION stream")
	} else if err != nil {
		return err
	}

	return nil
}

func (b *Broker) createSubmitStream(ctx context.Context, nc *nats.Conn) error {
	js, err := nc.JetStream(nats.Context(ctx))
	if err != nil {
		b.log.Errorf("Could not connect to Machine Room JetStream: %v", err)
		return err
	}

	_, err = js.StreamInfo("SUBMIT")
	if errors.Is(err, nats.ErrStreamNotFound) {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     "SUBMIT",
			Subjects: []string{"choria.submission.in.>"},
			MaxAge:   24 * time.Hour,
			Storage:  nats.FileStorage,
		})
		if err != nil {
			return err
		}
		b.log.Infof("Created SUBMIT stream")
	} else if err != nil {
		return err
	}

	return nil
}

func (b *Broker) setupStreams(ctx context.Context) {
	b.log.Infof("Setting up Machine Room Streams")

	err := backoff.Default.For(ctx, func(try int) error {
		if try > 10 {
			b.log.Warnf("Machine Room Stream setup still failing after %d tries", try)
		}

		b.log.Infof("Attempting to set up Machine Room streams: try %d", try)

		conn, err := b.fw.NewConnector(ctx, b.fw.MiddlewareServers, "stream_setup", b.log)
		if err != nil {
			b.log.Errorf("Could not connect to Machine Room broker: %v", err)
			return err
		}

		nc := conn.Nats()

		err = b.createDesiredStateBucket(ctx, nc)
		if err != nil {
			b.log.Errorf("Could not create CONFIG bucket: %v", err)
			return err
		}

		err = b.createRegistrationStream(ctx, nc)
		if err != nil {
			b.log.Errorf("Could not create Registration stream: %v", err)
			return err
		}

		err = b.createSubmitStream(ctx, nc)
		if err != nil {
			b.log.Errorf("Could not create Registration stream: %v", err)
			return err
		}

		return nil
	})
	if err == nil {
		b.log.Infof("Machine Room Streams created")
	} else {
		b.log.Errorf("Could not set up Machine Room streams: %v", err)
	}
}
