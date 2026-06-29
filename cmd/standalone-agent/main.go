package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/naviNBRuas/APA/pkg/agent"
)

const defaultConfigPath = "/etc/apa/config.yaml"

// Overridden at build time via -ldflags=-X main.version=<tag>
var version = "2.0.0-standalone"

func main() {
	configPath := flag.String("config", defaultConfigPath, "Path to agent config YAML")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("APA Standalone Agent v%s\n", version)
		fmt.Printf("Go %s | %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}

	rt, err := agent.NewRuntime(*configPath, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize agent: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	rt.Start(ctx, cancel)
}

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
}
