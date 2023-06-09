package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ripienaar/machine-room"
	"github.com/ripienaar/machine-room/options"
	"github.com/sirupsen/logrus"
)

func main() {
	app, err := machineroom.New(options.Options{
		Name:    "nats-manager",
		Version: "0.0.1",
		Contact: "info@example.net",
		Help:    "NATS Manager",

		// The public key of the autonomous agent spec encoding key, see setup/agents/signer.*
		MachineSigningKey: "b217b9c7574ad807f653754b9722e8001399c5646235742204963522da5c3b82",

		// optional below...

		// how frequently facts get updated on disk, we do it quick here for testing
		FactsRefreshInterval: time.Minute,

		// too noisy
		NoCPUFacts: true,

		// Users can plug in custom facts in addition to standard facts
		AdditionalFacts: extraFacts,
	})
	panicIfError(err)

	panicIfError(app.Run(context.Background()))
}

func extraFacts(_ context.Context, _ options.Options, _ *logrus.Entry) (map[string]any, error) {
	return map[string]any{"extra": true}, nil
}

func panicIfError(err error) {
	if err == nil {
		return
	}
	fmt.Printf("PANIC: %v\n", err)
	os.Exit(1)
}
