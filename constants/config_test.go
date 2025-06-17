package constants

import (
	"testing"
	"time"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"DefaultTestDuration", DefaultTestDuration, 2 * time.Minute},
		{"DefaultPrysmHTTPPort", DefaultPrysmHTTPPort, 443},
		{"DefaultMaxPeers", DefaultMaxPeers, 30},
		{"ShortPeerIDLength", ShortPeerIDLength, 12},
		{"HermesVersionDelegated", HermesVersionDelegated, "v0.0.4-0.20250513093811-320c1c3ee6e2"},
		{"HermesVersionIndependent", HermesVersionIndependent, "v0.0.4-0.20250611021139-b3e6fc7d4d79"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	if ErrInvalidValidationMode == "" {
		t.Error("ErrInvalidValidationMode should not be empty")
	}
	if ErrPrysmHostRequired == "" {
		t.Error("ErrPrysmHostRequired should not be empty")
	}
}