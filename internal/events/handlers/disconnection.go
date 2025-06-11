package handlers

import (
	"context"
	"time"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/common"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
)

// DisconnectionHandler handles peer disconnection events
type DisconnectionHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
}

// NewDisconnectionHandler creates a new disconnection event handler
func NewDisconnectionHandler(tool common.ToolInterface, logger logrus.FieldLogger) *DisconnectionHandler {
	return &DisconnectionHandler{
		tool:   tool,
		logger: logger.WithField("handler", "disconnection"),
	}
}

// EventType returns the event type this handler manages
func (h *DisconnectionHandler) EventType() string {
	return "DISCONNECTED"
}

// HandleEvent processes a disconnection event
func (h *DisconnectionHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	peerID := common.GetPeerID(event)
	now := time.Now()

	h.logger.WithField("peer_id", common.FormatShortPeerID(peerID)).Debug("Processing disconnection event")

	// Check if peer exists
	_, exists := h.tool.GetPeer(peerID)
	if !exists {
		h.logger.WithField("peer_id", common.FormatShortPeerID(peerID)).Warn("Received disconnection event for peer we've never seen")
		return nil
	}

	// Update peer to mark current session as disconnected
	h.tool.UpdatePeer(peerID, func(p interface{}) {
		if stats, ok := p.(*peer.Stats); ok {
			h.markSessionDisconnected(stats, now)
		}
	})

	// Increment disconnection event count
	h.tool.IncrementEventCount(peerID, "DISCONNECTED")

	h.logger.WithField("peer_id", common.FormatShortPeerID(peerID)).Info("Peer disconnected")
	return nil
}

// markSessionDisconnected marks the current active session as disconnected
func (h *DisconnectionHandler) markSessionDisconnected(peerStats *peer.Stats, disconnectedAt time.Time) {
	// Find the most recent active session and mark it as disconnected
	for i := len(peerStats.ConnectionSessions) - 1; i >= 0; i-- {
		session := &peerStats.ConnectionSessions[i]
		if !session.Disconnected {
			session.Disconnected = true
			session.DisconnectedAt = &disconnectedAt
			
			// Calculate session duration
			if session.ConnectedAt != nil {
				duration := disconnectedAt.Sub(*session.ConnectedAt)
				session.Duration = &duration
			}
			
			h.logger.WithFields(logrus.Fields{
				"peer_id":        common.FormatShortPeerID(peerStats.PeerID),
				"disconnected_at": disconnectedAt,
				"session_duration": session.Duration,
			}).Debug("Marked session as disconnected")
			return
		}
	}
	
	h.logger.WithField("peer_id", common.FormatShortPeerID(peerStats.PeerID)).Warn("No active session found to disconnect")
}