package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Configuration and command-line flags.
var (
	duration       = flag.Duration("duration", 2*time.Minute, "Test duration for peer scoring")
	prysmHost      = flag.String("prysm-host", "", "Prysm host connection string (required for both validation modes)")
	prysmHTTPPort  = flag.Int("prysm-http-port", 443, "Prysm HTTP port")
	prysmGRPCPort  = flag.Int("prysm-grpc-port", 443, "Prysm gRPC port")
	validationMode = flag.String("validation-mode", "delegated", "Validation mode: 'delegated' (delegates validation to Prysm) or 'independent' (uses Prysm for beacon data, validates internally)")
	htmlOnly       = flag.Bool("html-only", false, "Generate HTML report from existing JSON file without running peer score test")
	inputJSON      = flag.String("input-json", "peer-score-report.json", "Input JSON file for HTML-only mode")
	claudeAPIKey   = flag.String("openrouter-api-key", "", "OpenRouter API key for AI analysis (can also be set via OPENROUTER_API_KEY env var)")
	skipAI         = flag.Bool("skip-ai", false, "Skip AI analysis even if API key is available")
	updateGoMod    = flag.Bool("update-go-mod", false, "Update go.mod for the specified validation mode and exit")
	validateGoMod  = flag.Bool("validate-go-mod", false, "Validate go.mod configuration for the specified validation mode and exit")
)

func main() {
	flag.Parse()

	// Initialise logger.
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// If HTML-only mode is enabled, just generate HTML from existing JSON
	if *htmlOnly {
		generateHTMLOnlyMode(log)

		return
	}

	// Validate and parse validation mode
	validationModeValue, err := ValidateValidationMode(*validationMode)
	if err != nil {
		log.Fatalf("Invalid validation mode: %v", err)
	}

	// Handle go.mod management flags
	if *updateGoMod {
		log.Infof("Updating go.mod for validation mode: %s", validationModeValue)

		if err := UpdateGoModForValidationMode(validationModeValue); err != nil {
			log.Fatalf("Failed to update go.mod: %v", err)
		}

		log.Info("Go.mod updated successfully")

		return
	}

	if *validateGoMod {
		log.Infof("Validating go.mod for validation mode: %s", validationModeValue)

		if err := ValidateGoModForValidationMode(validationModeValue); err != nil {
			log.Fatalf("Go.mod validation failed: %v", err)
		}

		log.Info("Go.mod validation passed")

		return
	}

	log.Infof("Using validation mode: %s", validationModeValue)

	// Validate and optionally update go.mod for the validation mode
	if err := validateOrUpdateGoMod(log, validationModeValue); err != nil {
		log.Warnf("Go module validation issue: %v", err)
	}

	// Set up graceful shutdown handling.
	ctx, cancel := setupGracefulShutdown(log)
	defer cancel()

	// Intialise tool config.
	cfg := buildToolConfig()
	if err := cfg.Validate(validationModeValue); err != nil {
		log.Fatal(err)
	}

	// Initialize peer score tool configuration.
	tool := NewPeerScoreTool(ctx, log, PeerScoreConfig{
		ToolConfig:     cfg,
		ValidationMode: validationModeValue,
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

// generateHTMLOnlyMode generates HTML report from existing JSON file without running the peer scoring test.
func generateHTMLOnlyMode(log logrus.FieldLogger) {
	log.Info("Running in HTML-only mode")

	// Check if input JSON file exists
	if _, err := os.Stat(*inputJSON); os.IsNotExist(err) {
		log.Fatalf("Input JSON file does not exist: %s", *inputJSON)
	}

	// Determine output HTML file name based on input JSON
	htmlFile := strings.Replace(*inputJSON, ".json", ".html", 1)

	log.Infof("Generating HTML report from: %s", *inputJSON)
	log.Infof("Output HTML file: %s", htmlFile)

	// Get API key for AI analysis (optional)
	apiKey := *claudeAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	// Generate HTML report with optional AI analysis
	if apiKey != "" {
		log.Info("OpenRouter API key found - including AI analysis in HTML report")

		if err := GenerateHTMLReportWithAI(log, *inputJSON, htmlFile, apiKey, ""); err != nil {
			log.Fatalf("Failed to generate HTML report with AI analysis: %v", err)
		}
	} else {
		log.Info("No OpenRouter API key found - generating HTML report without AI analysis")
		log.Info("To include AI analysis, set OPENROUTER_API_KEY environment variable or use -openrouter-api-key flag")

		if err := GenerateHTMLReport(log, *inputJSON, htmlFile); err != nil {
			log.Fatalf("Failed to generate HTML report: %v", err)
		}
	}

	log.Infof("HTML report generated successfully: %s", htmlFile)
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

// validateOrUpdateGoMod checks if go.mod is configured correctly for the validation mode.
func validateOrUpdateGoMod(log logrus.FieldLogger, validationMode ValidationMode) error {
	// First, check if go.mod is already correctly configured
	if err := ValidateGoModForValidationMode(validationMode); err == nil {
		log.Infof("Go module is correctly configured for %s validation mode", validationMode)

		return nil
	}

	// If validation failed, log the current version
	currentVersion, _ := GetCurrentHermesVersion()
	expectedConfig := GetValidationConfigs()[validationMode]

	log.WithFields(logrus.Fields{
		"validation_mode":  validationMode,
		"current_version":  currentVersion,
		"expected_version": expectedConfig.HermesVersion,
	}).Info("Go module needs update for validation mode")

	// Provide guidance.
	log.Warnf("To use %s validation mode, update go.mod with:", validationMode)

	switch validationMode {
	case ValidationModeDelegated:
		log.Warn("replace github.com/probe-lab/hermes => github.com/ethpandaops/hermes v0.0.4-0.20250513093811-320c1c3ee6e2")
	case ValidationModeIndependent:
		log.Warn("replace github.com/probe-lab/hermes => github.com/ethpandaops/hermes v0.0.4-0.20250611021139-b3e6fc7d4d79")
	}

	log.Warn("Then run: go mod tidy")

	return nil
}
