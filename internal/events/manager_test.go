package events

import (
	"context"
	"testing"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
)

// MockHandler for testing
type MockHandler struct {
	eventType    string
	handleCalled bool
	handleError  error
}

func (m *MockHandler) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	m.handleCalled = true
	return m.handleError
}

func (m *MockHandler) EventType() string {
	return m.eventType
}

// MockToolInterface for testing
type MockToolInterface struct {
	peers       map[string]interface{}
	eventCounts map[string]map[string]int
}

func NewMockToolInterface() *MockToolInterface {
	return &MockToolInterface{
		peers:       make(map[string]interface{}),
		eventCounts: make(map[string]map[string]int),
	}
}

func (m *MockToolInterface) GetPeer(peerID string) (interface{}, bool) {
	peer, exists := m.peers[peerID]
	return peer, exists
}

func (m *MockToolInterface) CreatePeer(peerID string) interface{} {
	peer := map[string]interface{}{"peer_id": peerID}
	m.peers[peerID] = peer
	return peer
}

func (m *MockToolInterface) UpdatePeer(peerID string, updateFn func(interface{})) {
	if peer, exists := m.peers[peerID]; exists {
		updateFn(peer)
	}
}

func (m *MockToolInterface) GetLogger() logrus.FieldLogger {
	return logrus.New()
}

func (m *MockToolInterface) IncrementEventCount(peerID, eventType string) {
	if _, exists := m.eventCounts[peerID]; !exists {
		m.eventCounts[peerID] = make(map[string]int)
	}
	m.eventCounts[peerID][eventType]++
}

func (m *MockToolInterface) IncrementMessageCount(peerID string) {
	// Mock implementation - in a real test this could track message counts
}

func TestEventManager(t *testing.T) {
	tool := NewMockToolInterface()
	logger := logrus.New()
	manager := NewManager(tool, logger)

	// Test handler registration
	mockHandler := &MockHandler{eventType: "TEST_EVENT"}
	err := manager.RegisterHandler(mockHandler)
	if err != nil {
		t.Errorf("Expected no error registering handler, got %v", err)
	}

	// Test duplicate handler registration
	err = manager.RegisterHandler(mockHandler)
	if err == nil {
		t.Error("Expected error when registering duplicate handler")
	}

	// Test getting handler
	handler, exists := manager.GetHandler("TEST_EVENT")
	if !exists {
		t.Error("Expected to find registered handler")
	}
	if handler != mockHandler {
		t.Error("Expected to get the same handler instance")
	}

	// Test event handling
	event := &host.TraceEvent{
		Type:    "TEST_EVENT",
		Payload: map[string]interface{}{"PeerID": "test-peer"},
	}

	ctx := context.Background()
	err = manager.HandleEvent(ctx, event)
	if err != nil {
		t.Errorf("Expected no error handling event, got %v", err)
	}

	if !mockHandler.handleCalled {
		t.Error("Expected handler to be called")
	}

	// Test unhandled event type
	unhandledEvent := &host.TraceEvent{
		Type:    "UNHANDLED_EVENT",
		Payload: map[string]interface{}{},
	}

	err = manager.HandleEvent(ctx, unhandledEvent)
	if err != nil {
		t.Errorf("Expected no error for unhandled event, got %v", err)
	}
}

func TestGetPeerID(t *testing.T) {
	tests := []struct {
		name     string
		event    *host.TraceEvent
		expected string
	}{
		{
			name:     "nil event",
			event:    nil,
			expected: unknown,
		},
		{
			name: "event with string PeerID in payload map",
			event: &host.TraceEvent{
				Payload: map[string]interface{}{
					"PeerID": "test-peer-123",
				},
			},
			expected: "test-peer-123",
		},
		{
			name: "event with no peer ID",
			event: &host.TraceEvent{
				Payload: map[string]interface{}{
					"other": "value",
				},
			},
			expected: unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPeerID(tt.event)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}