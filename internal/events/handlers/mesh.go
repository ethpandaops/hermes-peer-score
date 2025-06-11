package handlers

import (
	"context"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/constants"
	"github.com/ethpandaops/hermes-peer-score/internal/common"
	"github.com/ethpandaops/hermes-peer-score/internal/events/parsers"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
)

// GraftHandler handles GRAFT mesh events
type GraftHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
	parser *parsers.DefaultParser
}

// NewGraftHandler creates a new GRAFT event handler
func NewGraftHandler(tool common.ToolInterface, logger logrus.FieldLogger) *GraftHandler {
	return &GraftHandler{
		tool:   tool,
		logger: logger.WithField("handler", "graft"),
		parser: &parsers.DefaultParser{},
	}
}

// EventType returns the event type this handler manages
func (h *GraftHandler) EventType() string {
	return "GRAFT"
}

// HandleEvent processes a GRAFT event
func (h *GraftHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	return h.handleMeshEvent(event, "GRAFT")
}

// PruneHandler handles PRUNE mesh events
type PruneHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
	parser *parsers.DefaultParser
}

// NewPruneHandler creates a new PRUNE event handler
func NewPruneHandler(tool common.ToolInterface, logger logrus.FieldLogger) *PruneHandler {
	return &PruneHandler{
		tool:   tool,
		logger: logger.WithField("handler", "prune"),
		parser: &parsers.DefaultParser{},
	}
}

// EventType returns the event type this handler manages
func (h *PruneHandler) EventType() string {
	return "PRUNE"
}

// HandleEvent processes a PRUNE event
func (h *PruneHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	return h.handleMeshEvent(event, "PRUNE")
}

// handleMeshEvent is shared logic for both GRAFT and PRUNE events
func (h *GraftHandler) handleMeshEvent(event *host.TraceEvent, eventType string) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		h.logger.Errorf("failed to convert %s payload to map[string]interface{}", eventType)
		return nil
	}

	peerID := common.GetPeerID(event)
	if peerID == constants.Unknown {
		h.logger.Errorf("%s event missing or invalid peer ID", eventType)
		return nil
	}

	// Parse mesh data
	meshData, err := h.parser.ParseMeshFromMap(payload, eventType)
	if err != nil {
		h.logger.WithError(err).WithField("peer_id", common.FormatShortPeerID(peerID)).Errorf("failed to parse %s data", eventType)
		return nil
	}

	h.logger.WithFields(logrus.Fields{
		"peer_id":   common.FormatShortPeerID(peerID),
		"type":      meshData.Type,
		"direction": meshData.Direction,
		"topic":     meshData.Topic,
		"reason":    meshData.Reason,
	}).Debugf("Processing %s event", eventType)

	// Update peer with mesh event
	h.tool.UpdatePeer(peerID, func(p interface{}) {
		if peerStats, ok := p.(*peer.Stats); ok {
			addMeshEvent(h.logger, peerStats, meshData)
		}
	})

	// Increment mesh event count
	h.tool.IncrementEventCount(peerID, eventType)

	return nil
}

// handleMeshEvent is shared logic for PRUNE events
func (h *PruneHandler) handleMeshEvent(event *host.TraceEvent, eventType string) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		h.logger.Errorf("failed to convert %s payload to map[string]interface{}", eventType)
		return nil
	}

	peerID := common.GetPeerID(event)
	if peerID == constants.Unknown {
		h.logger.Errorf("%s event missing or invalid peer ID", eventType)
		return nil
	}

	// Parse mesh data
	meshData, err := h.parser.ParseMeshFromMap(payload, eventType)
	if err != nil {
		h.logger.WithError(err).WithField("peer_id", common.FormatShortPeerID(peerID)).Errorf("failed to parse %s data", eventType)
		return nil
	}

	h.logger.WithFields(logrus.Fields{
		"peer_id":   common.FormatShortPeerID(peerID),
		"type":      meshData.Type,
		"direction": meshData.Direction,
		"topic":     meshData.Topic,
		"reason":    meshData.Reason,
	}).Debugf("Processing %s event", eventType)

	// Update peer with mesh event
	h.tool.UpdatePeer(peerID, func(p interface{}) {
		if peerStats, ok := p.(*peer.Stats); ok {
			addMeshEvent(h.logger, peerStats, meshData)
		}
	})

	// Increment mesh event count
	h.tool.IncrementEventCount(peerID, eventType)

	return nil
}

// addMeshEvent adds a mesh event to the peer's current session (shared implementation)
func addMeshEvent(logger logrus.FieldLogger, peerStats *peer.Stats, meshData *parsers.MeshData) {
	// Find the most recent active session
	for i := len(peerStats.ConnectionSessions) - 1; i >= 0; i-- {
		session := &peerStats.ConnectionSessions[i]
		if !session.Disconnected {
			// Add mesh event to this session
			meshEvent := peer.MeshEvent{
				Type:      meshData.Type,
				Direction: meshData.Direction,
				Topic:     meshData.Topic,
				Reason:    meshData.Reason,
				Timestamp: meshData.Timestamp,
			}
			
			session.MeshEvents = append(session.MeshEvents, meshEvent)
			
			logger.WithFields(logrus.Fields{
				"peer_id":   common.FormatShortPeerID(peerStats.PeerID),
				"type":      meshData.Type,
				"direction": meshData.Direction,
				"topic":     meshData.Topic,
				"timestamp": meshData.Timestamp,
			}).Debug("Added mesh event")
			return
		}
	}
	
	logger.WithField("peer_id", common.FormatShortPeerID(peerStats.PeerID)).Warn("No active session found for mesh event")
}