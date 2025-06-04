package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

// PeerScoreTool manages the peer scoring test execution and data collection.
// It orchestrates the Hermes process, parses logs in real-time, and aggregates
// peer connection statistics for scoring and analysis.
type PeerScoreTool struct {
	config     PeerScoreConfig                 // Test configuration and parameters.
	hermesCmd  *exec.Cmd                       // Handle to the running Hermes process.
	logRegexes map[string]*regexp.Regexp       // Compiled regex patterns for log parsing.
	mu         sync.RWMutex                    // Protects concurrent access to peer data.
	peers      map[string]*PeerStats           // Individual peer statistics indexed by peer ID.
	startTime  time.Time                       // When the test execution began.

	// Global counters for reporting.
	totalGoodbyes    int                            // Total goodbye messages received across all peers.
	goodbyeReasons   map[string]int                 // Aggregated goodbye reasons and their counts.
	goodbyesByClient map[string]map[string]int      // Goodbye reasons grouped by client type.

	// Error tracking.
	errors           []string // List of errors encountered during test execution.
	connectionFailed bool     // Whether the beacon node connection failed.
}

// NewPeerScoreTool creates a new peer score tool instance with the given configuration.
// It initializes all internal data structures and compiles regex patterns for log parsing.
func NewPeerScoreTool(config PeerScoreConfig) *PeerScoreTool {
	tool := &PeerScoreTool{
		config:           config,
		peers:            make(map[string]*PeerStats),
		goodbyeReasons:   make(map[string]int),
		goodbyesByClient: make(map[string]map[string]int),
		errors:           make([]string, 0),
		connectionFailed: false,
		// Pre-compiled regex patterns for efficient log parsing.
		logRegexes: map[string]*regexp.Regexp{
			"connected":  regexp.MustCompile(`Connected with peer.*peer_id=(\w+)`),                           // New peer connections.
			"handshake":  regexp.MustCompile(`Performed successful handshake.*peer_id=(\w+).*agent=([^,\s]+)`), // Successful handshakes with client info.
			"goodbye":    regexp.MustCompile(`Received goodbye message.*peer_id=(\w+).*msg="([^"]+)"`),        // Peer disconnection messages.
			"disconnect": regexp.MustCompile(`Disconnected from handshaked peer.*peer_id=(\w+)`),             // Peer disconnections.
		},
	}

	return tool
}

// StartHermes starts the Hermes process and begins real-time log parsing.
// It launches Hermes as a subprocess, captures its output streams, and starts
// goroutines to parse logs for peer events and statistics collection.
func (pst *PeerScoreTool) StartHermes(ctx context.Context) error {
	// Start Hermes directly with the provided arguments.
	pst.hermesCmd = exec.CommandContext(ctx, pst.config.HermesPath, pst.config.HermesArgs...)

	// Capture stdout and stderr for log parsing.
	stdout, err := pst.hermesCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := pst.hermesCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := pst.hermesCmd.Start(); err != nil {
		return fmt.Errorf("failed to start hermes: %w", err)
	}

	// Start enhanced log parsing in separate goroutines.
	// This allows real-time processing of both stdout and stderr streams.
	parser := NewEnhancedLogParser(pst)
	go parser.StartParsing(ctx, stdout)
	go parser.StartParsing(ctx, stderr)

	pst.startTime = time.Now()
	log.Printf("Started Hermes with PID %d", pst.hermesCmd.Process.Pid)

	return nil
}

// Stop terminates the Hermes process gracefully using SIGTERM.
// This should be called during cleanup to ensure the subprocess is properly terminated.
func (pst *PeerScoreTool) Stop() error {
	if pst.hermesCmd != nil && pst.hermesCmd.Process != nil {
		log.Printf("Stopping Hermes (PID %d)", pst.hermesCmd.Process.Pid)
		return pst.hermesCmd.Process.Signal(syscall.SIGTERM)
	}
	return nil
}

// GenerateReport creates the final peer score report with comprehensive analysis.
// It aggregates all collected data, calculates scores, and produces a detailed
// report suitable for both JSON serialization and further analysis.
func (pst *PeerScoreTool) GenerateReport() PeerScoreReport {
	log.Println("Generating peer score report...")

	endTime := time.Now()
	duration := endTime.Sub(pst.startTime)

	// Acquire read lock to safely access peer data during report generation.
	pst.mu.RLock()
	defer pst.mu.RUnlock()

	report := PeerScoreReport{
		Timestamp:        time.Now(),
		Config:           pst.config,
		StartTime:        pst.startTime,
		EndTime:          endTime,
		Duration:         duration,
		TotalConnections: len(pst.peers),
		GoodbyeMessages:  pst.totalGoodbyes,
		GoodbyeReasons:   make(map[string]int),
		GoodbyesByClient: make(map[string]map[string]int),
		PeersByClient:    make(map[string]int),
		Peers:            make(map[string]*PeerStats),
		Errors:           make([]string, len(pst.errors)),
		ConnectionFailed: pst.connectionFailed,
	}

	// Copy goodbye reasons from internal counters.
	for reason, count := range pst.goodbyeReasons {
		report.GoodbyeReasons[reason] = count
	}

	// Copy goodbye by client data for detailed analysis.
	for client, reasons := range pst.goodbyesByClient {
		report.GoodbyesByClient[client] = make(map[string]int)
		for reason, count := range reasons {
			report.GoodbyesByClient[client][reason] = count
		}
	}

	// Copy accumulated errors.
	copy(report.Errors, pst.errors)

	// Process individual peer data and calculate aggregated metrics.
	for id, peer := range pst.peers {
		// Copy peer data to report.
		report.Peers[id] = peer

		// Count handshake success/failure rates.
		if peer.HandshakeOK {
			report.SuccessfulHandshakes++
		} else {
			report.FailedHandshakes++
		}

		// Count peers by client implementation.
		if peer.ClientType != "" {
			report.PeersByClient[peer.ClientType]++
		}
	}

	// Count unique client implementations discovered.
	report.UniqueClients = len(report.PeersByClient)

	// Calculate overall score with connection success, diversity, and goodbye penalties.
	if report.ConnectionFailed {
		report.OverallScore = 0.0
	} else if report.TotalConnections > 0 {
		// Connection success rate (0-100%).
		connectionScore := float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
		
		// Client diversity score (0-100%, maxed at 4 different clients).
		diversityScore := float64(min(report.UniqueClients, 4)) / 4.0 * 100

		// Calculate goodbye penalty for ERROR-level messages.
		errorGoodbyes := 0
		for reason, count := range report.GoodbyeReasons {
			if pst.classifyGoodbyeSeverity(reason) == "ERROR" {
				errorGoodbyes += count
			}
		}

		// Apply penalty: 5 points deducted per ERROR-level goodbye message.
		goodbyePenalty := float64(errorGoodbyes) * 5.0

		// Combine connection and diversity scores, then apply goodbye penalty.
		baseScore := (connectionScore + diversityScore) / 2
		report.OverallScore = max(0.0, baseScore-goodbyePenalty)
	} else {
		report.OverallScore = 0.0
	}

	// Generate human-readable summary based on test results.
	if report.ConnectionFailed {
		report.Summary = fmt.Sprintf(
			"FAILED: Connection to beacon node failed | Errors: %d",
			len(report.Errors),
		)
	} else {
		report.Summary = fmt.Sprintf(
			"Score: %.1f%% | Connections: %d | Handshakes: %d | Clients: %d | Goodbyes: %d",
			report.OverallScore,
			report.TotalConnections,
			report.SuccessfulHandshakes,
			report.UniqueClients,
			report.GoodbyeMessages,
		)
	}

	log.Printf("Report completed: %s", report.Summary)

	return report
}

// classifyGoodbyeSeverity categorizes goodbye reasons by their severity level.
// NORMAL reasons are expected network behavior, ERROR reasons indicate problems.
func (pst *PeerScoreTool) classifyGoodbyeSeverity(reason string) string {
	switch reason {
	case "client has too many peers":
		return "NORMAL" // Normal network behavior, client is full.
	case "client shutdown":
		return "NORMAL" // Normal behavior, peer is shutting down.
	case "peer score too low":
		return "ERROR" // Our node is being rejected due to low reputation.
	case "client banned this node":
		return "ERROR" // Our node has been banned by the peer.
	case "irrelevant network":
		return "ERROR" // Network configuration mismatch.
	case "unable to verify network":
		return "ERROR" // Network verification failure.
	case "fault/error":
		return "ERROR" // General error condition.
	default:
		return "UNKNOWN" // Unrecognized goodbye reason.
	}
}

// Helper functions for mathematical operations.

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two floating-point numbers.
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}