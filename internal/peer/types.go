package peer

import "time"

// Stats contains detailed statistics for an individual peer across all connection sessions
type Stats struct {
	PeerID               string              `json:"peer_id"`
	ClientType           string              `json:"client_type"`
	ClientAgent          string              `json:"client_agent"`
	ConnectionSessions   []ConnectionSession `json:"connection_sessions"`
	TotalConnections     int                 `json:"total_connections"`
	TotalMessageCount    int                 `json:"total_message_count"`
	SuccessfulHandshakes int                 `json:"successful_handshakes"`
	FailedHandshakes     int                 `json:"failed_handshakes"`
	FirstSeenAt          *time.Time          `json:"first_seen_at"`
	LastSeenAt           *time.Time          `json:"last_seen_at"`
}

// ConnectionSession represents a single connection timeline for a peer
type ConnectionSession struct {
	ConnectedAt    *time.Time          `json:"connected_at"`
	IdentifiedAt   *time.Time          `json:"identified_at"`
	DisconnectedAt *time.Time          `json:"disconnected_at"`
	MessageCount   int                 `json:"message_count"`
	Duration       *time.Duration      `json:"duration"`
	Disconnected   bool                `json:"disconnected"`
	PeerScores     []PeerScoreSnapshot `json:"peer_scores"`
	GoodbyeEvents  []GoodbyeEvent      `json:"goodbye_events"`
	MeshEvents     []MeshEvent         `json:"mesh_events"`
}

// PeerScoreSnapshot represents a snapshot of a peer's score at a specific time
type PeerScoreSnapshot struct {
	Timestamp          time.Time          `json:"timestamp"`
	Score              float64            `json:"score"`
	AppSpecificScore   float64            `json:"app_specific_score"`
	IPColocationFactor float64            `json:"ip_colocation_factor"`
	BehaviourPenalty   float64            `json:"behaviour_penalty"`
	Topics             map[string]float64 `json:"topics"`
}

// TopicScore represents the peer score for a specific topic
type TopicScore struct {
	Topic                    string        `json:"topic"`
	TimeInMesh               time.Duration `json:"time_in_mesh"`
	FirstMessageDeliveries   float64       `json:"first_message_deliveries"`
	MeshMessageDeliveries    float64       `json:"mesh_message_deliveries"`
	InvalidMessageDeliveries float64       `json:"invalid_message_deliveries"`
}

// GoodbyeEvent represents a goodbye message received from a peer
type GoodbyeEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Code      uint64    `json:"code"`
	Reason    string    `json:"reason"`
}

// GoodbyeReasonStats tracks statistics for a specific goodbye reason
type GoodbyeReasonStats struct {
	Reason   string   `json:"reason"`   // Original reason string
	Count    int      `json:"count"`    // Number of occurrences
	Codes    []uint64 `json:"codes"`    // All unique codes seen with this reason
	Examples []string `json:"examples"` // First few examples of this reason
}

// GoodbyeEventsSummary contains aggregated goodbye event statistics
type GoodbyeEventsSummary struct {
	TotalEvents   int                   `json:"total_events"`   // Total number of goodbye events
	ReasonStats   []*GoodbyeReasonStats `json:"reason_stats"`   // Sorted by count (most common first)
	UniqueReasons int                   `json:"unique_reasons"` // Number of unique reasons
	TopReasons    []string              `json:"top_reasons"`    // Top 5 most common reasons
	CodeFrequency map[uint64]int        `json:"code_frequency"` // Code occurrence count
}

// MeshEvent represents a GRAFT/PRUNE event for mesh participation tracking
type MeshEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Direction string    `json:"direction"`
	Topic     string    `json:"topic"`
	Reason    string    `json:"reason"`
}

// ConnectionStats holds aggregate connection statistics
type ConnectionStats struct {
	TotalConnections     int `json:"total_connections"`
	SuccessfulHandshakes int `json:"successful_handshakes"`
	FailedHandshakes     int `json:"failed_handshakes"`
	ConnectedPeers       int `json:"connected_peers"`
}

// DurationStats holds aggregate duration statistics
type DurationStats struct {
	AverageDuration time.Duration `json:"average_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	MinDuration     time.Duration `json:"min_duration"`
}
