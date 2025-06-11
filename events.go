package main

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
)

const unknown = "unknown"

func (pst *PeerScoreTool) handleHermesEvent(ctx context.Context, event *host.TraceEvent) error {
	// Add validation mode context to all event logging
	eventLogger := pst.log.WithFields(logrus.Fields{
		"validation_mode": pst.config.ValidationMode,
		"event_type":      event.Type,
	})

	switch event.Type {
	case "CONNECTED":
		pst.handleConnectionEvent(ctx, event, eventLogger)
	case "DISCONNECTED":
		pst.handleDisconnectionEvent(ctx, event, eventLogger)
	case "REQUEST_STATUS":
		pst.handleStatusEvent(ctx, event, eventLogger)
	case "PEERSCORE":
		pst.handlePeerScoreEvent(ctx, event, eventLogger)
	case "HANDLE_GOODBYE":
		pst.handleGoodbyeEvent(ctx, event, eventLogger)
	case "GRAFT":
		pst.handleGraftEvent(ctx, event, eventLogger)
	case "PRUNE":
		pst.handlePruneEvent(ctx, event, eventLogger)
	default:
		eventLogger.WithField("event_type", event.Type).Debug("Unhandled event type")
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

func (pst *PeerScoreTool) handleConnectionEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
	data, err := libp2p.TraceEventToConnected(event)
	if err != nil {
		logger.WithError(err).Error("failed to convert event to connected event")

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

		logger.WithField("peer_id", peerID).Info("New peer connection")

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

		logger.WithFields(logrus.Fields{
			"peer_id":    peerID,
			"conn_count": len(peer.ConnectionSessions),
		}).Info("New peer connection")
	} else {
		// Duplicate connection event for active session. This is normal with libp2p.
		logger.WithFields(logrus.Fields{
			"peer_id": peerID,
		}).Debug("Duplicate peer connection event")
	}
}

func (pst *PeerScoreTool) handleDisconnectionEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
	data, err := libp2p.TraceEventToDisconnected(event)
	if err != nil {
		logger.WithError(err).Error("failed to convert event to disconnected event")

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

func (pst *PeerScoreTool) handleStatusEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
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
			if normalizeClientType(agentVersion) != unknown {
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

func (pst *PeerScoreTool) handlePeerScoreEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
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

	// Check if we should ignore this event for a disconnected peer
	if pst.shouldIgnoreEventForDisconnectedPeer(peerID, "PEERSCORE", scoreSnapshot.Timestamp) {
		return
	}

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
			ClientType:         unknown,
			ClientAgent:        unknown,
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

func (pst *PeerScoreTool) handleGoodbyeEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
	peerID := getPeerID(event)
	if peerID == "" {
		pst.log.Warn("handleGoodbyeEvent: could not extract peer ID from event")

		return
	}

	// Debug logging to understand the payload structure
	pst.log.WithFields(logrus.Fields{
		"peer_id":      peerID,
		"payload_type": fmt.Sprintf("%T", event.Payload),
		"payload":      event.Payload,
	}).Debug("handleGoodbyeEvent: received goodbye event")

	// Parse the goodbye data from the payload
	goodbyeEvent, err := pst.parseGoodbyeFromPayload(event.Payload)
	if err != nil {
		pst.log.WithFields(logrus.Fields{
			"peer_id":      peerID,
			"payload_type": fmt.Sprintf("%T", event.Payload),
			"payload":      event.Payload,
		}).WithError(err).Warn("handleGoodbyeEvent: failed to parse goodbye from payload")

		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	// Check if we should ignore this event for a disconnected peer
	if pst.shouldIgnoreEventForDisconnectedPeer(peerID, "HANDLE_GOODBYE", goodbyeEvent.Timestamp) {
		return
	}

	peer, exists := pst.peers[peerID]
	if !exists {
		// Create a new peer if we haven't seen them before
		now := time.Now()
		session := ConnectionSession{
			ConnectedAt:   &now,
			Disconnected:  false,
			GoodbyeEvents: []GoodbyeEvent{*goodbyeEvent},
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
			"code":    goodbyeEvent.Code,
			"reason":  goodbyeEvent.Reason,
		}).Info("Goodbye message received (new peer)")

		return
	}

	// Find the current session and add the goodbye event
	currentSession := pst.getCurrentSession(peer)
	if currentSession != nil {
		currentSession.GoodbyeEvents = append(currentSession.GoodbyeEvents, *goodbyeEvent)
		peer.LastSeenAt = &goodbyeEvent.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
			"code":    goodbyeEvent.Code,
			"reason":  goodbyeEvent.Reason,
		}).Info("Goodbye message received")
	} else {
		// No active session - create a new one for the goodbye event
		now := time.Now()
		newSession := ConnectionSession{
			ConnectedAt:   &now,
			Disconnected:  false,
			GoodbyeEvents: []GoodbyeEvent{*goodbyeEvent},
		}

		peer.ConnectionSessions = append(peer.ConnectionSessions, newSession)
		peer.TotalConnections++
		peer.LastSeenAt = &goodbyeEvent.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id": peerID,
			"code":    goodbyeEvent.Code,
			"reason":  goodbyeEvent.Reason,
		}).Info("Goodbye message received (new session)")
	}
}

// parsePeerScoreFromPayload extracts peer score data from the event payload.
func (pst *PeerScoreTool) parsePeerScoreFromPayload(payload interface{}) (*PeerScoreSnapshot, error) {
	// Try to parse as map[string]any (the format from composePeerScoreEventFromRawMap)
	if payloadMap, ok := payload.(map[string]any); ok {
		return pst.parsePeerScoreFromMap(payloadMap)
	}

	return nil, fmt.Errorf("unsupported payload type for peer score event: %T", payload)
}

// parsePeerScoreFromMap parses peer score data from a map[string]any payload.
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

// getCurrentSession returns the current active session for a peer, or nil if no active session exists.
func (pst *PeerScoreTool) getCurrentSession(peer *PeerStats) *ConnectionSession {
	if len(peer.ConnectionSessions) == 0 {
		return nil
	}

	// Return the last session
	return &peer.ConnectionSessions[len(peer.ConnectionSessions)-1]
}

// shouldIgnoreEventForDisconnectedPeer checks if we should ignore an event for a peer that's already disconnected.
//
// This function addresses a timing issue where peer score events and goodbye events can be received
// after a peer has already been marked as disconnected. This happens because:
//
// 1. GossipSub peer scoring runs on a timer (every 5 seconds) and is asynchronous to connection events
// 2. The peer scoring system doesn't immediately check connection status before generating snapshots
// 3. Connection state changes and event processing can have race conditions
// 4. Stream processing and libp2p event handling occur in separate goroutines
//
// Without this filtering, we would record "impossible" data like peer scores timestamped after
// disconnection, leading to data integrity issues and incorrect analysis of peer behavior.
//
// This function only checks the CURRENT session's disconnect status. If a peer reconnects later
// (creating a new session), those events will be processed normally since getCurrentSession()
// returns the most recent session, which would be the new connected session.
//
// Returns true if the event should be ignored (peer is disconnected), false if it should be processed.
func (pst *PeerScoreTool) shouldIgnoreEventForDisconnectedPeer(peerID string, eventType string, eventTime time.Time) bool {
	peer, exists := pst.peers[peerID]
	if !exists {
		return false // New peer, don't ignore
	}

	currentSession := pst.getCurrentSession(peer)
	if currentSession == nil {
		return false // No session, don't ignore
	}

	if currentSession.Disconnected {
		pst.log.WithFields(logrus.Fields{
			"peer_id":         peerID,
			"event_type":      eventType,
			"disconnected_at": currentSession.DisconnectedAt,
			"event_timestamp": eventTime,
		}).Debug("Ignoring event for disconnected peer")

		return true
	}

	return false
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

// extractPeerIDFromStruct uses reflection to extract a PeerID or RemotePeer field from any struct.
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

// parseGoodbyeFromPayload extracts goodbye data from the event payload.
func (pst *PeerScoreTool) parseGoodbyeFromPayload(payload interface{}) (*GoodbyeEvent, error) {
	// Try to parse as map[string]any (the format from hermes reqresp handler)
	if payloadMap, ok := payload.(map[string]any); ok {
		// Log all keys in the payload for debugging
		keys := make([]string, 0, len(payloadMap))
		for k := range payloadMap {
			keys = append(keys, k)
		}

		pst.log.WithFields(logrus.Fields{
			"payload_keys": keys,
			"payload_map":  payloadMap,
		}).Debug("parseGoodbyeFromPayload: parsing map payload")

		return pst.parseGoodbyeFromMap(payloadMap)
	}

	return nil, fmt.Errorf("unsupported payload type for goodbye event: %T", payload)
}

// parseGoodbyeFromMap parses goodbye data from a map[string]any payload.
func (pst *PeerScoreTool) parseGoodbyeFromMap(payloadMap map[string]any) (*GoodbyeEvent, error) {
	goodbyeEvent := &GoodbyeEvent{
		Timestamp: time.Now(),
	}

	// Check if this is an error payload (from stream reset, etc.)
	if errorValue, hasError := payloadMap["Error"]; hasError && errorValue != nil {
		// This is an error case - create a goodbye event with error information
		errorStr := fmt.Sprintf("%v", errorValue)
		goodbyeEvent.Code = 0 // Use 0 to indicate unknown/error case
		goodbyeEvent.Reason = fmt.Sprintf("connection error: %s", errorStr)

		pst.log.WithFields(logrus.Fields{
			"error": errorStr,
		}).Debug("Parsing goodbye event from error payload")

		return goodbyeEvent, nil
	}

	// Parse Code - handle various types that might come from hermes.
	codeValue, exists := payloadMap["Code"]
	if !exists {
		return nil, fmt.Errorf("missing Code field in goodbye payload")
	}

	// Debug logging to understand the actual type
	pst.log.WithFields(logrus.Fields{
		"code_value": codeValue,
		"code_type":  fmt.Sprintf("%T", codeValue),
	}).Debug("Parsing goodbye code from payload")

	switch v := codeValue.(type) {
	case uint64:
		goodbyeEvent.Code = v
	case int64:
		//nolint:gosec // fine.
		goodbyeEvent.Code = uint64(v)
	case int:
		//nolint:gosec // fine.
		goodbyeEvent.Code = uint64(v)
	case float64:
		goodbyeEvent.Code = uint64(v)
	case float32:
		goodbyeEvent.Code = uint64(v)
	default:
		// Try to extract using fmt.Sprintf and parse
		if str := fmt.Sprintf("%v", v); str != "" {
			if parsed, err := strconv.ParseUint(str, 10, 64); err == nil {
				goodbyeEvent.Code = parsed
			} else {
				return nil, fmt.Errorf("invalid Code field type %T with value %v in goodbye payload", v, v)
			}
		} else {
			return nil, fmt.Errorf("invalid Code field type %T in goodbye payload", v)
		}
	}

	// Parse Reason
	if reason, ok := payloadMap["Reason"].(string); ok {
		goodbyeEvent.Reason = reason
	} else {
		goodbyeEvent.Reason = unknown
	}

	return goodbyeEvent, nil
}

func (pst *PeerScoreTool) handleGraftEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
	peerID := getPeerID(event)
	if peerID == "" {
		pst.log.Warn("handleGraftEvent: could not extract peer ID from event")

		return
	}

	// Parse the mesh event data from the payload
	meshEvent, err := pst.parseMeshEventFromPayload(event.Payload, "GRAFT", event.Timestamp)
	if err != nil {
		pst.log.WithFields(logrus.Fields{
			"peer_id":      peerID,
			"payload_type": fmt.Sprintf("%T", event.Payload),
			"payload":      event.Payload,
		}).WithError(err).Warn("handleGraftEvent: failed to parse GRAFT from payload")

		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	// Check if we should ignore this event for a disconnected peer
	if pst.shouldIgnoreEventForDisconnectedPeer(peerID, "GRAFT", meshEvent.Timestamp) {
		return
	}

	peer, exists := pst.peers[peerID]
	if !exists {
		// Create a new peer if we haven't seen them before
		now := time.Now()
		session := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
			MeshEvents:   []MeshEvent{*meshEvent},
		}

		pst.peers[peerID] = &PeerStats{
			PeerID:             peerID,
			ClientType:         unknown,
			ClientAgent:        unknown,
			ConnectionSessions: []ConnectionSession{session},
			TotalConnections:   1,
			FirstSeenAt:        &now,
			LastSeenAt:         &now,
		}

		pst.log.WithFields(logrus.Fields{
			"peer_id":   peerID,
			"topic":     meshEvent.Topic,
			"direction": meshEvent.Direction,
		}).Info("GRAFT event received (new peer)")

		return
	}

	// Find the current session and add the mesh event
	currentSession := pst.getCurrentSession(peer)
	if currentSession != nil {
		currentSession.MeshEvents = append(currentSession.MeshEvents, *meshEvent)
		peer.LastSeenAt = &meshEvent.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id":   peerID,
			"topic":     meshEvent.Topic,
			"direction": meshEvent.Direction,
		}).Info("GRAFT event received")
	} else {
		// No active session - create a new one for the mesh event
		now := time.Now()
		newSession := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
			MeshEvents:   []MeshEvent{*meshEvent},
		}

		peer.ConnectionSessions = append(peer.ConnectionSessions, newSession)
		peer.TotalConnections++
		peer.LastSeenAt = &meshEvent.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id":   peerID,
			"topic":     meshEvent.Topic,
			"direction": meshEvent.Direction,
		}).Info("GRAFT event received (new session)")
	}
}

func (pst *PeerScoreTool) handlePruneEvent(_ context.Context, event *host.TraceEvent, logger logrus.FieldLogger) {
	peerID := getPeerID(event)
	if peerID == "" {
		pst.log.Warn("handlePruneEvent: could not extract peer ID from event")

		return
	}

	// Parse the mesh event data from the payload
	meshEvent, err := pst.parseMeshEventFromPayload(event.Payload, "PRUNE", event.Timestamp)
	if err != nil {
		pst.log.WithFields(logrus.Fields{
			"peer_id":      peerID,
			"payload_type": fmt.Sprintf("%T", event.Payload),
			"payload":      event.Payload,
		}).WithError(err).Warn("handlePruneEvent: failed to parse PRUNE from payload")

		return
	}

	pst.peersMu.Lock()
	defer pst.peersMu.Unlock()

	// Check if we should ignore this event for a disconnected peer
	if pst.shouldIgnoreEventForDisconnectedPeer(peerID, "PRUNE", meshEvent.Timestamp) {
		return
	}

	peer, exists := pst.peers[peerID]
	if !exists {
		// Create a new peer if we haven't seen them before
		now := time.Now()
		session := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
			MeshEvents:   []MeshEvent{*meshEvent},
		}

		pst.peers[peerID] = &PeerStats{
			PeerID:             peerID,
			ClientType:         unknown,
			ClientAgent:        unknown,
			ConnectionSessions: []ConnectionSession{session},
			TotalConnections:   1,
			FirstSeenAt:        &now,
			LastSeenAt:         &now,
		}

		pst.log.WithFields(logrus.Fields{
			"peer_id":   peerID,
			"topic":     meshEvent.Topic,
			"direction": meshEvent.Direction,
			"reason":    meshEvent.Reason,
		}).Info("PRUNE event received (new peer)")

		return
	}

	// Find the current session and add the mesh event
	currentSession := pst.getCurrentSession(peer)
	if currentSession != nil {
		currentSession.MeshEvents = append(currentSession.MeshEvents, *meshEvent)
		peer.LastSeenAt = &meshEvent.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id":   peerID,
			"topic":     meshEvent.Topic,
			"direction": meshEvent.Direction,
			"reason":    meshEvent.Reason,
		}).Info("PRUNE event received")
	} else {
		// No active session - create a new one for the mesh event
		now := time.Now()
		newSession := ConnectionSession{
			ConnectedAt:  &now,
			Disconnected: false,
			MeshEvents:   []MeshEvent{*meshEvent},
		}

		peer.ConnectionSessions = append(peer.ConnectionSessions, newSession)
		peer.TotalConnections++
		peer.LastSeenAt = &meshEvent.Timestamp

		pst.log.WithFields(logrus.Fields{
			"peer_id":   peerID,
			"topic":     meshEvent.Topic,
			"direction": meshEvent.Direction,
			"reason":    meshEvent.Reason,
		}).Info("PRUNE event received (new session)")
	}
}

// parseMeshEventFromPayload extracts mesh event data from the event payload.
func (pst *PeerScoreTool) parseMeshEventFromPayload(payload interface{}, eventType string, eventTimestamp time.Time) (*MeshEvent, error) {
	// Try to parse as map[string]any (the format from hermes gossipsub tracer)
	if payloadMap, ok := payload.(map[string]any); ok {
		return pst.parseMeshEventFromMap(payloadMap, eventType, eventTimestamp)
	}

	return nil, fmt.Errorf("unsupported payload type for mesh event: %T", payload)
}

// parseMeshEventFromMap parses mesh event data from a map[string]any payload.
func (pst *PeerScoreTool) parseMeshEventFromMap(payloadMap map[string]any, eventType string, eventTimestamp time.Time) (*MeshEvent, error) {
	meshEvent := &MeshEvent{
		Timestamp: eventTimestamp,
		Type:      eventType,
		Direction: unknown, // Will determine based on context
	}

	// Parse Topic
	if topic, ok := payloadMap["Topic"].(string); ok {
		meshEvent.Topic = topic
	} else {
		return nil, fmt.Errorf("missing or invalid Topic field in mesh event payload")
	}

	// GRAFT and PRUNE events represent OUR mesh management actions from OUR perspective:
	// GRAFT = WE are adding the specified peer TO our mesh for this topic
	// PRUNE = WE are removing the specified peer FROM our mesh for this topic
	//
	// The PeerID in the event payload is the peer WE are acting upon, not a peer acting on us.
	// This is our node making local mesh topology decisions for GossipSub.
	meshEvent.Direction = "outgoing"

	// Parse Reason for PRUNE events (may not be present for GRAFT)
	if reason, ok := payloadMap["Reason"].(string); ok {
		meshEvent.Reason = reason
	}

	return meshEvent, nil
}
