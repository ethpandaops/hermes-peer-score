package main

import (
	"time"
)

// GoodbyeTiming tracks detailed timing information for each goodbye message
type GoodbyeTiming struct {
	Reason            string        `json:"reason"`              // The goodbye reason
	Timestamp         time.Time     `json:"timestamp"`           // When the goodbye was received
	DurationFromStart time.Duration `json:"duration_from_start"` // Time since connection establishment
	Sequence          int           `json:"sequence"`            // Order of this goodbye (1st, 2nd, etc.)
}

// TimingPattern represents a detected pattern in connection/goodbye timing
type TimingPattern struct {
	ClientType        string        `json:"client_type"`         // Which client exhibits this pattern
	GoodbyeReason     string        `json:"goodbye_reason"`      // The goodbye reason associated with the pattern
	AverageDuration   time.Duration `json:"average_duration"`    // Average time from connection to goodbye
	Occurrences       int           `json:"occurrences"`         // How many times this pattern occurred
	Pattern           string        `json:"pattern"`             // Human-readable pattern description
}

// ConnectionTiming contains timing correlation analysis for the entire test
type ConnectionTiming struct {
	TotalConnections           int               `json:"total_connections"`            // Total connections analyzed
	AverageConnectionDuration  time.Duration     `json:"average_connection_duration"`  // Average time peers stay connected
	MedianConnectionDuration   time.Duration     `json:"median_connection_duration"`   // Median connection duration
	FastestDisconnect         time.Duration     `json:"fastest_disconnect"`           // Shortest connection duration
	LongestConnection         time.Duration     `json:"longest_connection"`           // Longest connection duration
	GoodbyeReasonTimings      map[string]GoodbyeReasonTiming `json:"goodbye_reason_timings"` // Timing analysis by goodbye reason
	ClientTimingPatterns      []TimingPattern   `json:"client_timing_patterns"`       // Detected client-specific patterns
	SuspiciousPatterns        []string          `json:"suspicious_patterns"`          // Patterns that might indicate scoring issues
}

// GoodbyeReasonTiming analyzes timing patterns for specific goodbye reasons
type GoodbyeReasonTiming struct {
	Reason            string        `json:"reason"`              // The goodbye reason
	Count             int           `json:"count"`               // How many times this reason occurred  
	AverageDuration   time.Duration `json:"average_duration"`    // Average time from connection to this goodbye
	MedianDuration    time.Duration `json:"median_duration"`     // Median time to this goodbye
	MinDuration       time.Duration `json:"min_duration"`        // Fastest occurrence
	MaxDuration       time.Duration `json:"max_duration"`        // Slowest occurrence
	ClientBreakdown   map[string]int `json:"client_breakdown"`   // Count by client type
}

// PeerScoreConfig holds configuration parameters for the peer score tool.
// It defines test duration, Hermes binary path, and command-line arguments.
type PeerScoreConfig struct {
	HermesPath     string        `yaml:"hermes_path"`     // Path to the Hermes binary executable.
	TestDuration   time.Duration `yaml:"test_duration"`   // How long to run the peer connectivity test.
	ReportInterval time.Duration `yaml:"report_interval"` // Frequency of status reports during testing.
	HermesArgs     []string      `yaml:"hermes_args"`     // Command-line arguments passed to Hermes.
}

// PeerStats contains detailed statistics for an individual peer connection.
// This tracks the lifecycle and behavior of each peer discovered during testing.
type PeerStats struct {
	PeerID         string    `json:"peer_id"`         // Unique peer identifier (libp2p peer ID).
	ClientType     string    `json:"client_type"`     // Ethereum client implementation (lighthouse, prysm, etc.).
	ConnectedAt    time.Time `json:"connected_at"`    // Timestamp when the peer connection was established.
	Disconnected   bool      `json:"disconnected"`    // Whether the peer has disconnected.
	DisconnectedAt time.Time `json:"disconnected_at"` // Timestamp when the peer disconnected.
	HandshakeOK    bool      `json:"handshake_ok"`    // Whether the initial handshake completed successfully.
	GoodbyeCount   int       `json:"goodbye_count"`   // Number of goodbye messages received from this peer.
	LastGoodbye    string    `json:"last_goodbye"`    // The most recent goodbye reason from this peer.
	MessageCount   int       `json:"message_count"`   // Total number of messages exchanged with this peer.
	
	// Timing correlation fields for enhanced analysis
	ConnectionDuration    time.Duration     `json:"connection_duration"`     // How long the peer was connected before goodbye/disconnect
	FirstGoodbyeAt       time.Time         `json:"first_goodbye_at"`        // When the first goodbye was received
	GoodbyeTimings       []GoodbyeTiming   `json:"goodbye_timings"`         // Detailed timing for each goodbye message
	TimeToFirstGoodbye   time.Duration     `json:"time_to_first_goodbye"`   // Duration from connection to first goodbye
	ReconnectionAttempts int              `json:"reconnection_attempts"`    // Number of times this peer reconnected
}

// PeerScoreReport contains the comprehensive analysis results from a peer scoring test.
// This is the main output structure containing all metrics, scores, and diagnostic information.
type PeerScoreReport struct {
	Timestamp            time.Time                 `json:"timestamp"`             // When this report was generated.
	Config               PeerScoreConfig           `json:"config"`                // Configuration used for this test run.
	StartTime            time.Time                 `json:"start_time"`            // When the test execution began.
	EndTime              time.Time                 `json:"end_time"`              // When the test execution completed.
	Duration             time.Duration             `json:"duration"`              // Total time spent running the test.
	TotalConnections     int                       `json:"total_connections"`     // Total number of peer connections established.
	SuccessfulHandshakes int                       `json:"successful_handshakes"` // Number of successful peer handshakes.
	FailedHandshakes     int                       `json:"failed_handshakes"`     // Number of failed peer handshakes.
	GoodbyeMessages      int                       `json:"goodbye_messages"`      // Total goodbye messages received from all peers.
	GoodbyeReasons       map[string]int            `json:"goodbye_reasons"`       // Breakdown of goodbye reasons and their frequency.
	GoodbyesByClient     map[string]map[string]int `json:"goodbyes_by_client"`    // Goodbye reasons grouped by client type.
	PeersByClient        map[string]int            `json:"peers_by_client"`       // Number of peers per client implementation.
	UniqueClients        int                       `json:"unique_clients"`        // Number of different client implementations discovered.
	Peers                map[string]*PeerStats     `json:"peers"`                 // Detailed statistics for each individual peer.
	OverallScore         float64                   `json:"overall_score"`         // Calculated overall peer score (0-100%).
	Summary              string                    `json:"summary"`               // Human-readable summary of the test results.
	Errors               []string                  `json:"errors"`                // List of errors encountered during testing.
	ConnectionFailed     bool                      `json:"connection_failed"`     // Whether the beacon node connection failed.
	
	// Enhanced timing correlation analysis
	TimingAnalysis       ConnectionTiming          `json:"timing_analysis"`       // Comprehensive timing correlation analysis
	DownscoreIndicators  []string                  `json:"downscore_indicators"`  // Indicators suggesting peer downscoring
}

// HTMLTemplateData represents the data structure used to generate HTML reports.
// This extends the basic report with additional computed fields needed for web presentation.
type HTMLTemplateData struct {
	GeneratedAt         time.Time       `json:"generated_at"`         // When the HTML report was generated.
	Report              PeerScoreReport `json:"report"`               // The underlying peer score report data.
	ScoreClassification string          `json:"score_classification"` // Human-readable score category (Excellent, Good, etc.).
	ConnectionRate      float64         `json:"connection_rate"`      // Percentage of successful connections.
	ClientList          []ClientStat    `json:"client_list"`          // Sorted list of client statistics for display.
}

// ClientStat represents client distribution data for HTML report rendering.
// This provides a clean structure for displaying client type statistics in templates.
type ClientStat struct {
	Name  string `json:"name"`  // Client implementation name (lighthouse, prysm, etc.).
	Count int    `json:"count"` // Number of peers running this client implementation.
}
