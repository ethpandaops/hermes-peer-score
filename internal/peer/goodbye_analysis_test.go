package peer

import (
	"testing"
	"time"
)

func TestAnalyzeGoodbyeReasons(t *testing.T) {
	// Create test goodbye events
	events := []GoodbyeEvent{
		{
			Timestamp: time.Now(),
			Code:      1,
			Reason:    "Client shutdown",
		},
		{
			Timestamp: time.Now(),
			Code:      1,
			Reason:    "Client shutdown",
		},
		{
			Timestamp: time.Now(),
			Code:      2,
			Reason:    "Network error",
		},
		{
			Timestamp: time.Now(),
			Code:      3,
			Reason:    "", // Empty reason
		},
	}

	// Analyze the events
	stats := AnalyzeGoodbyeReasons(events)

	// Verify we have the expected number of reason groups
	expectedGroups := 3 // "client shutdown", "network error", "unknown"
	if len(stats) != expectedGroups {
		t.Errorf("Expected %d reason groups, got %d", expectedGroups, len(stats))
	}

	// Check "client shutdown" group
	if clientShutdown, exists := stats["client shutdown"]; exists {
		if clientShutdown.Count != 2 {
			t.Errorf("Expected 2 occurrences of 'client shutdown', got %d", clientShutdown.Count)
		}
		if len(clientShutdown.Codes) != 1 || clientShutdown.Codes[0] != 1 {
			t.Errorf("Expected code [1] for 'client shutdown', got %v", clientShutdown.Codes)
		}
	} else {
		t.Error("Expected 'client shutdown' group not found")
	}

	// Check "network error" group
	if networkError, exists := stats["network error"]; exists {
		if networkError.Count != 1 {
			t.Errorf("Expected 1 occurrence of 'network error', got %d", networkError.Count)
		}
		if len(networkError.Codes) != 1 || networkError.Codes[0] != 2 {
			t.Errorf("Expected code [2] for 'network error', got %v", networkError.Codes)
		}
	} else {
		t.Error("Expected 'network error' group not found")
	}

	// Check "unknown" group (for empty reason)
	if unknown, exists := stats["unknown"]; exists {
		if unknown.Count != 1 {
			t.Errorf("Expected 1 occurrence of 'unknown', got %d", unknown.Count)
		}
		if len(unknown.Codes) != 1 || unknown.Codes[0] != 3 {
			t.Errorf("Expected code [3] for 'unknown', got %v", unknown.Codes)
		}
	} else {
		t.Error("Expected 'unknown' group not found")
	}
}

func TestCalculateGoodbyeEventsSummary(t *testing.T) {
	// Create test peer data
	peers := map[string]*Stats{
		"peer1": {
			PeerID:      "peer1",
			ClientType:  "test-client",
			ClientAgent: "test-agent/1.0",
			ConnectionSessions: []ConnectionSession{
				{
					GoodbyeEvents: []GoodbyeEvent{
						{
							Timestamp: time.Now(),
							Code:      1,
							Reason:    "Client shutdown",
						},
						{
							Timestamp: time.Now(),
							Code:      2,
							Reason:    "Network error",
						},
					},
				},
			},
		},
		"peer2": {
			PeerID:      "peer2",
			ClientType:  "test-client",
			ClientAgent: "test-agent/1.0",
			ConnectionSessions: []ConnectionSession{
				{
					GoodbyeEvents: []GoodbyeEvent{
						{
							Timestamp: time.Now(),
							Code:      1,
							Reason:    "Client shutdown",
						},
					},
				},
			},
		},
	}

	// Calculate summary
	summary := CalculateGoodbyeEventsSummary(peers)

	// Verify total events
	expectedTotal := 3
	if summary.TotalEvents != expectedTotal {
		t.Errorf("Expected %d total events, got %d", expectedTotal, summary.TotalEvents)
	}

	// Verify unique reasons
	expectedUnique := 2 // "Client shutdown" and "Network error"
	if summary.UniqueReasons != expectedUnique {
		t.Errorf("Expected %d unique reasons, got %d", expectedUnique, summary.UniqueReasons)
	}

	// Verify code frequency
	if summary.CodeFrequency[1] != 2 {
		t.Errorf("Expected code 1 to occur 2 times, got %d", summary.CodeFrequency[1])
	}
	if summary.CodeFrequency[2] != 1 {
		t.Errorf("Expected code 2 to occur 1 time, got %d", summary.CodeFrequency[2])
	}

	// Verify top reasons
	if len(summary.TopReasons) == 0 {
		t.Error("Expected at least one top reason")
	}
	
	// The most common reason should be "Client shutdown" (2 occurrences)
	if len(summary.TopReasons) > 0 && summary.TopReasons[0] != "Client shutdown" {
		t.Errorf("Expected first top reason to be 'Client shutdown', got '%s'", summary.TopReasons[0])
	}
}

func TestContainsCode(t *testing.T) {
	codes := []uint64{1, 2, 3}

	if !containsCode(codes, 2) {
		t.Error("Expected containsCode to return true for existing code")
	}

	if containsCode(codes, 4) {
		t.Error("Expected containsCode to return false for non-existing code")
	}

	if containsCode([]uint64{}, 1) {
		t.Error("Expected containsCode to return false for empty slice")
	}
}

func TestParseTimestampString(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool // whether parsing should succeed
	}{
		{"2023-12-01T15:04:05Z", true},
		{"2023-12-01T15:04:05.123456789Z", true},
		{"2023-12-01 15:04:05", true},
		{"2023-12-01T15:04:05", true},
		{"Dec  1 15:04:05", true},
		{"invalid-timestamp", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := parseTimestampString(tc.input)
		isZero := result.IsZero()
		
		if tc.expected && isZero {
			t.Errorf("Expected successful parsing for '%s', but got zero time", tc.input)
		}
		if !tc.expected && !isZero {
			t.Errorf("Expected failed parsing for '%s', but got valid time: %v", tc.input, result)
		}
	}
}

func TestExtractGoodbyeEvent(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected GoodbyeEvent
	}{
		{
			name: "Complete event with string timestamp",
			input: map[string]interface{}{
				"timestamp": "2023-12-01T15:04:05Z",
				"code":      float64(1),
				"reason":    "Client shutdown",
			},
			expected: GoodbyeEvent{
				Code:   1,
				Reason: "Client shutdown",
			},
		},
		{
			name: "Event with unix timestamp",
			input: map[string]interface{}{
				"timestamp": int64(1701442800), // Unix timestamp
				"code":      uint64(2),
				"reason":    "Network error",
			},
			expected: GoodbyeEvent{
				Code:   2,
				Reason: "Network error",
			},
		},
		{
			name: "Event with nanosecond timestamp",
			input: map[string]interface{}{
				"timestamp": float64(1701442800123456789), // Nanoseconds
				"code":      int(3),
				"reason":    "Protocol error",
			},
			expected: GoodbyeEvent{
				Code:   3,
				Reason: "Protocol error",
			},
		},
		{
			name: "Event with missing reason",
			input: map[string]interface{}{
				"timestamp": "2023-12-01T15:04:05Z",
				"code":      float64(4),
			},
			expected: GoodbyeEvent{
				Code:   4,
				Reason: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractGoodbyeEvent(tc.input)
			
			if result.Code != tc.expected.Code {
				t.Errorf("Expected code %d, got %d", tc.expected.Code, result.Code)
			}
			if result.Reason != tc.expected.Reason {
				t.Errorf("Expected reason '%s', got '%s'", tc.expected.Reason, result.Reason)
			}
			// Only check that timestamp was parsed (not zero) for valid inputs
			if tc.input["timestamp"] != nil && result.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}
		})
	}
}