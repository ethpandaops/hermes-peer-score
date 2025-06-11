package events

import (
	"context"
	"fmt"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/common"
	"github.com/ethpandaops/hermes-peer-score/internal/events/handlers"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	handlers map[string]Handler
	tool     common.ToolInterface
	logger   logrus.FieldLogger
}

// NewManager creates a new event manager with the given tool interface
func NewManager(tool common.ToolInterface, logger logrus.FieldLogger) *DefaultManager {
	return &DefaultManager{
		handlers: make(map[string]Handler),
		tool:     tool,
		logger:   logger,
	}
}

// RegisterHandler registers a handler for a specific event type
func (m *DefaultManager) RegisterHandler(handler Handler) error {
	eventType := handler.EventType()
	if eventType == "" {
		return fmt.Errorf("handler must specify a non-empty event type")
	}

	if _, exists := m.handlers[eventType]; exists {
		return fmt.Errorf("handler for event type %s already registered", eventType)
	}

	m.handlers[eventType] = handler
	m.logger.WithField("event_type", eventType).Debug("Registered event handler")
	return nil
}

// GetHandler returns the handler for the given event type
func (m *DefaultManager) GetHandler(eventType string) (Handler, bool) {
	handler, exists := m.handlers[eventType]
	return handler, exists
}

// HandleEvent routes the event to the appropriate handler
func (m *DefaultManager) HandleEvent(ctx context.Context, event *host.TraceEvent) error {
	// Add validation mode context to all event logging
	eventLogger := m.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
	})

	// Count the event by peer ID and event type
	peerID := common.GetPeerID(event)
	if peerID != "" && peerID != "unknown" {
		m.tool.IncrementEventCount(peerID, event.Type)
	}

	// Find and execute the appropriate handler
	handler, exists := m.handlers[event.Type]
	if !exists {
		eventLogger.Debug("Unhandled event type")
		return nil
	}

	// Execute the handler
	if err := handler.HandleEvent(ctx, event); err != nil {
		return fmt.Errorf("handler for event type %s failed: %w", event.Type, err)
	}

	return nil
}

// RegisterDefaultHandlers registers all the default event handlers
func (m *DefaultManager) RegisterDefaultHandlers() error {
	// Register all event handlers
	eventHandlers := []Handler{
		handlers.NewConnectionHandler(m.tool, m.logger),
		handlers.NewDisconnectionHandler(m.tool, m.logger),
		handlers.NewStatusHandler(m.tool, m.logger),
		handlers.NewPeerScoreHandler(m.tool, m.logger),
		handlers.NewGoodbyeHandler(m.tool, m.logger),
		handlers.NewGraftHandler(m.tool, m.logger),
		handlers.NewPruneHandler(m.tool, m.logger),
	}

	for _, handler := range eventHandlers {
		if err := m.RegisterHandler(handler); err != nil {
			return fmt.Errorf("failed to register handler %s: %w", handler.EventType(), err)
		}
		m.logger.WithField("event_type", handler.EventType()).Debug("Registered event handler")
	}
	return nil
}