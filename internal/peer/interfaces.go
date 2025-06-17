package peer

import (
	"sync"
	"time"
)

// Repository defines the interface for peer data management
type Repository interface {
	GetPeer(peerID string) (*Stats, bool)
	CreatePeer(peerID string) *Stats
	UpdatePeer(peerID string, updateFn func(*Stats))
	UpdateOrCreatePeer(peerID string, updateFn func(*Stats))
	GetAllPeers() map[string]*Stats
	GetPeerEventCounts() map[string]map[string]int
	IncrementEventCount(peerID, eventType string)
	GetMutex() *sync.RWMutex
	GetEventMutex() *sync.RWMutex
}

// SessionManager defines the interface for managing peer connection sessions
type SessionManager interface {
	StartSession(peerID string, connectedAt time.Time) error
	EndSession(peerID string, disconnectedAt time.Time) error
	IdentifyPeer(peerID string, identifiedAt time.Time, clientAgent string) error
	AddPeerScore(peerID string, score PeerScoreSnapshot) error
	AddGoodbyeEvent(peerID string, event GoodbyeEvent) error
	AddMeshEvent(peerID string, event MeshEvent) error
	IncrementMessageCount(peerID string) error
}

// StatsCalculator defines the interface for calculating peer statistics
type StatsCalculator interface {
	CalculateConnectionStats(peers map[string]*Stats) ConnectionStats
	CalculateClientDistribution(peers map[string]*Stats) map[string]int
	CalculateDurationStats(peers map[string]*Stats) DurationStats
}