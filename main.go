package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/portwatch/config"
	"github.com/user/portwatch/monitor"
)

const version = "0.1.0"

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "portwatch.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		log.Printf("portwatch v%s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", *configPath, err)
	}

	log.Printf("portwatch v%s starting (interval: %ds)", version, cfg.Interval)

	// Initialize the port monitor
	m, err := monitor.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize monitor: %v", err)
	}

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring in a goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := m.Run(); err != nil {
			errCh <- err
		}
	}()

	// Block until signal or fatal error
	select {
	case sig := <-sigCh:
		log.Printf("Received signal %s, shutting down...", sig)
		m.Stop()
	case err := <-errCh:
		log.Fatalf("Monitor error: %v", err)
	}

	log.Println("portwatch stopped.")
}
