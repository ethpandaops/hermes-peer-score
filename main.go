package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/constants"
	"github.com/ethpandaops/hermes-peer-score/internal/cli"
	"github.com/ethpandaops/hermes-peer-score/internal/config"
)

// Command-line flags
var (
	duration       = flag.Duration("duration", constants.DefaultTestDuration, "Test duration for peer scoring")
	prysmHost      = flag.String("prysm-host", "", "Prysm host connection string (required for both validation modes)")
	prysmHTTPPort  = flag.Int("prysm-http-port", constants.DefaultPrysmHTTPPort, "Prysm HTTP port")
	prysmGRPCPort  = flag.Int("prysm-grpc-port", constants.DefaultPrysmGRPCPort, "Prysm gRPC port")
	validationMode = flag.String("validation-mode", string(config.ValidationModeDelegated), "Validation mode: 'delegated' (delegates validation to Prysm) or 'independent' (uses Prysm for beacon data, validates internally)")
	htmlOnly       = flag.Bool("html-only", false, "Generate HTML report from existing JSON file without running peer score test")
	inputJSON      = flag.String("input-json", constants.DefaultJSONReportFile, "Input JSON file for HTML-only mode")
	claudeAPIKey   = flag.String("openrouter-api-key", "", "OpenRouter API key for AI analysis (can also be set via OPENROUTER_API_KEY env var)")
	skipAI         = flag.Bool("skip-ai", false, "Skip AI analysis even if API key is available")
	updateGoMod    = flag.Bool("update-go-mod", false, "Update go.mod for the specified validation mode and exit")
	validateGoMod  = flag.Bool("validate-go-mod", false, "Validate go.mod configuration for the specified validation mode and exit")
)

func main() {
	flag.Parse()
	
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	
	// Create configuration from flags
	cfg, err := createConfigFromFlags(logger)
	if err != nil {
		logger.Fatalf("Configuration error: %v", err)
	}
	
	// Create CLI handler
	cliHandler := cli.NewHandler(logger)
	
	// Run the application
	if err := cliHandler.Run(cfg); err != nil {
		logger.Fatalf("Application error: %v", err)
	}
}

// createConfigFromFlags creates configuration from command-line flags
func createConfigFromFlags(logger logrus.FieldLogger) (*config.DefaultConfig, error) {
	cfg := config.NewDefaultConfig()
	
	// Parse and validate validation mode
	validationModeValue, err := parseValidationMode(*validationMode)
	if err != nil {
		return nil, err
	}
	
	// Set configuration values from flags
	cfg.SetValidationMode(validationModeValue)
	cfg.SetTestDuration(*duration)
	cfg.SetPrysmHost(*prysmHost)
	cfg.SetPrysmHTTPPort(*prysmHTTPPort)
	cfg.SetPrysmGRPCPort(*prysmGRPCPort)
	cfg.SetHTMLOnly(*htmlOnly)
	cfg.SetInputJSON(*inputJSON)
	cfg.SetSkipAI(*skipAI)
	cfg.SetUpdateGoMod(*updateGoMod)
	cfg.SetValidateGoMod(*validateGoMod)
	
	// Get API key from flag or environment
	apiKey := *claudeAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	cfg.SetClaudeAPIKey(apiKey)
	
	logger.WithFields(logrus.Fields{
		"validation_mode": cfg.GetValidationMode(),
		"test_duration":   cfg.GetTestDuration(),
		"html_only":       cfg.IsHTMLOnly(),
		"prysm_host":      cfg.HostWithRedactedSecrets(),
	}).Info("Configuration loaded")
	
	return cfg, nil
}

// parseValidationMode parses and validates the validation mode string
func parseValidationMode(mode string) (config.ValidationMode, error) {
	switch mode {
	case string(config.ValidationModeDelegated):
		return config.ValidationModeDelegated, nil
	case string(config.ValidationModeIndependent):
		return config.ValidationModeIndependent, nil
	default:
		return "", fmt.Errorf(constants.ErrInvalidValidationMode)
	}
}