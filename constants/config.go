package constants

import "time"

// Default configuration values
const (
	// Time-related constants
	DefaultTestDuration         = 2 * time.Minute
	DefaultStatusReportInterval = 15 * time.Second
	DefaultPeerScoreFreq        = 5 * time.Second
	DefaultReportInterval       = 2 * time.Minute
	DefaultLibp2pPeerscoreFreq  = 30 * time.Second

	// Network and connection constants
	DefaultPrysmHTTPPort   = 443
	DefaultPrysmGRPCPort   = 443
	DefaultMaxPeers        = 80
	DefaultDialConcurrency = 16
	DefaultDialTimeout     = 5 * time.Second

	// PubSub and messaging constants
	DefaultPubSubLimit     = 200
	DefaultPubSubQueueSize = 200

	// File and data constants
	DefaultFilePermissions = 0644
	ShortPeerIDLength      = 12
	MaxDisconnectReasons   = 5

	// Default hosts and addresses
	DefaultDevp2pHost = "0.0.0.0"
	DefaultLibp2pHost = "0.0.0.0"

	// Validation configuration
	DefaultAttestationThreshold = 10
	DefaultValidationCacheSize  = 10000
	DefaultStateSyncInterval    = "30s"
)

// Hermes version constants for different validation modes
const (
	HermesVersionDelegated   = "v0.0.4-0.20250513093811-320c1c3ee6e2"
	HermesVersionIndependent = "v0.0.4-0.20250613124328-491d55340eb7"
)

// Default filenames
const (
	DefaultJSONReportFile = "peer-score-report.json"
	DefaultHTMLReportFile = "peer-score-report.html"
	DefaultDataJSFile     = "peer-score-report-data.js"
)

// Data stream types
const (
	DefaultDataStreamType = "callback"
)

// Error messages
const (
	ErrInvalidValidationMode = "invalid validation mode: must be 'delegated' or 'independent'"
	ErrPrysmHostRequired     = "--prysm-host is required for %s validation mode"
)
