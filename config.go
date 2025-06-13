package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
)

// ToolConfig represents configuration for our upstream prysm instance and locally initiated hermes instance.
type ToolConfig struct {
	// The private key for the libp2p host and local enode in hex format
	PrivateKeyStr string
	// General timeout when communicating with other network participants
	DialTimeout time.Duration
	// The address information of the local ethereuem [enode.Node].
	Devp2pHost string
	Devp2pPort int
	// The address information of the local libp2p host
	Libp2pHost string
	Libp2pPort int
	// The address information where the Beacon API or Prysm's custom API is accessible at
	PrysmHost     string
	PrysmPortHTTP int
	PrysmPortGRPC int
	PrysmUseTLS   bool
	// The maximum number of peers our libp2p host can be connected to.
	MaxPeers int
	// Limits the number of concurrent connection establishment routines. When
	// we discover peers over discv5 and are not at our MaxPeers limit we try
	// to establish a connection to a peer. However, we limit the concurrency to
	// this DialConcurrency value.
	DialConcurrency int
	// DataStreamType is the type of data stream to use for the node (e.g. kinesis, callback, etc).
	DataStreamType string
	// Subnets is the configuration for gossipsub subnets.
	Subnets map[string]*eth.SubnetConfig
}

// SetDefaults applies default values to the ToolConfig if fields are zero values.
func (n *ToolConfig) SetDefaults() {
	if n.DialTimeout == 0 {
		n.DialTimeout = 5 * time.Second
	}

	if n.Devp2pHost == "" {
		n.Devp2pHost = "0.0.0.0"
	}

	if n.Libp2pHost == "" {
		n.Libp2pHost = "0.0.0.0"
	}

	if n.PrysmPortHTTP == 0 {
		n.PrysmPortHTTP = 443
	}

	if n.PrysmPortGRPC == 0 {
		n.PrysmPortGRPC = 443
	}

	if n.PrysmPortGRPC == 443 || n.PrysmPortHTTP == 443 {
		n.PrysmUseTLS = true
	}

	if n.MaxPeers == 0 {
		n.MaxPeers = 80
	}

	if n.DialConcurrency == 0 {
		n.DialConcurrency = 16
	}

	if n.DataStreamType == "" {
		n.DataStreamType = host.DataStreamTypeCallback.String()
	}

	if n.Subnets == nil {
		n.Subnets = make(map[string]*eth.SubnetConfig)
	}
}

func (n *ToolConfig) Validate(validationMode ValidationMode) error {
	// Both validation modes require Prysm connection
	if n.PrysmHost == "" {
		return fmt.Errorf("--prysm-host is required for %s validation mode", validationMode)
	}

	return nil
}

func (n *ToolConfig) AsHermesConfig() *eth.NodeConfig {
	return &eth.NodeConfig{
		PrivateKeyStr:               n.PrivateKeyStr,
		DialTimeout:                 n.DialTimeout,
		Devp2pHost:                  n.Devp2pHost,
		Devp2pPort:                  n.Devp2pPort,
		Libp2pHost:                  n.Libp2pHost,
		Libp2pPort:                  n.Libp2pPort,
		Libp2pPeerscoreSnapshotFreq: 15 * time.Minute,
		PrysmHost:                   n.PrysmHost,
		PrysmPortHTTP:               n.PrysmPortHTTP,
		PrysmPortGRPC:               n.PrysmPortGRPC,
		PrysmUseTLS:                 n.PrysmUseTLS,
		MaxPeers:                    n.MaxPeers,
		DialConcurrency:             n.DialConcurrency,
		DataStreamType:              host.DataStreamtypeFromStr(n.DataStreamType),
		SubnetConfigs:               n.Subnets,
	}
}

// HostWithRedactedSecrets redacts passwords from connection strings for secure logging.
func (n *ToolConfig) HostWithRedactedSecrets() string {
	if !strings.Contains(n.PrysmHost, ":") || !strings.Contains(n.PrysmHost, "@") {
		return n.PrysmHost
	}

	// Format is user:pass@host, redact the password.
	parts := strings.Split(n.PrysmHost, "@")
	if len(parts) != 2 {
		return n.PrysmHost
	}

	userParts := strings.Split(parts[0], ":")
	if len(userParts) != 2 {
		return n.PrysmHost
	}

	return userParts[0] + ":****@" + parts[1]
}

// MarshalJSON implements custom JSON marshaling to redact sensitive information.
func (n *ToolConfig) MarshalJSON() ([]byte, error) {
	// Create a copy of the struct with redacted sensitive fields
	type Alias ToolConfig

	return json.Marshal(&struct {
		*Alias
		PrysmHost string `json:"PrysmHost"`
	}{
		Alias:     (*Alias)(n),
		PrysmHost: n.HostWithRedactedSecrets(),
	})
}

// buildHermesArgs constructs config for Hermes.
func buildToolConfig() *ToolConfig {
	cfg := &ToolConfig{
		PrysmHost:     *prysmHost,
		PrysmPortHTTP: *prysmHTTPPort,
		PrysmPortGRPC: *prysmGRPCPort,
	}

	// Add TLS flag if either HTTP or gRPC port is 443.
	if *prysmHTTPPort == 443 || *prysmGRPCPort == 443 {
		cfg.PrysmUseTLS = true
	}

	// Set any defaults we use.
	cfg.SetDefaults()

	return cfg
}

// GetValidationConfigs returns configuration mappings for each validation mode.
func GetValidationConfigs() map[ValidationMode]ValidationConfig {
	return map[ValidationMode]ValidationConfig{
		ValidationModeDelegated: {
			Mode:          ValidationModeDelegated,
			HermesVersion: "v0.0.4-0.20250613124328-491d55340eb7",
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
				"validation-state-sync-interval":   "30s",
			},
		},
	}
}

// ValidateValidationMode checks if the provided validation mode is valid.
func ValidateValidationMode(mode string) (ValidationMode, error) {
	valMode := ValidationMode(mode)

	switch valMode {
	case ValidationModeDelegated, ValidationModeIndependent:
		return valMode, nil
	default:
		return "", errors.New("invalid validation mode: must be 'delegated' or 'independent'")
	}
}

// GenerateTimestampedFilename creates a filename with timestamp and validation mode.
func GenerateTimestampedFilename(validationMode ValidationMode, baseFilename string, timestamp time.Time) string {
	// Extract extension and name parts
	parts := strings.Split(baseFilename, ".")
	if len(parts) < 2 {
		// No extension
		return fmt.Sprintf("%s-%s-%s", baseFilename, string(validationMode), timestamp.Format("2006-01-02_15-04-05"))
	}

	// Insert validation mode and timestamp before the extension
	nameWithoutExt := strings.Join(parts[:len(parts)-1], ".")
	ext := parts[len(parts)-1]

	return fmt.Sprintf("%s-%s-%s.%s", nameWithoutExt, string(validationMode), timestamp.Format("2006-01-02_15-04-05"), ext)
}
