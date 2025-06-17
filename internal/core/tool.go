package core

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/internal/config"
	"github.com/ethpandaops/hermes-peer-score/internal/events"
	"github.com/ethpandaops/hermes-peer-score/internal/peer"
	"github.com/ethpandaops/hermes-peer-score/internal/reports"
)

// DefaultTool implements the Tool interface.
type DefaultTool struct {
	config    config.Config
	logger    logrus.FieldLogger
	startTime time.Time

	// Core components
	peerRepo   peer.Repository
	sessionMgr peer.SessionManager
	eventMgr   *events.DefaultManager
	reportGen  *reports.DefaultGenerator
	hermesCtrl HermesController

	// Event counting
	peerEventCounts map[string]map[string]int
}

// NewTool creates a new peer score tool instance.
func NewTool(ctx context.Context, cfg config.Config, logger logrus.FieldLogger) (*DefaultTool, error) {
	tool := &DefaultTool{
		config:          cfg,
		logger:          logger.WithField("component", "core_tool"),
		peerEventCounts: make(map[string]map[string]int),
	}

	_ = ctx // Context will be passed to individual methods as needed

	// Initialize components
	if err := tool.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return tool, nil
}

// initializeComponents sets up all the tool's dependencies.
func (t *DefaultTool) initializeComponents() error {
	// Initialize peer repository
	t.peerRepo = peer.NewInMemoryRepository(t.logger)

	// Initialize session manager
	t.sessionMgr = peer.NewSessionManager(t.peerRepo, t.logger)

	// Initialize report generator
	var err error

	t.reportGen, err = reports.NewGenerator(t.logger)
	if err != nil {
		return fmt.Errorf("failed to create report generator: %w", err)
	}

	// Initialize event manager
	t.eventMgr = events.NewManager(t, t.logger)

	// Register default event handlers
	if err := t.eventMgr.RegisterDefaultHandlers(); err != nil {
		return fmt.Errorf("failed to register event handlers: %w", err)
	}

	// Initialize Hermes controller
	t.hermesCtrl = NewHermesController(t.config, t.logger)

	return nil
}

// Start begins the peer scoring test.
func (t *DefaultTool) Start(ctx context.Context) error {
	t.startTime = time.Now()
	t.logger.Info("Starting peer score tool")

	// Start Hermes
	if err := t.hermesCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Hermes: %w", err)
	}

	// Register event callback
	t.hermesCtrl.RegisterEventCallback(t.handleEvent)

	// Start status reporting
	go t.startStatusReporting(ctx)

	// Wait for test duration or context cancellation
	testDuration := t.config.GetTestDuration()
	t.logger.WithField("duration", testDuration).Info("Running peer score test")

	select {
	case <-ctx.Done():
		t.logger.Info("Test interrupted by context cancellation")
	case <-time.After(testDuration):
		t.logger.Info("Test duration completed")
	}

	return nil
}

// Stop gracefully shuts down the tool.
func (t *DefaultTool) Stop() error {
	t.logger.Info("Stopping peer score tool")

	if t.hermesCtrl != nil {
		if err := t.hermesCtrl.Stop(); err != nil {
			t.logger.WithError(err).Error("Error stopping Hermes controller")
		}
	}

	return nil
}

// GenerateReport creates the final peer score report.
func (t *DefaultTool) GenerateReport() (*Report, error) {
	t.logger.Info("Generating peer score report")

	endTime := time.Now()
	duration := endTime.Sub(t.startTime)

	// Get all peer data
	peers := t.peerRepo.GetAllPeers()
	eventCounts := t.peerRepo.GetPeerEventCounts()

	// Calculate statistics
	calculator := peer.NewStatsCalculator()
	connectionStats := calculator.CalculateConnectionStats(peers)

	// Convert peers to map[string]interface{} for report
	peerData := make(map[string]interface{})
	for peerID, peerStats := range peers {
		peerData[peerID] = peerStats
	}

	report := &Report{
		Config:               t.config,
		ValidationMode:       string(t.config.GetValidationMode()),
		Timestamp:            endTime,
		StartTime:            t.startTime,
		EndTime:              endTime,
		Duration:             duration,
		TotalConnections:     connectionStats.TotalConnections,
		SuccessfulHandshakes: connectionStats.SuccessfulHandshakes,
		FailedHandshakes:     connectionStats.FailedHandshakes,
		Peers:                peerData,
		PeerEventCounts:      eventCounts,
	}

	t.logger.WithFields(logrus.Fields{
		"total_connections":     connectionStats.TotalConnections,
		"successful_handshakes": connectionStats.SuccessfulHandshakes,
		"failed_handshakes":     connectionStats.FailedHandshakes,
		"unique_peers":          len(peers),
		"test_duration":         duration,
	}).Info("Report generation complete")

	return report, nil
}

// GetLogger returns the tool's logger.
func (t *DefaultTool) GetLogger() logrus.FieldLogger {
	return t.logger
}

// GetConfig returns the tool's configuration.
func (t *DefaultTool) GetConfig() Config {
	return t.config
}

// handleEvent processes events from Hermes.
func (t *DefaultTool) handleEvent(ctx context.Context, event interface{}) error {
	// This will be called by the Hermes controller when events are received
	if hermesEvent, ok := event.(*host.TraceEvent); ok {
		// Pass event to event manager for processing
		return t.eventMgr.HandleEvent(ctx, hermesEvent)
	}

	return fmt.Errorf("unsupported event type: %T", event)
}

// startStatusReporting provides periodic updates on peer connection status.
func (t *DefaultTool) startStatusReporting(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.logCurrentStatus()
		}
	}
}

// logCurrentStatus logs the current peer connection statistics.
func (t *DefaultTool) logCurrentStatus() {
	peers := t.peerRepo.GetAllPeers()
	peerCount := len(peers)

	// Count active peers manually
	activeCount := 0

	for _, peer := range peers {
		for _, session := range peer.ConnectionSessions {
			if !session.Disconnected {
				activeCount++

				break
			}
		}
	}

	t.logger.WithFields(logrus.Fields{
		"peer_count":   peerCount,
		"active_peers": activeCount,
	}).Info("Status report")
}

func (t *DefaultTool) GetPeer(peerID string) (interface{}, bool) {
	peer, exists := t.peerRepo.GetPeer(peerID)

	return peer, exists
}

func (t *DefaultTool) CreatePeer(peerID string) interface{} {
	return t.peerRepo.CreatePeer(peerID)
}

func (t *DefaultTool) UpdatePeer(peerID string, updateFn func(interface{})) {
	t.peerRepo.UpdatePeer(peerID, func(peer *peer.Stats) {
		updateFn(peer)
	})
}

func (t *DefaultTool) UpdateOrCreatePeer(peerID string, updateFn func(interface{})) {
	t.peerRepo.UpdateOrCreatePeer(peerID, func(peer *peer.Stats) {
		updateFn(peer)
	})
}

func (t *DefaultTool) IncrementEventCount(peerID, eventType string) {
	t.peerRepo.IncrementEventCount(peerID, eventType)
}

func (t *DefaultTool) IncrementMessageCount(peerID string) {
	if err := t.sessionMgr.IncrementMessageCount(peerID); err != nil {
		t.logger.WithError(err).WithField("peer_id", peerID).Debug("Failed to increment message count")
	}
}

// SaveReports generates and saves both JSON and HTML reports.
func (t *DefaultTool) SaveReports() error {
	report, err := t.GenerateReport()
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Get validation config details for the report
	validationConfigs := config.GetValidationConfigs()
	validationConfig := validationConfigs[t.config.GetValidationMode()]

	// Convert to reports package format
	reportsReport := &reports.Report{
		Config:         report.Config,
		ValidationMode: report.ValidationMode,
		ValidationConfig: map[string]interface{}{
			"mode":          string(t.config.GetValidationMode()),
			"HermesVersion": validationConfig.HermesVersion,
		},
		Timestamp:            report.Timestamp,
		StartTime:            report.StartTime,
		EndTime:              report.EndTime,
		Duration:             report.Duration,
		TotalConnections:     report.TotalConnections,
		SuccessfulHandshakes: report.SuccessfulHandshakes,
		FailedHandshakes:     report.FailedHandshakes,
		Peers:                report.Peers,
		PeerEventCounts:      report.PeerEventCounts,
	}

	// Save JSON report
	jsonFile, err := t.reportGen.GenerateJSON(reportsReport)
	if err != nil {
		return fmt.Errorf("failed to save JSON report: %w", err)
	}

	// Check for AI analysis API key
	apiKey := t.config.GetClaudeAPIKey()
	if apiKey == "" {
		// Also check environment variable as fallback
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	// Save HTML report with or without AI analysis
	var htmlFile string

	if apiKey != "" && !t.config.IsSkipAI() {
		t.logger.Info("Including AI analysis in HTML report")

		htmlFile, err = t.reportGen.GenerateHTMLWithAI(reportsReport, apiKey)
	} else {
		htmlFile, err = t.reportGen.GenerateHTML(reportsReport)
	}

	if err != nil {
		return fmt.Errorf("failed to save HTML report: %w", err)
	}

	t.logger.WithFields(logrus.Fields{
		"json_file": jsonFile,
		"html_file": htmlFile,
	}).Info("Reports saved successfully")

	return nil
}
