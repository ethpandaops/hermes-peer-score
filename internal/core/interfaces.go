package core

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/config"
)

// Tool defines the interface for the main peer score tool.
type Tool interface {
	Start(ctx context.Context) error
	Stop() error
	GenerateReport() (*Report, error)
	GetLogger() logrus.FieldLogger
	GetConfig() Config
}

// Config is an alias for the config package interface.
type Config = config.Config

// EventManager defines the interface for managing events.
type EventManager interface {
	RegisterHandlers() error
	HandleEvent(ctx context.Context, event interface{}) error
	Start(ctx context.Context) error
	Stop() error
}

// PeerManager defines the interface for managing peer data.
type PeerManager interface {
	GetPeer(peerID string) (interface{}, bool)
	CreatePeer(peerID string) interface{}
	UpdatePeer(peerID string, updateFn func(interface{}))
	GetAllPeers() map[string]interface{}
	GetStats() interface{}
}

// ReportGenerator defines the interface for generating reports.
type ReportGenerator interface {
	Generate(config Config, startTime, endTime time.Time, peers map[string]interface{}) (*Report, error)
	SaveReport(report *Report) error
}

// HermesController defines the interface for controlling the Hermes node.
type HermesController interface {
	Start(ctx context.Context) error
	Stop() error
	RegisterEventCallback(callback func(ctx context.Context, event interface{}) error)
	GetNode() interface{}
}

// Report represents the main report structure.
type Report struct {
	Config               Config                    `json:"config"`
	ValidationMode       string                    `json:"validation_mode"`
	Timestamp            time.Time                 `json:"timestamp"`
	StartTime            time.Time                 `json:"start_time"`
	EndTime              time.Time                 `json:"end_time"`
	Duration             time.Duration             `json:"duration"`
	TotalConnections     int                       `json:"total_connections"`
	SuccessfulHandshakes int                       `json:"successful_handshakes"`
	FailedHandshakes     int                       `json:"failed_handshakes"`
	Peers                map[string]interface{}    `json:"peers"`
	PeerEventCounts      map[string]map[string]int `json:"peer_event_counts"`
}
