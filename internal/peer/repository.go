package peer

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ethpandaops/hermes-peer-score/constants"
)

// InMemoryRepository implements the Repository interface using in-memory storage
type InMemoryRepository struct {
	peers        map[string]*Stats
	eventCounts  map[string]map[string]int
	mu           sync.RWMutex
	eventsMu     sync.RWMutex
	logger       logrus.FieldLogger
}

// NewInMemoryRepository creates a new in-memory peer repository
func NewInMemoryRepository(logger logrus.FieldLogger) *InMemoryRepository {
	return &InMemoryRepository{
		peers:       make(map[string]*Stats),
		eventCounts: make(map[string]map[string]int),
		logger:      logger.WithField("component", "peer_repository"),
	}
}

// GetPeer retrieves a peer by ID
func (r *InMemoryRepository) GetPeer(peerID string) (*Stats, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	peer, exists := r.peers[peerID]
	return peer, exists
}

// CreatePeer creates a new peer with the given ID
func (r *InMemoryRepository) CreatePeer(peerID string) *Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if existing, exists := r.peers[peerID]; exists {
		r.logger.WithField("peer_id", formatShortPeerID(peerID)).Debug("Peer already exists")
		return existing
	}
	
	now := time.Now()
	peer := &Stats{
		PeerID:             peerID,
		ConnectionSessions: make([]ConnectionSession, 0),
		TotalConnections:   0,
		TotalMessageCount:  0,
		FirstSeenAt:        &now,
		LastSeenAt:         &now,
	}
	
	r.peers[peerID] = peer
	r.logger.WithField("peer_id", formatShortPeerID(peerID)).Debug("Created new peer")
	
	return peer
}

// UpdatePeer safely updates a peer using the provided function
func (r *InMemoryRepository) UpdatePeer(peerID string, updateFn func(*Stats)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	peer, exists := r.peers[peerID]
	if !exists {
		r.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("Attempted to update non-existent peer")
		return
	}
	
	updateFn(peer)
}

// UpdateOrCreatePeer safely updates a peer or creates one if it doesn't exist
func (r *InMemoryRepository) UpdateOrCreatePeer(peerID string, updateFn func(*Stats)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	peer, exists := r.peers[peerID]
	if !exists {
		// Create a new peer with default values
		now := time.Now()
		peer = &Stats{
			PeerID:               peerID,
			ClientType:           constants.Unknown,
			ClientAgent:          "",
			FirstSeenAt:          &now,
			LastSeenAt:           &now,
			TotalConnections:     0,
			ConnectionSessions:   []ConnectionSession{},
		}
		r.peers[peerID] = peer
		r.logger.WithField("peer_id", formatShortPeerID(peerID)).Debug("Created new peer from event")
	}
	
	updateFn(peer)
}

// GetAllPeers returns a copy of all peers (thread-safe)
func (r *InMemoryRepository) GetAllPeers() map[string]*Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Create a deep copy to avoid data races
	peersCopy := make(map[string]*Stats)
	for peerID, peer := range r.peers {
		peersCopy[peerID] = r.deepCopyPeer(peer)
	}
	
	return peersCopy
}

// GetPeerEventCounts returns a copy of all peer event counts
func (r *InMemoryRepository) GetPeerEventCounts() map[string]map[string]int {
	r.eventsMu.RLock()
	defer r.eventsMu.RUnlock()
	
	// Create a deep copy to avoid data races
	eventsCopy := make(map[string]map[string]int)
	for peerID, events := range r.eventCounts {
		eventsCopy[peerID] = make(map[string]int)
		for eventType, count := range events {
			eventsCopy[peerID][eventType] = count
		}
	}
	
	return eventsCopy
}

// IncrementEventCount safely increments the event count for a peer
func (r *InMemoryRepository) IncrementEventCount(peerID, eventType string) {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()
	
	if _, exists := r.eventCounts[peerID]; !exists {
		r.eventCounts[peerID] = make(map[string]int)
	}
	
	r.eventCounts[peerID][eventType]++
}

// GetMutex returns the main mutex for external synchronization if needed
func (r *InMemoryRepository) GetMutex() *sync.RWMutex {
	return &r.mu
}

// GetEventMutex returns the events mutex for external synchronization if needed
func (r *InMemoryRepository) GetEventMutex() *sync.RWMutex {
	return &r.eventsMu
}

// GetPeerCount returns the total number of peers
func (r *InMemoryRepository) GetPeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers)
}

// GetActiveSessionCount returns the number of peers with active sessions
func (r *InMemoryRepository) GetActiveSessionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	activeCount := 0
	for _, peer := range r.peers {
		if r.hasActiveSession(peer) {
			activeCount++
		}
	}
	
	return activeCount
}

// deepCopyPeer creates a deep copy of a peer stats object
func (r *InMemoryRepository) deepCopyPeer(original *Stats) *Stats {
	if original == nil {
		return nil
	}
	
	// Deep copy connection sessions
	sessionsCopy := make([]ConnectionSession, len(original.ConnectionSessions))
	for i, session := range original.ConnectionSessions {
		sessionsCopy[i] = r.deepCopySession(session)
	}
	
	return &Stats{
		PeerID:             original.PeerID,
		ClientType:         original.ClientType,
		ClientAgent:        original.ClientAgent,
		ConnectionSessions: sessionsCopy,
		TotalConnections:   original.TotalConnections,
		TotalMessageCount:  original.TotalMessageCount,
		FirstSeenAt:        copyTimePtr(original.FirstSeenAt),
		LastSeenAt:         copyTimePtr(original.LastSeenAt),
	}
}

// deepCopySession creates a deep copy of a connection session
func (r *InMemoryRepository) deepCopySession(original ConnectionSession) ConnectionSession {
	// Deep copy peer scores
	scoresCopy := make([]PeerScoreSnapshot, len(original.PeerScores))
	for i, score := range original.PeerScores {
		// Deep copy topics map
		topicsCopy := make(map[string]float64)
		for topic, value := range score.Topics {
			topicsCopy[topic] = value
		}
		
		scoresCopy[i] = PeerScoreSnapshot{
			Timestamp:          score.Timestamp,
			Score:              score.Score,
			AppSpecificScore:   score.AppSpecificScore,
			IPColocationFactor: score.IPColocationFactor,
			BehaviourPenalty:   score.BehaviourPenalty,
			Topics:             topicsCopy,
		}
	}
	
	// Deep copy goodbye events
	goodbyesCopy := make([]GoodbyeEvent, len(original.GoodbyeEvents))
	copy(goodbyesCopy, original.GoodbyeEvents)
	
	// Deep copy mesh events
	meshCopy := make([]MeshEvent, len(original.MeshEvents))
	copy(meshCopy, original.MeshEvents)
	
	return ConnectionSession{
		ConnectedAt:        copyTimePtr(original.ConnectedAt),
		IdentifiedAt:       copyTimePtr(original.IdentifiedAt),
		DisconnectedAt:     copyTimePtr(original.DisconnectedAt),
		MessageCount:       original.MessageCount,
		Duration:           copyDurationPtr(original.Duration),
		Disconnected:       original.Disconnected,
		PeerScores:         scoresCopy,
		GoodbyeEvents:      goodbyesCopy,
		MeshEvents:         meshCopy,
	}
}

// hasActiveSession checks if a peer has any active (non-disconnected) sessions
func (r *InMemoryRepository) hasActiveSession(peer *Stats) bool {
	for _, session := range peer.ConnectionSessions {
		if !session.Disconnected {
			return true
		}
	}
	return false
}

// copyTimePtr creates a copy of a time pointer
func copyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copy := *t
	return &copy
}

// copyDurationPtr creates a copy of a duration pointer
func copyDurationPtr(d *time.Duration) *time.Duration {
	if d == nil {
		return nil
	}
	copy := *d
	return &copy
}

// formatShortPeerID returns a shortened version of the peer ID for logging
func formatShortPeerID(peerID string) string {
	if len(peerID) <= 12 {
		return peerID
	}
	return peerID[:12]
}