package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"

	"github.com/ethpandaops/hermes-peer-score/constants"
)

// DefaultConfig implements the Config interface.
type DefaultConfig struct {
	// Tool configuration
	validationMode ValidationMode
	testDuration   time.Duration
	reportInterval time.Duration

	// Connection settings
	prysmHost     string
	prysmHTTPPort int
	prysmGRPCPort   int
	useTLS          bool
	network         string
	devnetApacheURL string

	// Networking settings
	privateKeyStr   string
	dialTimeout     time.Duration
	devp2pHost      string
	devp2pPort      int
	libp2pHost      string
	libp2pPort      int
	maxPeers        int
	dialConcurrency int

	// Data stream settings
	dataStreamType string
	subnets        map[string]*eth.SubnetConfig

	// Report settings
	htmlOnly      bool
	inputJSON     string
	claudeAPIKey  string
	skipAI        bool
	updateGoMod   bool
	validateGoMod bool
}

// NewDefaultConfig creates a new configuration with default values.
func NewDefaultConfig() *DefaultConfig {
	cfg := &DefaultConfig{
		validationMode:  ValidationModeDelegated,
		testDuration:    constants.DefaultTestDuration,
		reportInterval:  constants.DefaultReportInterval,
		prysmHTTPPort:   constants.DefaultPrysmHTTPPort,
		prysmGRPCPort:   constants.DefaultPrysmGRPCPort,
		network:         "mainnet",
		dialTimeout:     constants.DefaultDialTimeout,
		devp2pHost:      constants.DefaultDevp2pHost,
		libp2pHost:      constants.DefaultLibp2pHost,
		maxPeers:        constants.DefaultMaxPeers,
		dialConcurrency: constants.DefaultDialConcurrency,
		dataStreamType:  constants.DefaultDataStreamType,
		subnets:         make(map[string]*eth.SubnetConfig),
	}

	return cfg
}

// GetValidationMode returns the validation mode.
func (c *DefaultConfig) GetValidationMode() ValidationMode {
	return c.validationMode
}

// GetTestDuration returns the test duration.
func (c *DefaultConfig) GetTestDuration() time.Duration {
	return c.testDuration
}

// GetReportInterval returns the report interval.
func (c *DefaultConfig) GetReportInterval() time.Duration {
	return c.reportInterval
}

// GetPrysmHost returns the Prysm host.
func (c *DefaultConfig) GetPrysmHost() string {
	return c.prysmHost
}

// GetPrysmHTTPPort returns the Prysm HTTP port.
func (c *DefaultConfig) GetPrysmHTTPPort() int {
	return c.prysmHTTPPort
}

// GetPrysmGRPCPort returns the Prysm gRPC port.
func (c *DefaultConfig) GetPrysmGRPCPort() int {
	return c.prysmGRPCPort
}

// GetUseTLS returns whether TLS is enabled.
func (c *DefaultConfig) GetUseTLS() bool {
	return c.useTLS
}

// GetNetwork returns the Ethereum network.
func (c *DefaultConfig) GetNetwork() string {
	return c.network
}

// GetDevnetApacheURL returns the Apache URL for devnet configuration.
func (c *DefaultConfig) GetDevnetApacheURL() string {
	return c.devnetApacheURL
}

// GetMaxPeers returns the maximum number of peers.
func (c *DefaultConfig) GetMaxPeers() int {
	return c.maxPeers
}

// GetDialConcurrency returns the dial concurrency.
func (c *DefaultConfig) GetDialConcurrency() int {
	return c.dialConcurrency
}

// GetPrivateKeyStr returns the private key string.
func (c *DefaultConfig) GetPrivateKeyStr() string {
	return c.privateKeyStr
}

// GetDialTimeout returns the dial timeout.
func (c *DefaultConfig) GetDialTimeout() time.Duration {
	return c.dialTimeout
}

// GetDevp2pHost returns the devp2p host.
func (c *DefaultConfig) GetDevp2pHost() string {
	return c.devp2pHost
}

// GetDevp2pPort returns the devp2p port.
func (c *DefaultConfig) GetDevp2pPort() int {
	return c.devp2pPort
}

// GetLibp2pHost returns the libp2p host.
func (c *DefaultConfig) GetLibp2pHost() string {
	return c.libp2pHost
}

// GetLibp2pPort returns the libp2p port.
func (c *DefaultConfig) GetLibp2pPort() int {
	return c.libp2pPort
}

// GetDataStreamType returns the data stream type.
func (c *DefaultConfig) GetDataStreamType() string {
	return c.dataStreamType
}

// GetSubnets returns the subnet configurations.
func (c *DefaultConfig) GetSubnets() map[string]*eth.SubnetConfig {
	return c.subnets
}

// IsHTMLOnly returns whether HTML-only mode is enabled.
func (c *DefaultConfig) IsHTMLOnly() bool {
	return c.htmlOnly
}

// GetInputJSON returns the input JSON file path.
func (c *DefaultConfig) GetInputJSON() string {
	return c.inputJSON
}

// GetClaudeAPIKey returns the Claude API key.
func (c *DefaultConfig) GetClaudeAPIKey() string {
	return c.claudeAPIKey
}

// IsSkipAI returns whether AI analysis should be skipped.
func (c *DefaultConfig) IsSkipAI() bool {
	return c.skipAI
}

// IsUpdateGoMod returns whether go.mod should be updated.
func (c *DefaultConfig) IsUpdateGoMod() bool {
	return c.updateGoMod
}

// IsValidateGoMod returns whether go.mod should be validated.
func (c *DefaultConfig) IsValidateGoMod() bool {
	return c.validateGoMod
}

// SetValidationMode sets the validation mode.
func (c *DefaultConfig) SetValidationMode(mode ValidationMode) {
	c.validationMode = mode
}

// SetTestDuration sets the test duration.
func (c *DefaultConfig) SetTestDuration(duration time.Duration) {
	c.testDuration = duration
}

// SetPrysmHost sets the Prysm host.
func (c *DefaultConfig) SetPrysmHost(host string) {
	c.prysmHost = host
}

// SetPrysmHTTPPort sets the Prysm HTTP port.
func (c *DefaultConfig) SetPrysmHTTPPort(port int) {
	c.prysmHTTPPort = port
}

// SetPrysmGRPCPort sets the Prysm gRPC port.
func (c *DefaultConfig) SetPrysmGRPCPort(port int) {
	c.prysmGRPCPort = port
}

// SetUseTLS sets whether to use TLS for Prysm connections.
func (c *DefaultConfig) SetUseTLS(useTLS bool) {
	c.useTLS = useTLS
}

// SetNetwork sets the Ethereum network.
func (c *DefaultConfig) SetNetwork(network string) {
	c.network = network
}

// SetDevnetApacheURL sets the Apache URL for devnet configuration.
func (c *DefaultConfig) SetDevnetApacheURL(url string) {
	c.devnetApacheURL = url
}

// SetHTMLOnly sets HTML-only mode.
func (c *DefaultConfig) SetHTMLOnly(htmlOnly bool) {
	c.htmlOnly = htmlOnly
}

// SetInputJSON sets the input JSON file path.
func (c *DefaultConfig) SetInputJSON(inputJSON string) {
	c.inputJSON = inputJSON
}

// SetClaudeAPIKey sets the Claude API key.
func (c *DefaultConfig) SetClaudeAPIKey(apiKey string) {
	c.claudeAPIKey = apiKey
}

// SetSkipAI sets whether to skip AI analysis.
func (c *DefaultConfig) SetSkipAI(skipAI bool) {
	c.skipAI = skipAI
}

// SetUpdateGoMod sets whether to update go.mod.
func (c *DefaultConfig) SetUpdateGoMod(update bool) {
	c.updateGoMod = update
}

// SetValidateGoMod sets whether to validate go.mod.
func (c *DefaultConfig) SetValidateGoMod(validate bool) {
	c.validateGoMod = validate
}

// Validate validates the configuration.
func (c *DefaultConfig) Validate() error {
	// Validation mode-specific validation
	switch c.validationMode {
	case ValidationModeDelegated, ValidationModeIndependent:
		// Valid modes
	default:
		return fmt.Errorf(constants.ErrInvalidValidationMode)
	}

	// Both validation modes require Prysm connection
	if c.prysmHost == "" {
		return fmt.Errorf(constants.ErrPrysmHostRequired, c.validationMode)
	}

	// Test duration should be positive
	if c.testDuration <= 0 {
		return fmt.Errorf("test duration must be positive")
	}

	// Ports should be valid
	if c.prysmHTTPPort <= 0 || c.prysmHTTPPort > 65535 {
		return fmt.Errorf("prysm HTTP port must be between 1 and 65535")
	}

	if c.prysmGRPCPort <= 0 || c.prysmGRPCPort > 65535 {
		return fmt.Errorf("prysm gRPC port must be between 1 and 65535")
	}

	return nil
}

// AsHermesConfig converts the configuration to Hermes node configuration.
func (c *DefaultConfig) AsHermesConfig() *eth.NodeConfig {
	return &eth.NodeConfig{
		PrivateKeyStr:               c.privateKeyStr,
		DialTimeout:                 c.dialTimeout,
		Devp2pHost:                  c.devp2pHost,
		Devp2pPort:                  c.devp2pPort,
		Libp2pHost:                  c.libp2pHost,
		Libp2pPort:                  c.libp2pPort,
		Libp2pPeerscoreSnapshotFreq: constants.DefaultLibp2pPeerscoreFreq,
		PrysmHost:                   c.prysmHost,
		PrysmPortHTTP:               c.prysmHTTPPort,
		PrysmPortGRPC:               c.prysmGRPCPort,
		PrysmUseTLS:                 c.useTLS,
		MaxPeers:                    c.maxPeers,
		DialConcurrency:             c.dialConcurrency,
		DataStreamType:              host.DataStreamtypeFromStr(c.dataStreamType),
		SubnetConfigs:               c.subnets,
	}
}

// HostWithRedactedSecrets redacts passwords from connection strings for secure logging.
func (c *DefaultConfig) HostWithRedactedSecrets() string {
	return redactConnectionString(c.prysmHost)
}

// redactConnectionString redacts passwords from connection strings.
func redactConnectionString(connStr string) string {
	// Handle user:pass@host format
	if len(connStr) > 0 && connStr[0] != '@' {
		parts := strings.Split(connStr, "@")
		if len(parts) == 2 {
			userParts := strings.Split(parts[0], ":")
			if len(userParts) == 2 {
				return userParts[0] + ":****@" + parts[1]
			}
		}
	}

	return connStr
}

// Clone creates a deep copy of the configuration.
func (c *DefaultConfig) Clone() *DefaultConfig {
	clone := *c

	// Deep copy subnets map
	clone.subnets = make(map[string]*eth.SubnetConfig)
	for k, v := range c.subnets {
		clone.subnets[k] = v
	}

	return &clone
}

// GetValidationConfigs returns configuration mappings for each validation mode.
func GetValidationConfigs() map[ValidationMode]ValidationConfig {
	return map[ValidationMode]ValidationConfig{
		ValidationModeDelegated: {
			Mode:          ValidationModeDelegated,
			HermesVersion: "v0.0.4-0.20250513093811-320c1c3ee6e2",
			ConfigOverrides: map[string]interface{}{
				"validation-mode": "delegated",
			},
		},
		ValidationModeIndependent: {
			Mode:          ValidationModeIndependent,
			HermesVersion: "v0.0.4-0.20250613124328-491d55340eb7",
			ConfigOverrides: map[string]interface{}{
				"validation-mode":                  "independent",
				"validation-attestation-threshold": 10,
				"validation-cache-size":            10000,
			},
		},
	}
}
