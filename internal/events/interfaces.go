package events

import (
	"context"

	"github.com/probe-lab/hermes/host"
	"github.com/ethpandaops/hermes-peer-score/internal/common"
)

// Handler defines the interface for handling specific event types
type Handler interface {
	HandleEvent(ctx context.Context, event *host.TraceEvent) error
	EventType() string
}

// PayloadParser defines the interface for parsing event payloads
type PayloadParser interface {
	Parse(payload interface{}) (interface{}, error)
	SupportedType() string
}

// Manager defines the interface for managing event handlers
type Manager interface {
	RegisterHandler(handler Handler) error
	HandleEvent(ctx context.Context, event *host.TraceEvent) error
	GetHandler(eventType string) (Handler, bool)
}

// ToolInterface is an alias for the common interface
type ToolInterface = common.ToolInterface

