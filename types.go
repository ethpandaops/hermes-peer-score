package main

import (
	"time"
)

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
	PeerID       string    `json:"peer_id"`       // Unique peer identifier (libp2p peer ID).
	ClientType   string    `json:"client_type"`   // Ethereum client implementation (lighthouse, prysm, etc.).
	ConnectedAt  time.Time `json:"connected_at"`  // Timestamp when the peer connection was established.
	HandshakeOK  bool      `json:"handshake_ok"`  // Whether the initial handshake completed successfully.
	GoodbyeCount int       `json:"goodbye_count"` // Number of goodbye messages received from this peer.
	LastGoodbye  string    `json:"last_goodbye"`  // The most recent goodbye reason from this peer.
	MessageCount int       `json:"message_count"` // Total number of messages exchanged with this peer.
}

// PeerScoreReport contains the comprehensive analysis results from a peer scoring test.
// This is the main output structure containing all metrics, scores, and diagnostic information.
type PeerScoreReport struct {
	Timestamp            time.Time                    `json:"timestamp"`             // When this report was generated.
	Config               PeerScoreConfig              `json:"config"`                // Configuration used for this test run.
	StartTime            time.Time                    `json:"start_time"`            // When the test execution began.
	EndTime              time.Time                    `json:"end_time"`              // When the test execution completed.
	Duration             time.Duration                `json:"duration"`              // Total time spent running the test.
	TotalConnections     int                          `json:"total_connections"`     // Total number of peer connections established.
	SuccessfulHandshakes int                          `json:"successful_handshakes"` // Number of successful peer handshakes.
	FailedHandshakes     int                          `json:"failed_handshakes"`     // Number of failed peer handshakes.
	GoodbyeMessages      int                          `json:"goodbye_messages"`      // Total goodbye messages received from all peers.
	GoodbyeReasons       map[string]int               `json:"goodbye_reasons"`       // Breakdown of goodbye reasons and their frequency.
	GoodbyesByClient     map[string]map[string]int    `json:"goodbyes_by_client"`    // Goodbye reasons grouped by client type.
	PeersByClient        map[string]int               `json:"peers_by_client"`       // Number of peers per client implementation.
	UniqueClients        int                          `json:"unique_clients"`        // Number of different client implementations discovered.
	Peers                map[string]*PeerStats        `json:"peers"`                 // Detailed statistics for each individual peer.
	OverallScore         float64                      `json:"overall_score"`         // Calculated overall peer score (0-100%).
	Summary              string                       `json:"summary"`               // Human-readable summary of the test results.
	Errors               []string                     `json:"errors"`                // List of errors encountered during testing.
	ConnectionFailed     bool                         `json:"connection_failed"`     // Whether the beacon node connection failed.
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