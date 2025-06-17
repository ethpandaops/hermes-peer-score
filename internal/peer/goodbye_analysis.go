package peer

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// AnalyzeGoodbyeReasons groups and analyzes goodbye reasons from events
func AnalyzeGoodbyeReasons(events []GoodbyeEvent) map[string]*GoodbyeReasonStats {
	stats := make(map[string]*GoodbyeReasonStats)

	for _, event := range events {
		// Use lowercase for grouping but preserve original
		key := strings.ToLower(strings.TrimSpace(event.Reason))
		if key == "" {
			key = "unknown"
		}

		if stat, exists := stats[key]; exists {
			stat.Count++
			// Track unique codes
			if !containsCode(stat.Codes, event.Code) {
				stat.Codes = append(stat.Codes, event.Code)
			}
			// Keep up to 3 examples
			if len(stat.Examples) < 3 && event.Reason != "" {
				// Check if this exact example already exists
				alreadyExists := false
				for _, ex := range stat.Examples {
					if ex == event.Reason {
						alreadyExists = true
						break
					}
				}
				if !alreadyExists {
					stat.Examples = append(stat.Examples, event.Reason)
				}
			}
		} else {
			examples := []string{}
			if event.Reason != "" {
				examples = []string{event.Reason}
			}

			stats[key] = &GoodbyeReasonStats{
				Reason:   event.Reason, // Preserve original casing
				Count:    1,
				Codes:    []uint64{event.Code},
				Examples: examples,
			}
		}
	}

	for _, stat := range stats {
		fmt.Printf("stat: %+v\n", stat)
	}

	return stats
}

// containsCode checks if a code exists in the slice
func containsCode(codes []uint64, code uint64) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}

// CalculateGoodbyeEventsSummary aggregates goodbye event statistics from all peers
func CalculateGoodbyeEventsSummary(peers map[string]*Stats) GoodbyeEventsSummary {
	var allEvents []GoodbyeEvent
	codeFreq := make(map[uint64]int)

	// Collect all goodbye events from all peers
	for _, peer := range peers {
		for _, session := range peer.ConnectionSessions {
			for _, goodbye := range session.GoodbyeEvents {
				allEvents = append(allEvents, goodbye)
				codeFreq[goodbye.Code]++
			}
		}
	}

	// Analyze reasons
	reasonStats := AnalyzeGoodbyeReasons(allEvents)

	// Convert map to sorted slice
	statsList := make([]*GoodbyeReasonStats, 0, len(reasonStats))
	for _, stat := range reasonStats {
		statsList = append(statsList, stat)
	}
	// Sort by count (descending)
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].Count > statsList[j].Count
	})

	// Get top 5 reasons
	var topReasons []string
	for i := 0; i < 5 && i < len(statsList); i++ {
		reason := statsList[i].Reason
		if reason == "" {
			reason = "no reason provided"
		}
		topReasons = append(topReasons, reason)
	}

	return GoodbyeEventsSummary{
		TotalEvents:   len(allEvents),
		ReasonStats:   statsList,
		UniqueReasons: len(reasonStats),
		TopReasons:    topReasons,
		CodeFrequency: codeFreq,
	}
}

// CalculateGoodbyeEventsSummaryFromInterface calculates goodbye events summary from generic peer data
func CalculateGoodbyeEventsSummaryFromInterface(peers map[string]interface{}) GoodbyeEventsSummary {
	var allEvents []GoodbyeEvent
	codeFreq := make(map[uint64]int)

	// Collect all goodbye events from all peers
	for _, peerData := range peers {
		// Handle different types of peer data
		switch peer := peerData.(type) {
		case *Stats:
			for _, session := range peer.ConnectionSessions {
				for _, goodbye := range session.GoodbyeEvents {
					allEvents = append(allEvents, goodbye)
					codeFreq[goodbye.Code]++
				}
			}
		case map[string]interface{}:
			// Handle map-based peer data
			if sessions, ok := peer["connection_sessions"].([]interface{}); ok {
				for _, sessionData := range sessions {
					if session, ok := sessionData.(map[string]interface{}); ok {
						if goodbyes, ok := session["goodbye_events"].([]interface{}); ok {
							for _, goodbyeData := range goodbyes {
								if goodbyeMap, ok := goodbyeData.(map[string]interface{}); ok {
									goodbye := extractGoodbyeEvent(goodbyeMap)
									if goodbye != nil {
										allEvents = append(allEvents, *goodbye)
										codeFreq[goodbye.Code]++
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Analyze reasons
	reasonStats := AnalyzeGoodbyeReasons(allEvents)

	// Convert map to sorted slice
	statsList := make([]*GoodbyeReasonStats, 0, len(reasonStats))
	for _, stat := range reasonStats {
		statsList = append(statsList, stat)
	}
	// Sort by count (descending)
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].Count > statsList[j].Count
	})

	// Get top 5 reasons
	var topReasons []string
	for i := 0; i < 5 && i < len(statsList); i++ {
		reason := statsList[i].Reason
		if reason == "" {
			reason = "no reason provided"
		}
		topReasons = append(topReasons, reason)
	}

	return GoodbyeEventsSummary{
		TotalEvents:   len(allEvents),
		ReasonStats:   statsList,
		UniqueReasons: len(reasonStats),
		TopReasons:    topReasons,
		CodeFrequency: codeFreq,
	}
}

// extractGoodbyeEvent extracts a GoodbyeEvent from a map
func extractGoodbyeEvent(data map[string]interface{}) *GoodbyeEvent {
	event := &GoodbyeEvent{}

	// Extract timestamp - handle various timestamp formats used in production
	if timestampData, ok := data["timestamp"]; ok {
		switch ts := timestampData.(type) {
		case string:
			// Try parsing common timestamp formats
			event.Timestamp = parseTimestampString(ts)
		case time.Time:
			event.Timestamp = ts
		case int64:
			// Unix timestamp (seconds)
			event.Timestamp = time.Unix(ts, 0)
		case float64:
			// Unix timestamp as float (could be nanoseconds or seconds)
			if ts > 1e12 { // Likely nanoseconds if > 1e12
				event.Timestamp = time.Unix(0, int64(ts))
			} else { // Likely seconds
				event.Timestamp = time.Unix(int64(ts), 0)
			}
		default:
			event.Timestamp = time.Time{}
		}
	}

	// Extract code - handle different numeric types
	if code, ok := data["code"].(float64); ok {
		event.Code = uint64(code)
	} else if code, ok := data["code"].(uint64); ok {
		event.Code = code
	} else if code, ok := data["code"].(int64); ok {
		event.Code = uint64(code)
	} else if code, ok := data["code"].(int); ok {
		event.Code = uint64(code)
	}

	// Extract reason
	if reason, ok := data["reason"].(string); ok {
		event.Reason = reason
	}

	return event
}

// parseTimestampString attempts to parse timestamp strings in common formats
func parseTimestampString(timestampStr string) time.Time {
	// Common timestamp formats to try
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,       // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05Z", // ISO 8601 UTC
		"2006-01-02 15:04:05",  // Simple format
		"2006-01-02T15:04:05",  // ISO without timezone
		time.DateTime,          // "2006-01-02 15:04:05"
		time.Stamp,             // "Jan _2 15:04:05"
		time.StampMilli,        // "Jan _2 15:04:05.000"
		time.StampMicro,        // "Jan _2 15:04:05.000000"
		time.StampNano,         // "Jan _2 15:04:05.000000000"
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, timestampStr); err == nil {
			return parsed
		}
	}

	// If all parsing attempts fail, return zero time
	return time.Time{}
}
