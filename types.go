package main

import (
	"time"
)

// TopicScore represents the peer score for a specific topic.
type TopicScore struct {
	Topic                    string        `json:"topic"`
	TimeInMesh               time.Duration `json:"time_in_mesh"`
	FirstMessageDeliveries   float64       `json:"first_message_deliveries"`
	MeshMessageDeliveries    float64       `json:"mesh_message_deliveries"`
	InvalidMessageDeliveries float64       `json:"invalid_message_deliveries"`
}

// PeerScoreSnapshot represents a snapshot of a peer's score at a specific time.
type PeerScoreSnapshot struct {
	Timestamp            time.Time    `json:"timestamp"`
	Score                float64      `json:"score"`
	AppSpecificScore     float64      `json:"app_specific_score"`
	IPColocationFactor   float64      `json:"ip_colocation_factor"`
	BehaviourPenalty     float64      `json:"behaviour_penalty"`
	Topics               []TopicScore `json:"topics"`
}

// PeerScoreConfig holds configuration parameters for the peer score tool.
// It defines test duration, Hermes binary path, and command-line arguments.
type PeerScoreConfig struct {
	ToolConfig *ToolConfig `json:"tool_config"`

	TestDuration   time.Duration `yaml:"test_duration"`   // How long to run the peer connectivity test.
	ReportInterval time.Duration `yaml:"report_interval"` // Frequency of status reports during testing.
}

// ConnectionSession represents a single connection timeline for a peer.
// A peer may have multiple sessions if they reconnect multiple times.
type ConnectionSession struct {
	ConnectedAt        *time.Time          `json:"connected_at"`        // Timestamp when this session's connection was established.
	IdentifiedAt       *time.Time          `json:"identified_at"`       // Timestamp when the peer was identified in this session.
	DisconnectedAt     *time.Time          `json:"disconnected_at"`     // Timestamp when this session disconnected.
	MessageCount       int                 `json:"message_count"`       // Total number of messages exchanged in this session.
	ConnectionDuration time.Duration       `json:"connection_duration"` // How long this session lasted.
	Disconnected       bool                `json:"disconnected"`        // Whether this session has disconnected.
	PeerScores         []PeerScoreSnapshot `json:"peer_scores"`         // All peer score snapshots for this session.
}

// PeerStats contains detailed statistics for an individual peer across all connection sessions.
// This tracks the lifecycle and behavior of each peer discovered during testing.
type PeerStats struct {
	PeerID             string              `json:"peer_id"`              // Unique peer identifier (libp2p peer ID).
	ClientType         string              `json:"client_type"`          // Ethereum client implementation (lighthouse, prysm, etc.).
	ClientAgent        string              `json:"client_agent"`         // Raw agent of the client (from most recent identification).
	ConnectionSessions []ConnectionSession `json:"connection_sessions"`  // All connection sessions for this peer.
	TotalConnections   int                 `json:"total_connections"`    // Total number of connection attempts.
	TotalMessageCount  int                 `json:"total_message_count"`  // Total messages across all sessions.
	FirstSeenAt        *time.Time          `json:"first_seen_at"`        // When we first encountered this peer.
	LastSeenAt         *time.Time          `json:"last_seen_at"`         // When we last interacted with this peer.
}

// PeerScoreReport contains the comprehensive analysis results from a peer scoring test.
// This is the main output structure containing all metrics, scores, and diagnostic information.
type PeerScoreReport struct {
	Config               PeerScoreConfig            `json:"config"`                 // Configuration used for this test run.
	Timestamp            time.Time                  `json:"timestamp"`              // When this report was generated.
	StartTime            time.Time                  `json:"start_time"`             // When the test execution began.
	EndTime              time.Time                  `json:"end_time"`               // When the test execution completed.
	Duration             time.Duration              `json:"duration"`               // Total time spent running the test.
	TotalConnections     int                        `json:"total_connections"`      // Total number of peer connections established.
	SuccessfulHandshakes int                        `json:"successful_handshakes"`  // Number of successful peer identifications.
	FailedHandshakes     int                        `json:"failed_handshakes"`      // Number of failed peer identifications.
	Peers                map[string]*PeerStats      `json:"peers"`                  // Detailed statistics for each individual peer.
	PeerEventCounts      map[string]map[string]int  `json:"peer_event_counts"`      // Count of event types by peer ID.
}

// HTMLTemplateData represents the data structure used to generate HTML reports.
// This extends the basic report with additional computed fields needed for web presentation.
type HTMLTemplateData struct {
	GeneratedAt time.Time       `json:"generated_at"` // When the HTML report was generated.
	Report      PeerScoreReport `json:"report"`       // The underlying peer score report data.
}
