package handlers

import (
	"context"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/common"
	"github.com/ethpandaops/hermes-peer-score/internal/events/parsers"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
)

// PeerScoreHandler handles peer score events.
type PeerScoreHandler struct {
	tool   common.ToolInterface
	logger logrus.FieldLogger
	parser *parsers.DefaultParser
}

// NewPeerScoreHandler creates a new peer score event handler.
func NewPeerScoreHandler(tool common.ToolInterface, logger logrus.FieldLogger) *PeerScoreHandler {
	return &PeerScoreHandler{
		tool:   tool,
		logger: logger.WithField("handler", "peer_score"),
		parser: &parsers.DefaultParser{},
	}
}

// EventType returns the event type this handler manages.
func (h *PeerScoreHandler) EventType() string {
	return "PEERSCORE"
}

// HandleEvent processes a peer score event.
func (h *PeerScoreHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		h.logger.Error("failed to convert peer score payload to map[string]interface{}")

		return nil
	}

	peerID, ok := payload["PeerID"].(string)
	if !ok {
		h.logger.Error("peer score event missing or invalid PeerID")

		return nil
	}

	// Parse the peer score data
	scoreData, err := h.parser.ParsePeerScoreFromMap(payload)
	if err != nil {
		h.logger.WithError(err).WithField("peer_id", common.FormatShortPeerID(peerID)).Error("failed to parse peer score data")

		return nil
	}

	h.logger.WithFields(logrus.Fields{
		"peer_id": common.FormatShortPeerID(peerID),
		"score":   scoreData.Score,
	}).Debug("Processing peer score event")

	// Update or create peer with new score data
	h.tool.UpdateOrCreatePeer(peerID, func(p interface{}) {
		if peerStats, ok := p.(*peer.Stats); ok {
			h.addPeerScore(peerStats, scoreData)
		}
	})

	// Increment peer score event count
	h.tool.IncrementEventCount(peerID, "PEERSCORE")

	// Increment message count for this session
	h.tool.IncrementMessageCount(peerID)

	return nil
}

// addPeerScore adds a peer score snapshot to the peer's current session.
func (h *PeerScoreHandler) addPeerScore(peerStats *peer.Stats, scoreData *parsers.PeerScoreData) {
	// Find the most recent active session
	for i := len(peerStats.ConnectionSessions) - 1; i >= 0; i-- {
		session := &peerStats.ConnectionSessions[i]
		if !session.Disconnected {
			// Add score to this session
			scoreSnapshot := peer.PeerScoreSnapshot{
				Score:              scoreData.Score,
				Timestamp:          scoreData.Timestamp,
				AppSpecificScore:   scoreData.AppSpecificScore,
				IPColocationFactor: scoreData.IPColocationFactor,
				BehaviourPenalty:   scoreData.BehaviourPenalty,
				Topics:             make([]peer.TopicScore, 0, len(scoreData.Topics)),
			}

			// Copy topic scores with full data
			for _, topicScore := range scoreData.Topics {
				scoreSnapshot.Topics = append(scoreSnapshot.Topics, peer.TopicScore{
					Topic:                    topicScore.Topic,
					TimeInMesh:               topicScore.TimeInMesh,
					FirstMessageDeliveries:   topicScore.FirstMessageDeliveries,
					MeshMessageDeliveries:    topicScore.MeshMessageDeliveries,
					InvalidMessageDeliveries: topicScore.InvalidMessageDeliveries,
				})
			}

			session.PeerScores = append(session.PeerScores, scoreSnapshot)

			h.logger.WithFields(logrus.Fields{
				"peer_id":   common.FormatShortPeerID(peerStats.PeerID),
				"score":     scoreData.Score,
				"topics":    len(scoreData.Topics),
				"timestamp": scoreData.Timestamp,
			}).Debug("Added peer score snapshot")

			return
		}
	}

	// No active session found, create a new one for this score event.
	h.logger.WithField("peer_id", common.FormatShortPeerID(peerStats.PeerID)).Debug("Creating new session for peer score event")

	now := scoreData.Timestamp
	session := peer.ConnectionSession{
		ConnectedAt:   &now,
		Disconnected:  false,
		PeerScores:    []peer.PeerScoreSnapshot{},
		GoodbyeEvents: []peer.GoodbyeEvent{},
		MeshEvents:    []peer.MeshEvent{},
	}

	// Add the score to the new session
	scoreSnapshot := peer.PeerScoreSnapshot{
		Score:              scoreData.Score,
		Timestamp:          scoreData.Timestamp,
		AppSpecificScore:   scoreData.AppSpecificScore,
		IPColocationFactor: scoreData.IPColocationFactor,
		BehaviourPenalty:   scoreData.BehaviourPenalty,
		Topics:             make([]peer.TopicScore, 0, len(scoreData.Topics)),
	}

	// Copy topic scores with full data
	for _, topicScore := range scoreData.Topics {
		scoreSnapshot.Topics = append(scoreSnapshot.Topics, peer.TopicScore{
			Topic:                    topicScore.Topic,
			TimeInMesh:               topicScore.TimeInMesh,
			FirstMessageDeliveries:   topicScore.FirstMessageDeliveries,
			MeshMessageDeliveries:    topicScore.MeshMessageDeliveries,
			InvalidMessageDeliveries: topicScore.InvalidMessageDeliveries,
		})
	}

	session.PeerScores = append(session.PeerScores, scoreSnapshot)

	peerStats.ConnectionSessions = append(peerStats.ConnectionSessions, session)

	h.logger.WithFields(logrus.Fields{
		"peer_id":   common.FormatShortPeerID(peerStats.PeerID),
		"score":     scoreData.Score,
		"topics":    len(scoreData.Topics),
		"timestamp": scoreData.Timestamp,
	}).Debug("Added peer score snapshot to new session")
}
