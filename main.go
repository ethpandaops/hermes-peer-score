package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Configuration and command-line flags.
var (
	duration      = flag.Duration("duration", 2*time.Minute, "Test duration for peer scoring")
	outputFile    = flag.String("output", "peer-score-report.json", "Output file for results")
	prysmHost     = flag.String("prysm-host", "", "Prysm host connection string (required)")
	prysmHTTPPort = flag.Int("prysm-http-port", 443, "Prysm HTTP port")
	prysmGRPCPort = flag.Int("prysm-grpc-port", 443, "Prysm gRPC port")
)

func main() {
	flag.Parse()

	// Initialise logger.
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set up graceful shutdown handling.
	ctx, cancel := setupGracefulShutdown(log)
	defer cancel()

	// Intialise tool config.
	cfg := buildToolConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	// Initialize peer score tool configuration.
	tool := NewPeerScoreTool(ctx, log, PeerScoreConfig{
		ToolConfig:     cfg,
		TestDuration:   *duration,
		ReportInterval: 2 * time.Minute,
	})

	// Log connection settings for debugging.
	logConnectionSettings(ctx, log, tool)

	// Execute the peer scoring test.
	runPeerScoreTest(ctx, log, tool)

	// Generate and save reports.
	generateReports(ctx, log, tool)
}

// setupGracefulShutdown configures signal handling for graceful shutdown.
func setupGracefulShutdown(log logrus.FieldLogger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal")
		cancel()
	}()

	return ctx, cancel
}

// runPeerScoreTest executes the main peer scoring test.
func runPeerScoreTest(ctx context.Context, log logrus.FieldLogger, tool *PeerScoreTool) {
	// Start the Hermes process.
	if err := tool.StartHermes(ctx); err != nil {
		log.Fatalf("Failed to start Hermes: %v", err)
	}

	defer func() {
		if err := tool.Stop(); err != nil {
			log.Printf("Error stopping tool: %v", err)
		}
	}()

	log.Infof("Running peer score tests for %v...", *duration)

	// Start periodic status reporting.
	go startStatusReporting(ctx, log, tool)

	// Wait for test completion or cancellation.
	select {
	case <-ctx.Done():
		log.Println("Test interrupted")
	case <-time.After(*duration):
		log.Println("Test duration completed")
	}
}

// startStatusReporting provides periodic updates on peer connection status.
func startStatusReporting(ctx context.Context, log logrus.FieldLogger, tool *PeerScoreTool) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logCurrentStatus(ctx, log, tool)
		}
	}
}

// logConnectionSettings logs connection details with password redaction for security.
func logConnectionSettings(_ context.Context, log logrus.FieldLogger, tool *PeerScoreTool) {
	cfg := tool.config.ToolConfig

	log.Info("Connection settings:")
	log.Infof("  Prysm Host: %s", cfg.HostWithRedactedSecrets())
	log.Infof("  HTTP Port: %d", cfg.PrysmPortHTTP)
	log.Infof("  gRPC Port: %d", cfg.PrysmPortGRPC)
	log.Infof("  TLS Enabled: %t", cfg.PrysmUseTLS)
}

// logCurrentStatus logs the current peer connection statistics.
func logCurrentStatus(_ context.Context, log logrus.FieldLogger, tool *PeerScoreTool) {
	tool.peersMu.RLock()
	defer tool.peersMu.RUnlock()

	peerCount := len(tool.peers)
	identified := 0

	for _, peer := range tool.peers {
		// Check if any session has been identified
		for _, session := range peer.ConnectionSessions {
			if session.IdentifiedAt != nil {
				identified++
				break // Count each peer only once
			}
		}
	}

	log.WithFields(logrus.Fields{
		"peer_count":             peerCount,
		"identified_peers_count": identified,
	}).Infof("Status report")
}
