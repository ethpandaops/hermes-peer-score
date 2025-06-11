package handlers

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
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
	data, err := libp2p.TraceEventToConnected(event)
	if err != nil {
		h.logger.WithError(err).Error("failed to convert event to connected event")
		return err
	}

	peerID := data.RemotePeer.GetValue()
	now := time.Now()
	clientAgent := data.AgentVersion.GetValue()
	clientType := common.NormalizeClientType(clientAgent)

	h.logger.WithFields(logrus.Fields{
		"peer_id":      common.FormatShortPeerID(peerID),
		"client_type":  clientType,
		"client_agent": clientAgent,
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
			h.updatePeerConnection(peerStats, clientType, clientAgent, now)
		}
	})

	// Increment connection event count
	h.tool.IncrementEventCount(peerID, "CONNECTED")
	
	return nil
}

// updatePeerConnection updates peer connection information
func (h *ConnectionHandler) updatePeerConnection(peerStats *peer.Stats, clientType, clientAgent string, connectedAt time.Time) {
	// Update client information if not already set
	if peerStats.ClientType == "" {
		peerStats.ClientType = clientType
	}
	if peerStats.ClientAgent == "" {
		peerStats.ClientAgent = clientAgent
	}

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
		"client_type":  clientType,
		"session_count": len(peerStats.ConnectionSessions),
	}).Debug("Updated peer connection")
}