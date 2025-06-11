package peer

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	
	"github.com/ethpandaops/hermes-peer-score/constants"
)

// DefaultSessionManager implements the SessionManager interface
type DefaultSessionManager struct {
	repo   Repository
	logger logrus.FieldLogger
}

// NewSessionManager creates a new session manager
func NewSessionManager(repo Repository, logger logrus.FieldLogger) *DefaultSessionManager {
	return &DefaultSessionManager{
		repo:   repo,
		logger: logger.WithField("component", "session_manager"),
	}
}

// StartSession starts a new connection session for a peer
func (sm *DefaultSessionManager) StartSession(peerID string, connectedAt time.Time) error {
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		// Check if we need a new session or this is a duplicate event
		currentSession := sm.getCurrentSession(peer)
		
		if currentSession == nil || currentSession.Disconnected {
			// Previous session ended or no active session - start new session
			newSession := ConnectionSession{
				ConnectedAt:  &connectedAt,
				Disconnected: false,
				PeerScores:   make([]PeerScoreSnapshot, 0),
				GoodbyeEvents: make([]GoodbyeEvent, 0),
				MeshEvents:   make([]MeshEvent, 0),
			}
			
			peer.ConnectionSessions = append(peer.ConnectionSessions, newSession)
			peer.TotalConnections++
			peer.LastSeenAt = &connectedAt
			
			sm.logger.WithFields(logrus.Fields{
				"peer_id":    formatShortPeerID(peerID),
				"conn_count": len(peer.ConnectionSessions),
			}).Debug("Started new session")
		} else {
			// Duplicate connection event for active session
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Debug("Duplicate connection event")
		}
	})
	
	return nil
}

// EndSession ends the current active session for a peer
func (sm *DefaultSessionManager) EndSession(peerID string, disconnectedAt time.Time) error {
	var sessionFound bool
	
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		// Find the current active session and mark it as disconnected
		currentSession := sm.getCurrentSession(peer)
		if currentSession == nil {
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("No active session found for disconnection")
			return
		}
		
		if currentSession.Disconnected {
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("Session already marked as disconnected")
			return
		}
		
		// Mark session as disconnected and calculate duration
		currentSession.Disconnected = true
		currentSession.DisconnectedAt = &disconnectedAt
		
		if currentSession.ConnectedAt != nil {
			duration := disconnectedAt.Sub(*currentSession.ConnectedAt)
			currentSession.Duration = &duration
		}
		
		// Update peer's last seen time
		peer.LastSeenAt = &disconnectedAt
		sessionFound = true
		
		sm.logger.WithFields(logrus.Fields{
			"peer_id":  formatShortPeerID(peerID),
			"duration": currentSession.Duration,
		}).Debug("Ended session")
	})
	
	if !sessionFound {
		return fmt.Errorf("no active session found for peer %s", peerID)
	}
	
	return nil
}

// IdentifyPeer marks a peer as identified in the current session
func (sm *DefaultSessionManager) IdentifyPeer(peerID string, identifiedAt time.Time, clientAgent string) error {
	var sessionFound bool
	
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		currentSession := sm.getCurrentSession(peer)
		if currentSession == nil {
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("No active session for identification")
			return
		}
		
		// Set identification details
		currentSession.IdentifiedAt = &identifiedAt
		
		// Update peer client information if provided
		if clientAgent != "" {
			peer.ClientAgent = clientAgent
			peer.ClientType = normalizeClientType(clientAgent)
		}
		
		peer.LastSeenAt = &identifiedAt
		sessionFound = true
		
		sm.logger.WithFields(logrus.Fields{
			"peer_id":     formatShortPeerID(peerID),
			"client_type": peer.ClientType,
		}).Debug("Identified peer")
	})
	
	if !sessionFound {
		return fmt.Errorf("no active session found for peer %s", peerID)
	}
	
	return nil
}

// AddPeerScore adds a peer score snapshot to the current session
func (sm *DefaultSessionManager) AddPeerScore(peerID string, score PeerScoreSnapshot) error {
	var sessionFound bool
	
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		currentSession := sm.getCurrentSession(peer)
		if currentSession == nil {
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("No active session for peer score")
			return
		}
		
		currentSession.PeerScores = append(currentSession.PeerScores, score)
		sessionFound = true
		
		sm.logger.WithFields(logrus.Fields{
			"peer_id": formatShortPeerID(peerID),
			"score":   score.Score,
		}).Debug("Added peer score")
	})
	
	if !sessionFound {
		return fmt.Errorf("no active session found for peer %s", peerID)
	}
	
	return nil
}

// AddGoodbyeEvent adds a goodbye event to the current session
func (sm *DefaultSessionManager) AddGoodbyeEvent(peerID string, event GoodbyeEvent) error {
	var sessionFound bool
	
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		currentSession := sm.getCurrentSession(peer)
		if currentSession == nil {
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("No active session for goodbye event")
			return
		}
		
		currentSession.GoodbyeEvents = append(currentSession.GoodbyeEvents, event)
		sessionFound = true
		
		sm.logger.WithFields(logrus.Fields{
			"peer_id": formatShortPeerID(peerID),
			"code":    event.Code,
			"reason":  event.Reason,
		}).Debug("Added goodbye event")
	})
	
	if !sessionFound {
		return fmt.Errorf("no active session found for peer %s", peerID)
	}
	
	return nil
}

// AddMeshEvent adds a mesh event to the current session
func (sm *DefaultSessionManager) AddMeshEvent(peerID string, event MeshEvent) error {
	var sessionFound bool
	
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		currentSession := sm.getCurrentSession(peer)
		if currentSession == nil {
			sm.logger.WithField("peer_id", formatShortPeerID(peerID)).Warn("No active session for mesh event")
			return
		}
		
		currentSession.MeshEvents = append(currentSession.MeshEvents, event)
		sessionFound = true
		
		sm.logger.WithFields(logrus.Fields{
			"peer_id":   formatShortPeerID(peerID),
			"type":      event.Type,
			"direction": event.Direction,
			"topic":     event.Topic,
		}).Debug("Added mesh event")
	})
	
	if !sessionFound {
		return fmt.Errorf("no active session found for peer %s", peerID)
	}
	
	return nil
}

// IncrementMessageCount increments the message count for the current session
func (sm *DefaultSessionManager) IncrementMessageCount(peerID string) error {
	var sessionFound bool
	
	sm.repo.UpdatePeer(peerID, func(peer *Stats) {
		currentSession := sm.getCurrentSession(peer)
		if currentSession == nil {
			return // No active session, silently ignore
		}
		
		currentSession.MessageCount++
		sessionFound = true
	})
	
	if !sessionFound {
		return fmt.Errorf("no active session found for peer %s", peerID)
	}
	
	return nil
}

// getCurrentSession returns the current active session for a peer
func (sm *DefaultSessionManager) getCurrentSession(peer *Stats) *ConnectionSession {
	if peer == nil || len(peer.ConnectionSessions) == 0 {
		return nil
	}
	
	// Return the last session if it's not disconnected
	lastSession := &peer.ConnectionSessions[len(peer.ConnectionSessions)-1]
	if !lastSession.Disconnected {
		return lastSession
	}
	
	return nil
}

// normalizeClientType normalizes client agent strings to standard types
func normalizeClientType(clientAgent string) string {
	if clientAgent == "" {
		return constants.Unknown
	}
	
	// This is the same logic from the events package but kept here to avoid circular imports
	agent := clientAgent
	switch {
	case contains(agent, constants.Lighthouse):
		return constants.Lighthouse
	case contains(agent, constants.Prysm):
		return constants.Prysm
	case contains(agent, constants.Teku):
		return constants.Teku
	case contains(agent, constants.Nimbus):
		return constants.Nimbus
	case contains(agent, constants.Lodestar):
		return constants.Lodestar
	case contains(agent, constants.Besu):
		return constants.Besu
	case contains(agent, constants.Grandine):
		return constants.Grandine
	default:
		return constants.Unknown
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive contains check
	sLower := toLower(s)
	substrLower := toLower(substr)
	return stringContains(sLower, substrLower)
}

// toLower converts string to lowercase
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// stringContains checks if s contains substr
func stringContains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}