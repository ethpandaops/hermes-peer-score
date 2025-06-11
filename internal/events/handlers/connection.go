package handlers

import (
	"context"
	"time"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/common"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
)

// ConnectionHandler handles peer connection events
type ConnectionHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
}

// NewConnectionHandler creates a new connection event handler
func NewConnectionHandler(tool common.ToolInterface, logger logrus.FieldLogger) *ConnectionHandler {
	return &ConnectionHandler{
		tool:   tool,
		logger: logger.WithField("handler", "connection"),
	}
}

// EventType returns the event type this handler manages
func (h *ConnectionHandler) EventType() string {
	return "CONNECTED"
}

// HandleEvent processes a connection event
func (h *ConnectionHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	peerID := common.GetPeerID(event)
	now := time.Now()

	h.logger.WithFields(logrus.Fields{
		"peer_id":      common.FormatShortPeerID(peerID),
	}).Debug("Processing connection event")

	// Check if peer already exists
	_, exists := h.tool.GetPeer(peerID)
	if !exists {
		// Create new peer
		h.tool.CreatePeer(peerID)
		h.logger.WithField("peer_id", common.FormatShortPeerID(peerID)).Info("New peer connection")
	}

	// Update peer with connection information
	h.tool.UpdatePeer(peerID, func(p interface{}) {
		if peerStats, ok := p.(*peer.Stats); ok {
			h.updatePeerConnection(peerStats, now)
		}
	})

	// Increment connection event count
	h.tool.IncrementEventCount(peerID, "CONNECTED")
	
	return nil
}

// updatePeerConnection updates peer connection information
func (h *ConnectionHandler) updatePeerConnection(peerStats *peer.Stats, connectedAt time.Time) {
	// Update last seen time
	peerStats.LastSeenAt = &connectedAt

	// Start a new connection session
	session := peer.ConnectionSession{
		ConnectedAt:  &connectedAt,
		MessageCount: 0,
		Disconnected: false,
		PeerScores:   []peer.PeerScoreSnapshot{},
		GoodbyeEvents: []peer.GoodbyeEvent{},
		MeshEvents:   []peer.MeshEvent{},
	}

	peerStats.ConnectionSessions = append(peerStats.ConnectionSessions, session)
	peerStats.TotalConnections++

	h.logger.WithFields(logrus.Fields{
		"peer_id":      common.FormatShortPeerID(peerStats.PeerID),
		"session_count": len(peerStats.ConnectionSessions),
	}).Debug("Updated peer connection")
}