package handlers

import (
	"context"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/common"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
)

// StatusHandler handles status request events
type StatusHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
}

// NewStatusHandler creates a new status event handler
func NewStatusHandler(tool common.ToolInterface, logger logrus.FieldLogger) *StatusHandler {
	return &StatusHandler{
		tool:   tool,
		logger: logger.WithField("handler", "status"),
	}
}

// EventType returns the event type this handler manages
func (h *StatusHandler) EventType() string {
	return "REQUEST_STATUS"
}

// HandleEvent processes a status request event
func (h *StatusHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		h.logger.Error("failed to convert status payload to map[string]interface{}")
		return nil
	}

	peerID, ok := payload["PeerID"].(string)
	if !ok {
		h.logger.WithFields(logrus.Fields{
			"payload": payload,
		}).Error("status event missing or invalid PeerID")
		return nil
	}

	h.logger.WithField("peer_id", common.FormatShortPeerID(peerID)).Debug("Processing status event")

	// Check if peer exists, create if not
	_, exists := h.tool.GetPeer(peerID)
	if !exists {
		h.tool.CreatePeer(peerID)
		h.logger.WithField("peer_id", common.FormatShortPeerID(peerID)).Debug("Created peer from status event")
	}

	// Update peer with status information
	h.tool.UpdatePeer(peerID, func(p interface{}) {
		if peerStats, ok := p.(*peer.Stats); ok {
			h.handleStatusUpdate(peerStats, payload)
		}
	})

	// Increment status event count
	h.tool.IncrementEventCount(peerID, "REQUEST_STATUS")

	return nil
}

// handleStatusUpdate processes the status update for a peer
func (h *StatusHandler) handleStatusUpdate(peerStats *peer.Stats, payload map[string]interface{}) {
	// Extract and process status information
	success := true // Default assumption
	if err, hasErr := payload["Error"]; hasErr && err != nil {
		success = false
		peerStats.FailedHandshakes++
	} else {
		peerStats.SuccessfulHandshakes++
	}

	// Extract client identification information from AgentVersion
	if agentVersion, ok := payload["AgentVersion"].(string); ok && agentVersion != "" {
		clientType := common.NormalizeClientType(agentVersion)

		// Update client information if not already set or if it's the default "unknown"
		if peerStats.ClientType == "" || peerStats.ClientType == "unknown" {
			peerStats.ClientType = clientType
		}
		if peerStats.ClientAgent == "" {
			peerStats.ClientAgent = agentVersion
		}

		h.logger.WithFields(logrus.Fields{
			"peer_id":      common.FormatShortPeerID(peerStats.PeerID),
			"client_type":  clientType,
			"client_agent": agentVersion,
		}).Info("Peer identified")
	}

	h.logger.WithFields(logrus.Fields{
		"peer_id": common.FormatShortPeerID(peerStats.PeerID),
		"success": success,
	}).Debug("Handled status update")
}
