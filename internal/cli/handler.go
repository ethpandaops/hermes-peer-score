package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/config"
	"github.com/ethpandaops/hermes-peer-score/internal/core"
	"github.com/ethpandaops/hermes-peer-score/internal/reports"
)

// Handler manages CLI operations and command routing
type Handler struct {
	logger logrus.FieldLogger
}

// NewHandler creates a new CLI handler
func NewHandler(logger logrus.FieldLogger) *Handler {
	return &Handler{
		logger: logger.WithField("component", "cli_handler"),
	}
}

// Run executes the main application logic based on configuration
func (h *Handler) Run(cfg *config.DefaultConfig) error {
	h.logger.Info("Starting Hermes Peer Score Tool")
	
	// Handle different execution modes
	switch {
	case cfg.IsHTMLOnly():
		return h.handleHTMLOnlyMode(cfg)
	case cfg.IsUpdateGoMod():
		return h.handleGoModUpdate(cfg)
	case cfg.IsValidateGoMod():
		return h.handleGoModValidation(cfg)
	default:
		return h.handlePeerScoreTest(cfg)
	}
}

// handleHTMLOnlyMode generates HTML report from existing JSON file
func (h *Handler) handleHTMLOnlyMode(cfg *config.DefaultConfig) error {
	h.logger.Info("Running in HTML-only mode")
	
	inputFile := cfg.GetInputJSON()
	if inputFile == "" {
		return fmt.Errorf("input JSON file must be specified for HTML-only mode")
	}
	
	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input JSON file does not exist: %s", inputFile)
	}
	
	// Generate output filename
	outputFile := generateHTMLFilename(inputFile)
	
	h.logger.WithFields(logrus.Fields{
		"input":  inputFile,
		"output": outputFile,
	}).Info("Generating HTML report from JSON")
	
	// Create report generator
	reportGen, err := reports.NewGenerator(h.logger)
	if err != nil {
		return fmt.Errorf("failed to create report generator: %w", err)
	}
	
	// Get API key for AI analysis
	apiKey := cfg.GetClaudeAPIKey()
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	
	// Generate HTML report
	if apiKey != "" && !cfg.IsSkipAI() {
		h.logger.Info("Including AI analysis in HTML report")
		err = reportGen.GenerateHTMLFromJSONWithAI(inputFile, outputFile, apiKey)
	} else {
		h.logger.Info("Generating HTML report without AI analysis")
		err = reportGen.GenerateHTMLFromJSON(inputFile, outputFile)
	}
	
	if err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}
	
	h.logger.WithField("output", outputFile).Info("HTML report generated successfully")
	return nil
}

// handleGoModUpdate updates go.mod for the specified validation mode
func (h *Handler) handleGoModUpdate(cfg *config.DefaultConfig) error {
	h.logger.WithField("validation_mode", cfg.GetValidationMode()).Info("Updating go.mod")
	
	// TODO: Implement go.mod update logic
	// This would involve reading the current go.mod, updating the Hermes version
	// based on the validation mode, and writing it back
	
	h.logger.Info("Go.mod update functionality not yet implemented in refactored version")
	return nil
}

// handleGoModValidation validates go.mod for the specified validation mode
func (h *Handler) handleGoModValidation(cfg *config.DefaultConfig) error {
	h.logger.WithField("validation_mode", cfg.GetValidationMode()).Info("Validating go.mod")
	
	// TODO: Implement go.mod validation logic
	// This would involve checking if the current go.mod has the correct Hermes version
	// for the specified validation mode
	
	h.logger.Info("Go.mod validation functionality not yet implemented in refactored version")
	return nil
}

// handlePeerScoreTest runs the main peer scoring test
func (h *Handler) handlePeerScoreTest(cfg *config.DefaultConfig) error {
	h.logger.WithField("validation_mode", cfg.GetValidationMode()).Info("Starting peer score test")
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Set up graceful shutdown
	ctx, cancel := h.setupGracefulShutdown()
	defer cancel()
	
	// Create and configure the core tool
	tool, err := core.NewTool(ctx, cfg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to create peer score tool: %w", err)
	}
	
	// Log connection settings
	h.logConnectionSettings(cfg)
	
	// Start the tool
	if err := tool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start peer score tool: %w", err)
	}
	
	// Ensure cleanup
	defer func() {
		if err := tool.Stop(); err != nil {
			h.logger.WithError(err).Error("Error stopping tool")
		}
	}()
	
	// Save reports
	if err := tool.SaveReports(); err != nil {
		return fmt.Errorf("failed to save reports: %w", err)
	}
	
	h.logger.Info("Peer score test completed successfully")
	return nil
}

// setupGracefulShutdown configures signal handling for graceful shutdown
func (h *Handler) setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		h.logger.Info("Received shutdown signal")
		cancel()
	}()
	
	return ctx, cancel
}

// logConnectionSettings logs connection details with password redaction
func (h *Handler) logConnectionSettings(cfg *config.DefaultConfig) {
	h.logger.Info("Connection settings:")
	h.logger.WithFields(logrus.Fields{
		"prysm_host":  cfg.HostWithRedactedSecrets(),
		"http_port":   cfg.GetPrysmHTTPPort(),
		"grpc_port":   cfg.GetPrysmGRPCPort(),
		"tls_enabled": cfg.GetUseTLS(),
	}).Info("Prysm connection configured")
}

// generateHTMLFilename generates HTML filename from JSON filename
func generateHTMLFilename(jsonFile string) string {
	if len(jsonFile) > 5 && jsonFile[len(jsonFile)-5:] == ".json" {
		return jsonFile[:len(jsonFile)-5] + ".html"
	}
	return jsonFile + ".html"
}