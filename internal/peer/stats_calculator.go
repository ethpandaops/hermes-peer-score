package peer

import (
	"time"

	"github.com/ethpandaops/hermes-peer-score/constants"
)

// DefaultStatsCalculator implements the StatsCalculator interface.
type DefaultStatsCalculator struct{}

// NewStatsCalculator creates a new statistics calculator.
func NewStatsCalculator() *DefaultStatsCalculator {
	return &DefaultStatsCalculator{}
}

// CalculateConnectionStats calculates aggregate connection statistics.
func (sc *DefaultStatsCalculator) CalculateConnectionStats(peers map[string]*Stats) ConnectionStats {
	stats := ConnectionStats{}

	for _, peer := range peers {
		// Count successful/failed handshakes per session
		hasActiveSession := false

		for _, session := range peer.ConnectionSessions {
			// Count this session toward total connections
			if session.ConnectedAt != nil {
				stats.TotalConnections++

				// Determine if handshake was successful or failed
				if session.IdentifiedAt != nil {
					stats.SuccessfulHandshakes++
				} else {
					// Connected but never identified = failed handshake
					stats.FailedHandshakes++
				}
			}

			// Check if peer has an active (non-disconnected) session
			if !session.Disconnected {
				hasActiveSession = true
			}
		}

		if hasActiveSession {
			stats.ConnectedPeers++
		}
	}

	return stats
}

// CalculateClientDistribution calculates the distribution of client types.
func (sc *DefaultStatsCalculator) CalculateClientDistribution(peers map[string]*Stats) map[string]int {
	distribution := make(map[string]int)

	for _, peer := range peers {
		if peer.ClientType != "" {
			distribution[peer.ClientType]++
		} else {
			distribution[constants.Unknown]++
		}
	}

	return distribution
}

// CalculateDurationStats calculates connection duration statistics.
func (sc *DefaultStatsCalculator) CalculateDurationStats(peers map[string]*Stats) DurationStats {
	var durations []time.Duration

	var totalDuration time.Duration

	for _, peer := range peers {
		for _, session := range peer.ConnectionSessions {
			if session.Duration != nil && *session.Duration > 0 {
				durations = append(durations, *session.Duration)
				totalDuration += *session.Duration
			}
		}
	}

	stats := DurationStats{}

	if len(durations) > 0 {
		stats.AverageDuration = totalDuration / time.Duration(len(durations))
		stats.MinDuration = durations[0]
		stats.MaxDuration = durations[0]

		for _, duration := range durations {
			if duration < stats.MinDuration {
				stats.MinDuration = duration
			}

			if duration > stats.MaxDuration {
				stats.MaxDuration = duration
			}
		}
	}

	return stats
}

// CalculateMessageStats calculates message-related statistics.
func (sc *DefaultStatsCalculator) CalculateMessageStats(peers map[string]*Stats) MessageStats {
	stats := MessageStats{}

	var messageCounts []int

	for _, peer := range peers {
		totalMessages := 0
		for _, session := range peer.ConnectionSessions {
			totalMessages += session.MessageCount

			if session.MessageCount > 0 {
				messageCounts = append(messageCounts, session.MessageCount)
			}
		}

		stats.TotalMessages += totalMessages
		if totalMessages > 0 {
			stats.PeersWithMessages++
		}
	}

	if len(messageCounts) > 0 {
		// Calculate average messages per active session
		totalSessionMessages := 0
		for _, count := range messageCounts {
			totalSessionMessages += count
		}

		stats.AverageMessagesPerSession = float64(totalSessionMessages) / float64(len(messageCounts))

		// Find min/max
		stats.MinMessagesPerSession = messageCounts[0]
		stats.MaxMessagesPerSession = messageCounts[0]

		for _, count := range messageCounts {
			if count < stats.MinMessagesPerSession {
				stats.MinMessagesPerSession = count
			}

			if count > stats.MaxMessagesPerSession {
				stats.MaxMessagesPerSession = count
			}
		}
	}

	return stats
}

// CalculateIdentificationStats calculates peer identification statistics.
func (sc *DefaultStatsCalculator) CalculateIdentificationStats(peers map[string]*Stats) IdentificationStats {
	stats := IdentificationStats{}

	var identificationTimes []time.Duration

	for _, peer := range peers {
		hasIdentification := false

		for _, session := range peer.ConnectionSessions {
			if session.ConnectedAt != nil && session.IdentifiedAt != nil {
				identificationTime := session.IdentifiedAt.Sub(*session.ConnectedAt)
				identificationTimes = append(identificationTimes, identificationTime)
				hasIdentification = true
			}
		}

		if hasIdentification {
			stats.IdentifiedPeers++
		}
	}

	stats.TotalPeers = len(peers)

	if len(identificationTimes) > 0 {
		// Calculate average identification time
		var totalTime time.Duration
		for _, t := range identificationTimes {
			totalTime += t
		}

		stats.AverageIdentificationTime = totalTime / time.Duration(len(identificationTimes))

		// Find min/max
		stats.MinIdentificationTime = identificationTimes[0]
		stats.MaxIdentificationTime = identificationTimes[0]

		for _, t := range identificationTimes {
			if t < stats.MinIdentificationTime {
				stats.MinIdentificationTime = t
			}

			if t > stats.MaxIdentificationTime {
				stats.MaxIdentificationTime = t
			}
		}
	}

	return stats
}

// MessageStats holds message-related statistics.
type MessageStats struct {
	TotalMessages             int     `json:"total_messages"`
	PeersWithMessages         int     `json:"peers_with_messages"`
	AverageMessagesPerSession float64 `json:"average_messages_per_session"`
	MinMessagesPerSession     int     `json:"min_messages_per_session"`
	MaxMessagesPerSession     int     `json:"max_messages_per_session"`
}

// IdentificationStats holds peer identification statistics.
type IdentificationStats struct {
	TotalPeers                int           `json:"total_peers"`
	IdentifiedPeers           int           `json:"identified_peers"`
	AverageIdentificationTime time.Duration `json:"average_identification_time"`
	MinIdentificationTime     time.Duration `json:"min_identification_time"`
	MaxIdentificationTime     time.Duration `json:"max_identification_time"`
}
