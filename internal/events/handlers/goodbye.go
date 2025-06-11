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

// GoodbyeHandler handles goodbye message events
type GoodbyeHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
	parser *parsers.DefaultParser
}

// NewGoodbyeHandler creates a new goodbye event handler
func NewGoodbyeHandler(tool common.ToolInterface, logger logrus.FieldLogger) *GoodbyeHandler {
	return &GoodbyeHandler{
		tool:   tool,
		logger: logger.WithField("handler", "goodbye"),
		parser: &parsers.DefaultParser{},
	}
}

// EventType returns the event type this handler manages
func (h *GoodbyeHandler) EventType() string {
	return "HANDLE_GOODBYE"
}

// HandleEvent processes a goodbye event
func (h *GoodbyeHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		h.logger.Error("failed to convert goodbye payload to map[string]interface{}")
		return nil
	}

	peerID := common.GetPeerID(event)
	if peerID == constants.Unknown {
		h.logger.Error("goodbye event missing or invalid peer ID")
		return nil
	}

	// Parse goodbye data
	goodbyeData, err := h.parser.ParseGoodbyeFromMap(payload)
	if err != nil {
		h.logger.WithError(err).WithField("peer_id", common.FormatShortPeerID(peerID)).Error("failed to parse goodbye data")
		return nil
	}

	h.logger.WithFields(logrus.Fields{
		"peer_id": common.FormatShortPeerID(peerID),
		"code":    goodbyeData.Code,
		"reason":  goodbyeData.Reason,
	}).Debug("Processing goodbye event")

	// Update peer with goodbye event
	h.tool.UpdatePeer(peerID, func(p interface{}) {
		if peerStats, ok := p.(*peer.Stats); ok {
			h.addGoodbyeEvent(peerStats, goodbyeData)
		}
	})

	// Increment goodbye event count
	h.tool.IncrementEventCount(peerID, "HANDLE_GOODBYE")

	return nil
}

// addGoodbyeEvent adds a goodbye event to the peer's current session
func (h *GoodbyeHandler) addGoodbyeEvent(peerStats *peer.Stats, goodbyeData *parsers.GoodbyeData) {
	// Find the most recent active session
	for i := len(peerStats.ConnectionSessions) - 1; i >= 0; i-- {
		session := &peerStats.ConnectionSessions[i]
		if !session.Disconnected {
			// Add goodbye event to this session
			goodbyeEvent := peer.GoodbyeEvent{
				Code:      goodbyeData.Code,
				Reason:    goodbyeData.Reason,
				Timestamp: goodbyeData.Timestamp,
			}
			
			session.GoodbyeEvents = append(session.GoodbyeEvents, goodbyeEvent)
			
			h.logger.WithFields(logrus.Fields{
				"peer_id":   common.FormatShortPeerID(peerStats.PeerID),
				"code":      goodbyeData.Code,
				"reason":    goodbyeData.Reason,
				"timestamp": goodbyeData.Timestamp,
			}).Debug("Added goodbye event")
			return
		}
	}
	
	h.logger.WithField("peer_id", common.FormatShortPeerID(peerStats.PeerID)).Warn("No active session found for goodbye event")
}