package parsers

import "time"

// PeerScoreData represents parsed peer score information.
type PeerScoreData struct {
	Timestamp          time.Time    `json:"timestamp"`
	Score              float64      `json:"score"`
	AppSpecificScore   float64      `json:"app_specific_score"`
	IPColocationFactor float64      `json:"ip_colocation_factor"`
	BehaviourPenalty   float64      `json:"behaviour_penalty"`
	Topics             []TopicScore `json:"topics"`
}

// TopicScore represents the peer score for a specific topic.
type TopicScore struct {
	Topic                    string        `json:"topic"`
	TimeInMesh               time.Duration `json:"time_in_mesh"`
	FirstMessageDeliveries   float64       `json:"first_message_deliveries"`
	MeshMessageDeliveries    float64       `json:"mesh_message_deliveries"`
	InvalidMessageDeliveries float64       `json:"invalid_message_deliveries"`
}

// GoodbyeData represents parsed goodbye event information.
type GoodbyeData struct {
	Timestamp time.Time `json:"timestamp"`
	Code      uint64    `json:"code"`
	Reason    string    `json:"reason"`
}

// MeshData represents parsed mesh event information.
type MeshData struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`      // "GRAFT" or "PRUNE"
	Direction string    `json:"direction"` // "sent" or "received"
	Topic     string    `json:"topic"`
	Reason    string    `json:"reason"`
}

// ConnectionData represents parsed connection event information.
type ConnectionData struct {
	Timestamp   time.Time `json:"timestamp"`
	PeerID      string    `json:"peer_id"`
	ClientAgent string    `json:"client_agent"`
	ClientType  string    `json:"client_type"`
}

// DisconnectionData represents parsed disconnection event information.
type DisconnectionData struct {
	Timestamp time.Time `json:"timestamp"`
	PeerID    string    `json:"peer_id"`
}

// StatusData represents parsed status event information.
type StatusData struct {
	Timestamp time.Time `json:"timestamp"`
	PeerID    string    `json:"peer_id"`
	Success   bool      `json:"success"`
}
