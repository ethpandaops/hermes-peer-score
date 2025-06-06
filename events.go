package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
)

func (pst *PeerScoreTool) handleHermesEvent(ctx context.Context, event *host.TraceEvent) error {
	switch event.Type {
	case "CONNECTED":
		pst.handleConnectionEvent(ctx, event)
	case "DISCONNECTED":
		pst.handleDisconnectionEvent(ctx, event)
	case "REQUEST_STATUS":
		pst.handleStatusEvent(ctx, event)
	case "PEERSCORE":
		pst.handlePeerScoreEvent(ctx, event)
	default:
	}

	pst.peerEventCountsMu.Lock()
	defer pst.peerEventCountsMu.Unlock()

	peerID := getPeerID(event)

	if _, ok := pst.peerEventCounts[peerID]; !ok {
		pst.peerEventCounts[peerID] = make(map[string]int)
	}

	pst.peerEventCounts[peerID][event.Type]++

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	peer, exists := pst.peers[peerID]
	if exists {
		currentSession := pst.getCurrentSession(peer)
		if currentSession != nil {
			currentSession.MessageCount++
		}
	}

	return nil
}

func (pst *PeerScoreTool) handleConnectionEvent(_ context.Context, event *host.TraceEvent) {
	data, err := libp2p.TraceEventToConnected(event)
	if err != nil {
		pst.log.WithError(err).Error("failed to convert event to connected event")
		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	peerID := data.RemotePeer.GetValue()
	now := time.Now()

	peer, exists := pst.peers[peerID]

	if !exists {
		// First time seeing this peer - create new PeerStats with first session
		session := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
		}

		pst.peers[peerID] = &PeerStats{
			PeerID:             peerID,
			ClientType:         normalizeClientType(data.AgentVersion.GetValue()),
			ClientAgent:        data.AgentVersion.GetValue(),
			ConnectionSessions: []ConnectionSession{session},
			TotalConnections:   1,
			FirstSeenAt:        &now,
			LastSeenAt:         &now,
		}

		pst.log.WithField("peer_id", peerID).Info("New peer connection")

		return
	}

	// Peer exists - check if we need a new session or this is a duplicate event.
	currentSession := pst.getCurrentSession(peer)

	if currentSession == nil || currentSession.Disconnected {
		// Previous session ended or no active session - start new session.
		newSession := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
		}

		peer.ConnectionSessions = append(peer.ConnectionSessions, newSession)
		peer.TotalConnections++
		peer.LastSeenAt = &now

		pst.log.WithFields(logrus.Fields{
			"peer_id":    peerID,
			"conn_count": len(peer.ConnectionSessions),
		}).Info("New peer connection")
	} else {
		// Duplicate connection event for active session. This is normal with libp2p.
		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
		}).Debug("Duplicate peer connection event")
	}
}

func (pst *PeerScoreTool) handleDisconnectionEvent(_ context.Context, event *host.TraceEvent) {
	data, err := libp2p.TraceEventToDisconnected(event)
	if err != nil {
		pst.log.WithError(err).Error("failed to convert event to disconnected event")
		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	peerID := data.RemotePeer.GetValue()
	peer, exists := pst.peers[peerID]
	now := time.Now()

	if !exists {
		// We've never seen this peer. Log it, unlikely to happen and not data we can use anyway.
		pst.log.WithField("peer_id", peerID).Warn("Received disconnection event for peer we've never seen")
		return
	}

	// Find the current active session and mark it as disconnected
	currentSession := pst.getCurrentSession(peer)
	if currentSession == nil {
		pst.log.WithField("peer_id", peerID).Warn("Received disconnection event but no active session found")
		return
	}

	if currentSession.Disconnected {
		pst.log.WithField("peer_id", peerID).Warn("Received disconnection event but session already marked as disconnected")
		return
	}

	// Mark session as disconnected and calculate duration
	currentSession.Disconnected = true
	currentSession.DisconnectedAt = &now

	if currentSession.ConnectedAt != nil {
		currentSession.ConnectionDuration = now.Sub(*currentSession.ConnectedAt)
	}

	// Update peer's last seen time
	peer.LastSeenAt = &now

	pst.log.WithFields(logrus.Fields{
		"peer_id":      peerID,
		"client_agent": normalizeClientType(peer.ClientAgent),
		"conn_count":   len(peer.ConnectionSessions),
		"duration":     currentSession.ConnectionDuration,
	}).Info("Peer disconnected")
}

func (pst *PeerScoreTool) handleStatusEvent(_ context.Context, event *host.TraceEvent) {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		pst.log.Errorf("handleStatusEvent: failed to convert request status payload to map[string]any")
		return
	}

	peerID, ok := payload["PeerID"].(string)
	if !ok {
		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
		}).Warn("handleStatusEvent: failed to parse peer_id from host payload")
		return
	}

	agentVersion, ok := payload["AgentVersion"].(string)
	if !ok {
		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
		}).Warn("handleStatusEvent: failed to parse agent_version from host payload")

		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	now := time.Now()
	peer, exists := pst.peers[peerID]

	if !exists {
		// We might not know about this peer yet, create it with a new session
		session := ConnectionSession{
			ConnectedAt:  &now,
			IdentifiedAt: &now,
			Disconnected: false,
		}

		pst.peers[peerID] = &PeerStats{
			PeerID:             peerID,
			ClientType:         normalizeClientType(agentVersion),
			ClientAgent:        agentVersion,
			ConnectionSessions: []ConnectionSession{session},
			TotalConnections:   1,
			FirstSeenAt:        &now,
			LastSeenAt:         &now,
		}

		pst.log.WithFields(logrus.Fields{
			"peer_id":      peerID,
			"client_agent": normalizeClientType(agentVersion),
		}).Info("Identified peer (new)")
		return
	}

	// Update peer-level info with most recent identification
	peer.ClientAgent = agentVersion
	peer.ClientType = normalizeClientType(agentVersion)
	peer.LastSeenAt = &now

	// Update the current session with identification details
	currentSession := pst.getCurrentSession(peer)
	if currentSession != nil {
		// Update identification even if the session is already disconnected
		// This handles cases where identification arrives after disconnection
		currentSession.IdentifiedAt = &now

		if currentSession.Disconnected {
			if normalizeClientType(agentVersion) != "unknown" {
				pst.log.WithFields(logrus.Fields{
					"peer_id":      peerID,
					"client_agent": normalizeClientType(agentVersion),
					"conn_count":   len(peer.ConnectionSessions),
				}).Debug("Identified peer (post disconnect)")
			}
		} else {
			pst.log.WithFields(logrus.Fields{
				"peer_id":      peerID,
				"client_agent": normalizeClientType(agentVersion),
				"conn_count":   len(peer.ConnectionSessions),
			}).Info("Identified peer")
		}
	} else {
		// This should be very rare - peer exists but has no sessions at all
		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
		}).Warn("Received identification for peer with no connection sessions")
	}
}

func (pst *PeerScoreTool) handlePeerScoreEvent(_ context.Context, event *host.TraceEvent) {
	// Extract peer ID from the event
	peerID := getPeerID(event)
	if peerID == "" {
		pst.log.Warn("handlePeerScoreEvent: could not extract peer ID from event")
		return
	}

	// Parse the peer score data from the payload
	scoreSnapshot, err := pst.parsePeerScoreFromPayload(event.Payload)
	if err != nil {
		pst.log.WithError(err).Warn("handlePeerScoreEvent: failed to parse peer score from payload")
		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	peer, exists := pst.peers[peerID]
	if !exists {
		// Create a new peer if we haven't seen them before
		now := time.Now()
		session := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
			PeerScores:   []PeerScoreSnapshot{*scoreSnapshot},
		}

		pst.peers[peerID] = &PeerStats{
			PeerID:             peerID,
			ClientType:         "unknown",
			ClientAgent:        "unknown",
			ConnectionSessions: []ConnectionSession{session},
			TotalConnections:   1,
			FirstSeenAt:        &now,
			LastSeenAt:         &now,
		}

		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
			"score":   scoreSnapshot.Score,
		}).Info("Peer score received")

		return
	}

	// Find the current session and add the score snapshot
	currentSession := pst.getCurrentSession(peer)
	if currentSession != nil {
		currentSession.PeerScores = append(currentSession.PeerScores, *scoreSnapshot)
		peer.LastSeenAt = &scoreSnapshot.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
			"score":   scoreSnapshot.Score,
		}).Info("Peer score received")
	} else {
		// No active session - create a new one for the score
		now := time.Now()
		newSession := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
			PeerScores:   []PeerScoreSnapshot{*scoreSnapshot},
		}

		peer.ConnectionSessions = append(peer.ConnectionSessions, newSession)
		peer.TotalConnections++
		peer.LastSeenAt = &scoreSnapshot.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
			"score":   scoreSnapshot.Score,
		}).Info("Peer score received")
	}
}

// parsePeerScoreFromPayload extracts peer score data from the event payload
func (pst *PeerScoreTool) parsePeerScoreFromPayload(payload interface{}) (*PeerScoreSnapshot, error) {
	// Try to parse as map[string]any (the format from composePeerScoreEventFromRawMap)
	if payloadMap, ok := payload.(map[string]any); ok {
		return pst.parsePeerScoreFromMap(payloadMap)
	}

	return nil, fmt.Errorf("unsupported payload type for peer score event: %T", payload)
}

// parsePeerScoreFromMap parses peer score data from a map[string]any payload
func (pst *PeerScoreTool) parsePeerScoreFromMap(payloadMap map[string]any) (*PeerScoreSnapshot, error) {
	snapshot := &PeerScoreSnapshot{
		Timestamp: time.Now(),
	}

	// Parse Score
	if score, ok := payloadMap["Score"].(float64); ok {
		snapshot.Score = score
	}

	// Parse AppSpecificScore
	if appScore, ok := payloadMap["AppSpecificScore"].(float64); ok {
		snapshot.AppSpecificScore = appScore
	}

	// Parse IPColocationFactor
	if ipFactor, ok := payloadMap["IPColocationFactor"].(float64); ok {
		snapshot.IPColocationFactor = ipFactor
	}

	// Parse BehaviourPenalty
	if penalty, ok := payloadMap["BehaviourPenalty"].(float64); ok {
		snapshot.BehaviourPenalty = penalty
	}

	// Parse Topics
	if topicsInterface, ok := payloadMap["Topics"]; ok {
		if topicsSlice, ok := topicsInterface.([]any); ok {
			for _, topicInterface := range topicsSlice {
				if topicMap, ok := topicInterface.(map[string]any); ok {
					topicScore := TopicScore{}

					if topic, ok := topicMap["Topic"].(string); ok {
						topicScore.Topic = topic
					}

					if timeInMesh, ok := topicMap["TimeInMesh"].(time.Duration); ok {
						topicScore.TimeInMesh = timeInMesh
					}

					if firstMsgDeliveries, ok := topicMap["FirstMessageDeliveries"].(float64); ok {
						topicScore.FirstMessageDeliveries = firstMsgDeliveries
					}

					if meshMsgDeliveries, ok := topicMap["MeshMessageDeliveries"].(float64); ok {
						topicScore.MeshMessageDeliveries = meshMsgDeliveries
					}

					if invalidMsgDeliveries, ok := topicMap["InvalidMessageDeliveries"].(float64); ok {
						topicScore.InvalidMessageDeliveries = invalidMsgDeliveries
					}

					snapshot.Topics = append(snapshot.Topics, topicScore)
				}
			}
		}
	}

	return snapshot, nil
}

// getCurrentSession returns the current active session for a peer, or nil if no active session exists
func (pst *PeerScoreTool) getCurrentSession(peer *PeerStats) *ConnectionSession {
	if len(peer.ConnectionSessions) == 0 {
		return nil
	}

	// Return the last session
	lastSession := &peer.ConnectionSessions[len(peer.ConnectionSessions)-1]
	return lastSession
}

func normalizeClientType(agent string) string {
	agent = strings.ToLower(agent)

	clients := []string{"lighthouse", "prysm", "nimbus", "lodestar", "grandine", "teku", "erigon", "caplin"}
	for _, client := range clients {
		if strings.Contains(agent, client) {
			return client
		}
	}

	// Extract first part before slash if present
	parts := strings.Split(agent, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return "unknown"
}

func getPeerID(event *host.TraceEvent) string {
	// For map-style payloads, try the map-style access
	if payload, ok := event.Payload.(map[string]any); ok {
		// Check if the event payload has a PeerID field
		if remotePeerID, found := payload["PeerID"]; found {
			return fmt.Sprintf("%v", remotePeerID)
		}

		// Some events use RemotePeer as a field name
		if remotePeerID, found := payload["RemotePeer"]; found {
			return fmt.Sprintf("%v", remotePeerID)
		}
	}

	// First try to extract PeerID using reflection - works for any struct with a PeerID field
	if event.Payload != nil {
		if peerID := extractPeerIDFromStruct(event.Payload); peerID != "" {
			return peerID
		}
	}

	// Return empty string if no peer ID can be extracted
	return ""
}

// extractPeerIDFromStruct uses reflection to extract a PeerID or RemotePeer field from any struct
func extractPeerIDFromStruct(payload interface{}) string {
	if payload == nil {
		return ""
	}

	val := reflect.ValueOf(payload)

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ""
		}
		val = val.Elem()
	}

	// Only works on structs
	if val.Kind() != reflect.Struct {
		return ""
	}

	// Try multiple field names in order of preference
	fieldNames := []string{"PeerID", "RemotePeer"}

	for _, fieldName := range fieldNames {
		peerIDField := val.FieldByName(fieldName)
		if !peerIDField.IsValid() {
			continue // Try next field name
		}

		// Try to call String() method if it exists (like peer.ID.String())
		if peerIDField.CanInterface() {
			peerIDInterface := peerIDField.Interface()

			// Check if it has a String() method
			stringMethod := reflect.ValueOf(peerIDInterface).MethodByName("String")
			if stringMethod.IsValid() && stringMethod.Type().NumIn() == 0 && stringMethod.Type().NumOut() == 1 {
				result := stringMethod.Call(nil)
				if len(result) > 0 {
					if str, ok := result[0].Interface().(string); ok {
						return str
					}
				}
			}

			// Fallback to fmt.Sprintf
			return fmt.Sprintf("%v", peerIDInterface)
		}
	}

	return ""
}
