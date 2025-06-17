package config

import (
	"time"

	"github.com/probe-lab/hermes/eth"
)

// ValidationMode represents the type of validation approach used by Hermes.
type ValidationMode string

const (
	ValidationModeDelegated   ValidationMode = "delegated"   // Delegates validation processing to Prysm
	ValidationModeIndependent ValidationMode = "independent" // Uses Prysm for beacon state but validates internally
)

// Config defines the interface for tool configuration.
type Config interface {
	GetValidationMode() ValidationMode
	GetTestDuration() time.Duration
	GetReportInterval() time.Duration
	GetPrysmHost() string
	GetPrysmHTTPPort() int
	GetPrysmGRPCPort() int
	GetUseTLS() bool
	GetMaxPeers() int
	GetDialConcurrency() int
	AsHermesConfig() *eth.NodeConfig
	Validate() error
	HostWithRedactedSecrets() string

	// Report configuration
	IsHTMLOnly() bool
	GetInputJSON() string
	GetClaudeAPIKey() string
	IsSkipAI() bool
	IsUpdateGoMod() bool
	IsValidateGoMod() bool
}

// Validator defines the interface for configuration validation.
type Validator interface {
	ValidateValidationMode(mode string) (ValidationMode, error)
	ValidateConfig(config Config) error
}

// Manager defines the interface for configuration management.
type Manager interface {
	LoadFromFlags() (Config, error)
	GetValidationConfig(mode ValidationMode) ValidationConfig
	GenerateTimestampedFilename(mode ValidationMode, base string, timestamp time.Time) string
}

// ValidationConfig holds configuration specific to a validation mode.
type ValidationConfig struct {
	Mode            ValidationMode         `json:"mode"`
	HermesVersion   string                 `json:"hermes_version"`
	ConfigOverrides map[string]interface{} `json:"config_overrides"`
}
