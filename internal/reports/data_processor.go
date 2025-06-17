package reports

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	
	"github.com/ethpandaops/hermes-peer-score/constants"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
)

// DefaultDataProcessor implements the DataProcessor interface
type DefaultDataProcessor struct {
	logger logrus.FieldLogger
}

// NewDefaultDataProcessor creates a new data processor
func NewDefaultDataProcessor(logger logrus.FieldLogger) *DefaultDataProcessor {
	return &DefaultDataProcessor{
		logger: logger.WithField("component", "data_processor"),
	}
}

// ProcessPeerData processes peer data for JavaScript consumption
func (dp *DefaultDataProcessor) ProcessPeerData(peers map[string]interface{}) (interface{}, error) {
	processedPeers := make([]map[string]interface{}, 0, len(peers))
	
	for peerID, peerData := range peers {
		processed := dp.processSinglePeer(peerID, peerData)
		processedPeers = append(processedPeers, processed)
	}
	
	// Sort peers by peer ID for consistent output
	sort.Slice(processedPeers, func(i, j int) bool {
		id1, _ := processedPeers[i]["peer_id"].(string)
		id2, _ := processedPeers[j]["peer_id"].(string)
		return id1 < id2
	})
	
	return map[string]interface{}{
		"peers": processedPeers,
		"metadata": map[string]interface{}{
			"total_peers":      len(processedPeers),
			"processed_at":     time.Now(),
			"format_version":   "1.0",
		},
	}, nil
}

// CalculateSummaryStats calculates summary statistics from the report
func (dp *DefaultDataProcessor) CalculateSummaryStats(report *Report) (interface{}, error) {
	summary := map[string]interface{}{
		"TestDuration":         report.Duration.Seconds(),
		"StartTime":           report.StartTime,
		"EndTime":             report.EndTime,
		"TotalConnections":    report.TotalConnections,
		"SuccessfulHandshakes": report.SuccessfulHandshakes,
		"FailedHandshakes":    report.FailedHandshakes,
		"UniquePeers":         len(report.Peers),
	}
	
	// Calculate goodbye events summary
	goodbyeSummary := peer.CalculateGoodbyeEventsSummaryFromInterface(report.Peers)
	summary["goodbye_events_summary"] = goodbyeSummary
	
	// Calculate additional statistics
	clientDistribution := make(map[string]int)
	peerSummaries := make([]map[string]interface{}, 0, len(report.Peers))
	
	for peerID, peerData := range report.Peers {
		peerSummary := dp.createPeerSummary(peerID, peerData)
		peerSummaries = append(peerSummaries, peerSummary)
		
		// Count client types
		if clientType, ok := peerSummary["client_type"].(string); ok && clientType != "" {
			clientDistribution[clientType]++
		}
	}
	
	// Sort peer summaries by peer ID
	sort.Slice(peerSummaries, func(i, j int) bool {
		id1, _ := peerSummaries[i]["peer_id"].(string)
		id2, _ := peerSummaries[j]["peer_id"].(string)
		return id1 < id2
	})
	
	summary["client_distribution"] = clientDistribution
	summary["peer_summaries"] = peerSummaries
	
	return summary, nil
}

// FormatForTemplate formats the report data for template rendering
func (dp *DefaultDataProcessor) FormatForTemplate(report *Report) (interface{}, error) {
	summaryStats, err := dp.CalculateSummaryStats(report)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate summary stats: %w", err)
	}
	
	summary, ok := summaryStats.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid summary stats format")
	}
	
	templateData := map[string]interface{}{
		"GeneratedAt":      time.Now(),
		"Summary":          summary,
		"ValidationMode":   report.ValidationMode,
		"ValidationConfig": report.ValidationConfig,
		"DataFile":         "", // Will be set by generator
		"AIAnalysis":       "", // Will be set by generator if available
		"AIAnalysisHTML":   template.HTML(""), // Safe HTML version
	}
	
	return templateData, nil
}

// processSinglePeer processes a single peer's data
func (dp *DefaultDataProcessor) processSinglePeer(peerID string, peerData interface{}) map[string]interface{} {
	processed := map[string]interface{}{
		"peer_id":       peerID,
		"short_peer_id": dp.formatShortPeerID(peerID),
	}
	
	// Handle different types of peer data structures
	switch peerObj := peerData.(type) {
	case map[string]interface{}:
		dp.extractFromMap(peerObj, processed)
	case *peer.Stats:
		// Handle peer.Stats directly
		dp.extractFromPeerStats(peerObj, processed)
	default:
		dp.logger.WithFields(logrus.Fields{
			"peer_id": peerID,
			"type":    fmt.Sprintf("%T", peerData),
		}).Warn("Unknown peer data format")
		processed["client_type"] = constants.Unknown
		processed["session_count"] = 0
		processed["event_count"] = 0
	}
	
	return processed
}

// extractFromPeerStats extracts data from a peer.Stats struct
func (dp *DefaultDataProcessor) extractFromPeerStats(peerStats *peer.Stats, target map[string]interface{}) {
	// Copy basic fields
	target["client_type"] = peerStats.ClientType
	target["client_agent"] = peerStats.ClientAgent
	target["total_connections"] = peerStats.TotalConnections
	target["total_message_count"] = peerStats.TotalMessageCount
	target["successful_handshakes"] = peerStats.SuccessfulHandshakes
	target["failed_handshakes"] = peerStats.FailedHandshakes
	target["first_seen_at"] = peerStats.FirstSeenAt
	target["last_seen_at"] = peerStats.LastSeenAt
	
	// Process sessions
	sessionCount := len(peerStats.ConnectionSessions)
	target["session_count"] = sessionCount
	target["connection_sessions"] = peerStats.ConnectionSessions
	
	// Calculate session statistics
	eventCount := 0
	goodbyeCount := 0
	meshCount := 0
	hasScores := false
	var minScore, maxScore float64
	var lastSessionStatus string
	
	for _, session := range peerStats.ConnectionSessions {
		eventCount += session.MessageCount
		goodbyeCount += len(session.GoodbyeEvents)
		meshCount += len(session.MeshEvents)
		
		if len(session.PeerScores) > 0 {
			for _, score := range session.PeerScores {
				if !hasScores {
					minScore = score.Score
					maxScore = score.Score
					hasScores = true
				} else {
					if score.Score < minScore {
						minScore = score.Score
					}
					if score.Score > maxScore {
						maxScore = score.Score
					}
				}
			}
		}
		
		// Determine last session status
		if session.Disconnected {
			lastSessionStatus = "Disconnected"
		} else {
			lastSessionStatus = "Connected"
		}
	}
	
	target["event_count"] = eventCount
	target["goodbye_count"] = goodbyeCount
	target["mesh_count"] = meshCount
	target["has_scores"] = hasScores
	target["min_peer_score"] = minScore
	target["max_peer_score"] = maxScore
	target["last_session_status"] = lastSessionStatus
}

// extractFromMap extracts data from a map-based peer structure
func (dp *DefaultDataProcessor) extractFromMap(source, target map[string]interface{}) {
	// Copy basic fields
	if clientType, ok := source["client_type"].(string); ok {
		target["client_type"] = clientType
	} else {
		target["client_type"] = constants.Unknown
	}
	
	if clientAgent, ok := source["client_agent"].(string); ok {
		target["client_agent"] = clientAgent
	}
	
	// Process sessions
	sessionCount := 0
	if sessions, ok := source["connection_sessions"].([]interface{}); ok {
		sessionCount = len(sessions)
		target["session_count"] = sessionCount
		
		// Extract additional session information
		dp.processSessionData(sessions, target)
	}
	
	target["session_count"] = sessionCount
	
	// Set default values for missing fields
	if _, ok := target["event_count"]; !ok {
		target["event_count"] = 0
	}
	if _, ok := target["goodbye_count"]; !ok {
		target["goodbye_count"] = 0
	}
	if _, ok := target["mesh_count"]; !ok {
		target["mesh_count"] = 0
	}
	if _, ok := target["has_scores"]; !ok {
		target["has_scores"] = false
	}
	if _, ok := target["last_session_status"]; !ok {
		target["last_session_status"] = constants.Unknown
	}
	if _, ok := target["last_session_time"]; !ok {
		target["last_session_time"] = ""
	}
}

// processSessionData extracts information from session data
func (dp *DefaultDataProcessor) processSessionData(sessions []interface{}, target map[string]interface{}) {
	goodbyeCount := 0
	meshCount := 0
	hasScores := false
	minScore, maxScore := 0.0, 0.0
	lastSessionStatus := constants.Unknown
	lastSessionTime := ""
	
	for _, sessionData := range sessions {
		if session, ok := sessionData.(map[string]interface{}); ok {
			// Count goodbye events
			if goodbyes, ok := session["goodbye_events"].([]interface{}); ok {
				goodbyeCount += len(goodbyes)
			}
			
			// Count mesh events
			if meshEvents, ok := session["mesh_events"].([]interface{}); ok {
				meshCount += len(meshEvents)
			}
			
			// Process peer scores
			if scores, ok := session["peer_scores"].([]interface{}); ok && len(scores) > 0 {
				hasScores = true
				for i, scoreData := range scores {
					if score, ok := scoreData.(map[string]interface{}); ok {
						if scoreValue, ok := score["score"].(float64); ok {
							if i == 0 {
								minScore = scoreValue
								maxScore = scoreValue
							} else {
								if scoreValue < minScore {
									minScore = scoreValue
								}
								if scoreValue > maxScore {
									maxScore = scoreValue
								}
							}
						}
					}
				}
			}
			
			// Determine session status
			if disconnected, ok := session["disconnected"].(bool); ok {
				if disconnected {
					lastSessionStatus = "disconnected"
				} else {
					lastSessionStatus = "connected"
				}
			}
			
			// Get last session time
			if connectedAt, ok := session["connected_at"].(string); ok {
				lastSessionTime = connectedAt
			}
		}
	}
	
	target["goodbye_count"] = goodbyeCount
	target["mesh_count"] = meshCount
	target["has_scores"] = hasScores
	target["min_peer_score"] = minScore
	target["max_peer_score"] = maxScore
	target["last_session_status"] = lastSessionStatus
	target["last_session_time"] = lastSessionTime
}

// createPeerSummary creates a summary for a single peer
func (dp *DefaultDataProcessor) createPeerSummary(peerID string, peerData interface{}) map[string]interface{} {
	summary := map[string]interface{}{
		"peer_id":            peerID,
		"short_peer_id":      dp.formatShortPeerID(peerID),
		"client_type":        constants.Unknown,
		"client_agent":       "",
		"session_count":      0,
		"event_count":        0,
		"goodbye_count":      0,
		"mesh_count":         0,
		"min_peer_score":     0.0,
		"max_peer_score":     0.0,
		"has_scores":         false,
		"last_session_status": constants.Unknown,
		"last_session_time":  "",
	}
	
	if peer, ok := peerData.(map[string]interface{}); ok {
		dp.extractFromMap(peer, summary)
	}
	
	return summary
}

// formatShortPeerID returns a shortened version of the peer ID
func (dp *DefaultDataProcessor) formatShortPeerID(peerID string) string {
	if len(peerID) <= 12 {
		return peerID
	}
	return peerID[:12]
}

// CleanAIHTML cleans and sanitizes AI-generated HTML content
func (dp *DefaultDataProcessor) CleanAIHTML(content string) template.HTML {
	// Basic HTML cleaning - remove potentially dangerous elements
	cleaned := content
	
	// Remove script tags
	cleaned = removeHTMLTags(cleaned, "script")
	cleaned = removeHTMLTags(cleaned, "style")
	cleaned = removeHTMLTags(cleaned, "link")
	cleaned = removeHTMLTags(cleaned, "meta")
	
	// Convert markdown-style formatting to HTML
	cleaned = dp.convertMarkdownToHTML(cleaned)
	
	return template.HTML(cleaned) //nolint:gosec // Content is cleaned above
}

// convertMarkdownToHTML converts basic markdown to HTML
func (dp *DefaultDataProcessor) convertMarkdownToHTML(content string) string {
	// Convert **bold** to <strong>
	content = strings.ReplaceAll(content, "**", "<strong>")
	content = strings.ReplaceAll(content, "</strong>", "</strong>")
	
	// Convert *italic* to <em>
	content = strings.ReplaceAll(content, "*", "<em>")
	content = strings.ReplaceAll(content, "</em>", "</em>")
	
	// Convert line breaks to <br>
	content = strings.ReplaceAll(content, "\n", "<br>")
	
	return content
}

// removeHTMLTags removes specified HTML tags from content
func removeHTMLTags(content, tagName string) string {
	// Simple tag removal - for production use a proper HTML parser
	openTag := "<" + tagName
	closeTag := "</" + tagName + ">"
	
	for {
		startIndex := strings.Index(content, openTag)
		if startIndex == -1 {
			break
		}
		
		endIndex := strings.Index(content[startIndex:], closeTag)
		if endIndex == -1 {
			// No closing tag found, remove to end
			content = content[:startIndex]
			break
		}
		
		endIndex += startIndex + len(closeTag)
		content = content[:startIndex] + content[endIndex:]
	}
	
	return content
}