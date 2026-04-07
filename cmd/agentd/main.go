package main

import (
	"context"
	"flag"
	"log"

	"github.com/naviNBRuas/APA/pkg/agent"
	"github.com/naviNBRuas/APA/pkg/update"
)

// version is the current version of the agent. It should be set at build time.
var version = "v0.1.0" // Default version

func main() {
	// Define a flag for the config file path.
	configPath := flag.String("config", "configs/agent-config.yaml", "Path to the agent configuration file.")
	flag.Parse()

	// At the very start of the program, check for and apply any pending updates.
	// If an update is applied, this function will cause the process to restart.
	update.ApplyPendingUpdate()

	// Create and start the agent runtime
	runtime, err := agent.NewRuntime(*configPath, version)
	if err != nil {
		log.Fatalf("Failed to create agent runtime: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtime.Start(ctx, cancel)
}
