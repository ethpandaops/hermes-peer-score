package common

import (
	"github.com/sirupsen/logrus"
)

// ToolInterface defines the interface that event handlers need from the peer score tool.
type ToolInterface interface {
	GetPeer(peerID string) (interface{}, bool)
	CreatePeer(peerID string) interface{}
	UpdatePeer(peerID string, updateFn func(interface{}))
	UpdateOrCreatePeer(peerID string, updateFn func(interface{}))
	GetLogger() logrus.FieldLogger
	IncrementEventCount(peerID, eventType string)
	IncrementMessageCount(peerID string)
}
