package reports

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Generator defines the interface for report generation
type Generator interface {
	GenerateJSON(report *Report) (string, error)
	GenerateHTML(report *Report) (string, error)
	GenerateHTMLWithAI(report *Report, apiKey string) (string, error)
}

// TemplateManager defines the interface for template management
type TemplateManager interface {
	LoadTemplates() error
	RenderReport(data interface{}) (string, error)
	GetTemplate(name string) (string, error)
}

// FileManager defines the interface for file operations
type FileManager interface {
	SaveJSON(filename string, data interface{}) error
	SaveHTML(filename string, content string) error
	FileExists(filename string) bool
	GenerateFilename(base string, timestamp time.Time) string
}

// Report represents the comprehensive analysis results from a peer scoring test
type Report struct {
	Config               interface{}               `json:"config"`
	ValidationMode       string                    `json:"validation_mode"`
	ValidationConfig     interface{}               `json:"validation_config"`
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

// AIAnalyzer defines the interface for AI-powered analysis
type AIAnalyzer interface {
	AnalyzeReport(report *Report, apiKey string) (string, error)
	GenerateInsights(data interface{}) (string, error)
}

// DataProcessor defines the interface for processing report data
type DataProcessor interface {
	ProcessPeerData(peers map[string]interface{}) (interface{}, error)
	CalculateSummaryStats(report *Report) (interface{}, error)
	FormatForTemplate(report *Report) (interface{}, error)
}

// Logger defines the interface for report generation logging
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	WithFields(fields logrus.Fields) logrus.FieldLogger
}