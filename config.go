package main

import (
	"errors"
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
		n.MaxPeers = 30
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

func (n *ToolConfig) Validate() error {
	if n.PrysmHost == "" {
		return errors.New("--prysm-host is required")
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
		Libp2pPeerscoreSnapshotFreq: 30 * time.Second,
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
