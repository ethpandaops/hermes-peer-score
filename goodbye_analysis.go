package main

import (
	"sort"
	"strings"
)

// GoodbyeReasonStats tracks statistics for a specific goodbye reason
type GoodbyeReasonStats struct {
	Reason   string   `json:"reason"`   // Original reason string
	Count    int      `json:"count"`    // Number of occurrences
	Codes    []uint64 `json:"codes"`    // All unique codes seen with this reason
	Examples []string `json:"examples"` // First few examples of this reason
}

// GoodbyeEventsSummary contains aggregated goodbye event statistics
type GoodbyeEventsSummary struct {
	TotalEvents   int                   `json:"total_events"`   // Total number of goodbye events
	ReasonStats   []*GoodbyeReasonStats `json:"reason_stats"`   // Sorted by count (most common first)
	UniqueReasons int                   `json:"unique_reasons"` // Number of unique reasons
	TopReasons    []string              `json:"top_reasons"`    // Top 5 most common reasons
	CodeFrequency map[uint64]int        `json:"code_frequency"` // Code occurrence count
}

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

// calculateGoodbyeEventsSummary aggregates goodbye event statistics from all peers
func calculateGoodbyeEventsSummary(peers map[string]*PeerStats) GoodbyeEventsSummary {
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
	var statsList []*GoodbyeReasonStats
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
