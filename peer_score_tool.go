package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os/exec"
	"regexp"
	"sort"
	"sync"
	"syscall"
	"time"
)

// PeerScoreTool manages the peer scoring test execution and data collection.
// It orchestrates the Hermes process, parses logs in real-time, and aggregates
// peer connection statistics for scoring and analysis.
type PeerScoreTool struct {
	config     PeerScoreConfig           // Test configuration and parameters.
	hermesCmd  *exec.Cmd                 // Handle to the running Hermes process.
	logRegexes map[string]*regexp.Regexp // Compiled regex patterns for log parsing.
	mu         sync.RWMutex              // Protects concurrent access to peer data.
	peers      map[string]*PeerStats     // Individual peer statistics indexed by peer ID.
	startTime  time.Time                 // When the test execution began.

	// Global counters for reporting.
	totalGoodbyes    int                       // Total goodbye messages received across all peers.
	goodbyeReasons   map[string]int            // Aggregated goodbye reasons and their counts.
	goodbyesByClient map[string]map[string]int // Goodbye reasons grouped by client type.

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
			"connected":  regexp.MustCompile(`Connected with peer.*peer_id=(\w+)`),                             // New peer connections.
			"handshake":  regexp.MustCompile(`Performed successful handshake.*peer_id=(\w+).*agent=([^,\s]+)`), // Successful handshakes with client info.
			"goodbye":    regexp.MustCompile(`Received goodbye message.*peer_id=(\w+).*msg="([^"]+)"`),         // Peer disconnection messages.
			"disconnect": regexp.MustCompile(`Disconnected from handshaked peer.*peer_id=(\w+)`),               // Peer disconnections.
		},
	}

	return tool
}

// StartHermes starts the Hermes process and begins real-time log parsing.
// It launches Hermes as a subprocess, captures its output streams, and starts
// goroutines to parse logs for peer events and statistics collection.
func (pst *PeerScoreTool) StartHermes(ctx context.Context) error {
	// Start Hermes directly with the provided arguments.
	//nolint:gosec // Controlled input.
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
		diversityScore := float64(minInt(report.UniqueClients, 4)) / 4.0 * 100

		// Calculate goodbye penalty for ERROR-level messages.
		errorGoodbyes := 0

		for reason, count := range report.GoodbyeReasons {
			if pst.classifyGoodbyeSeverity(reason) == SeverityError {
				errorGoodbyes += count
			}
		}

		// Apply penalty: 5 points deducted per ERROR-level goodbye message.
		goodbyePenalty := float64(errorGoodbyes) * 5.0

		// Combine connection and diversity scores, then apply goodbye penalty.
		baseScore := (connectionScore + diversityScore) / 2
		report.OverallScore = maxFloat(0.0, baseScore-goodbyePenalty)
	} else {
		report.OverallScore = 0.0
	}

	// Perform timing correlation analysis
	report.TimingAnalysis = pst.AnalyzeConnectionTiming()
	
	// Generate downscore indicators
	report.DownscoreIndicators = pst.generateDownscoreIndicators(report.TimingAnalysis)

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
	
	// Log timing insights
	if len(report.TimingAnalysis.SuspiciousPatterns) > 0 {
		log.Printf("Suspicious patterns detected:")
		for _, pattern := range report.TimingAnalysis.SuspiciousPatterns {
			log.Printf("  - %s", pattern)
		}
	}
	
	if len(report.TimingAnalysis.ClientTimingPatterns) > 0 {
		log.Printf("Timing patterns detected:")
		for _, pattern := range report.TimingAnalysis.ClientTimingPatterns {
			log.Printf("  - %s", pattern.Pattern)
		}
	}
	
	// Log client-specific scoring patterns
	clientPatterns := pst.analyzeClientSpecificScoringPatterns()
	if len(clientPatterns) > 0 {
		log.Printf("Client-specific patterns detected:")
		for clientType, patterns := range clientPatterns {
			log.Printf("  %s:", clientType)
			for _, pattern := range patterns {
				log.Printf("    - %s", pattern)
			}
		}
	}

	return report
}

// classifyGoodbyeSeverity categorizes goodbye reasons by their severity level.
// NORMAL reasons are expected network behavior, ERROR reasons indicate problems.
func (pst *PeerScoreTool) classifyGoodbyeSeverity(reason string) string {
	switch reason {
	case "client has too many peers":
		return SeverityNormal // Normal network behavior, client is full.
	case "client shutdown":
		return SeverityNormal // Normal behavior, peer is shutting down.
	case "peer score too low":
		return SeverityError // Our node is being rejected due to low reputation.
	case "client banned this node":
		return SeverityError // Our node has been banned by the peer.
	case "irrelevant network":
		return SeverityError // Network configuration mismatch.
	case "unable to verify network":
		return SeverityError // Network verification failure.
	case "fault/error":
		return SeverityError // General error condition.
	default:
		return SeverityUnknown // Unrecognized goodbye reason.
	}
}

// Helper functions for mathematical operations.

// minInt returns the smaller of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// maxFloat returns the larger of two floating-point numbers.
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

// AnalyzeConnectionTiming performs comprehensive timing correlation analysis
func (pst *PeerScoreTool) AnalyzeConnectionTiming() ConnectionTiming {
	pst.mu.RLock()
	defer pst.mu.RUnlock()
	
	analysis := ConnectionTiming{
		TotalConnections:      0,
		GoodbyeReasonTimings:  make(map[string]GoodbyeReasonTiming),
		ClientTimingPatterns:  make([]TimingPattern, 0),
		SuspiciousPatterns:    make([]string, 0),
	}
	
	// Collect all connection durations and goodbye timings
	var allDurations []time.Duration
	goodbyeTimings := make(map[string][]time.Duration) // reason -> durations
	clientGoodbyeTimings := make(map[string]map[string][]time.Duration) // client -> reason -> durations
	
	for _, peer := range pst.peers {
		if peer.ConnectedAt.IsZero() {
			continue // Skip peers without connection timing
		}
		
		analysis.TotalConnections++
		
		// Use the connection duration (calculated when peer disconnected or said goodbye)
		duration := peer.ConnectionDuration
		if duration > 0 {
			allDurations = append(allDurations, duration)
		}
		
		// Analyze goodbye timings for this peer
		for _, goodbye := range peer.GoodbyeTimings {
			if goodbye.DurationFromStart > 0 {
				// Track by goodbye reason
				if goodbyeTimings[goodbye.Reason] == nil {
					goodbyeTimings[goodbye.Reason] = make([]time.Duration, 0)
				}
				goodbyeTimings[goodbye.Reason] = append(goodbyeTimings[goodbye.Reason], goodbye.DurationFromStart)
				
				// Track by client and goodbye reason
				if clientGoodbyeTimings[peer.ClientType] == nil {
					clientGoodbyeTimings[peer.ClientType] = make(map[string][]time.Duration)
				}
				if clientGoodbyeTimings[peer.ClientType][goodbye.Reason] == nil {
					clientGoodbyeTimings[peer.ClientType][goodbye.Reason] = make([]time.Duration, 0)
				}
				clientGoodbyeTimings[peer.ClientType][goodbye.Reason] = append(
					clientGoodbyeTimings[peer.ClientType][goodbye.Reason], 
					goodbye.DurationFromStart,
				)
			}
		}
	}
	
	// Calculate overall connection duration statistics
	if len(allDurations) > 0 {
		sort.Slice(allDurations, func(i, j int) bool {
			return allDurations[i] < allDurations[j]
		})
		
		analysis.FastestDisconnect = allDurations[0]
		analysis.LongestConnection = allDurations[len(allDurations)-1]
		analysis.MedianConnectionDuration = allDurations[len(allDurations)/2]
		
		total := time.Duration(0)
		for _, d := range allDurations {
			total += d
		}
		analysis.AverageConnectionDuration = total / time.Duration(len(allDurations))
	}
	
	// Analyze goodbye reason timings
	for reason, durations := range goodbyeTimings {
		if len(durations) == 0 {
			continue
		}
		
		reasonTiming := pst.analyzeGoodbyeReasonTiming(reason, durations, clientGoodbyeTimings)
		analysis.GoodbyeReasonTimings[reason] = reasonTiming
	}
	
	// Detect client-specific timing patterns
	analysis.ClientTimingPatterns = pst.detectTimingPatterns(clientGoodbyeTimings)
	
	// Identify suspicious patterns that might indicate downscoring
	analysis.SuspiciousPatterns = pst.identifySuspiciousPatterns(analysis)
	
	return analysis
}

// analyzeGoodbyeReasonTiming analyzes timing patterns for a specific goodbye reason
func (pst *PeerScoreTool) analyzeGoodbyeReasonTiming(reason string, durations []time.Duration, clientBreakdown map[string]map[string][]time.Duration) GoodbyeReasonTiming {
	if len(durations) == 0 {
		return GoodbyeReasonTiming{Reason: reason, Count: 0}
	}
	
	// Sort durations for statistical analysis
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	
	// Calculate basic statistics
	total := time.Duration(0)
	for _, d := range durations {
		total += d
	}
	average := total / time.Duration(len(durations))
	median := durations[len(durations)/2]
	
	
	// Count by client type
	clientCounts := make(map[string]int)
	for clientType, reasons := range clientBreakdown {
		if reasonDurations, exists := reasons[reason]; exists {
			clientCounts[clientType] = len(reasonDurations)
		}
	}
	
	return GoodbyeReasonTiming{
		Reason:            reason,
		Count:             len(durations),
		AverageDuration:   average,
		MedianDuration:    median,
		MinDuration:       durations[0],
		MaxDuration:       durations[len(durations)-1],
		ClientBreakdown:   clientCounts,
	}
}

// detectTimingPatterns identifies client-specific timing patterns
func (pst *PeerScoreTool) detectTimingPatterns(clientGoodbyeTimings map[string]map[string][]time.Duration) []TimingPattern {
	patterns := make([]TimingPattern, 0)
	
	for clientType, reasonTimings := range clientGoodbyeTimings {
		for reason, durations := range reasonTimings {
			if len(durations) < 3 { // Need at least 3 occurrences to detect a pattern
				continue
			}
			
			// Calculate statistics for this client+reason combination
			sort.Slice(durations, func(i, j int) bool {
				return durations[i] < durations[j]
			})
			
			total := time.Duration(0)
			for _, d := range durations {
				total += d
			}
			average := total / time.Duration(len(durations))
			
			// Detect patterns based on timing consistency
			pattern := pst.classifyTimingPattern(clientType, reason, average, len(durations))
			if pattern != "" {
				patterns = append(patterns, TimingPattern{
					ClientType:        clientType,
					GoodbyeReason:     reason,
					AverageDuration:   average,
					Occurrences:       len(durations),
					Pattern:           pattern,
				})
			}
		}
	}
	
	return patterns
}

// classifyTimingPattern classifies timing patterns and returns a description
func (pst *PeerScoreTool) classifyTimingPattern(clientType, reason string, avg time.Duration, count int) string {
	
	// Specific time intervals that might indicate timeouts or scoring intervals
	avgSeconds := avg.Seconds()
	
	// Check for common timeout patterns (multiples of 30s, 60s, 120s, 240s)
	timeoutIntervals := []float64{30, 60, 120, 240, 300, 600} // Common timeout values
	for _, interval := range timeoutIntervals {
		if math.Abs(avgSeconds-interval) < 5 && count >= 3 { // Within 5 seconds of common timeout
			return fmt.Sprintf("Timeout pattern: %s peers disconnect with '%s' around %v mark (%d occurrences)", 
				clientType, reason, time.Duration(interval)*time.Second, count)
		}
	}
	
	// Fast disconnects might indicate immediate scoring/banning
	if avg < 10*time.Second && count >= 3 {
		return fmt.Sprintf("Fast rejection: %s peers send '%s' very quickly (avg %v, %d occurrences)", 
			clientType, reason, avg.Round(time.Second), count)
	}
	
	return "" // No notable pattern detected
}

// identifySuspiciousPatterns identifies patterns that might indicate peer downscoring
func (pst *PeerScoreTool) identifySuspiciousPatterns(analysis ConnectionTiming) []string {
	suspicious := make([]string, 0)
	
	// Check for high frequency of error-level goodbye reasons
	errorGoodbyes := 0
	totalGoodbyes := 0
	for reason, timing := range analysis.GoodbyeReasonTimings {
		totalGoodbyes += timing.Count
		if pst.classifyGoodbyeSeverity(reason) == SeverityError {
			errorGoodbyes += timing.Count
		}
	}
	
	if totalGoodbyes > 0 {
		errorRate := float64(errorGoodbyes) / float64(totalGoodbyes) * 100
		if errorRate > 30 { // More than 30% error-level goodbyes
			suspicious = append(suspicious, fmt.Sprintf("High error rate: %.1f%% of goodbyes are error-level (%d/%d)", 
				errorRate, errorGoodbyes, totalGoodbyes))
		}
	}
	
	// Check for patterns indicating peer score issues
	if peerScoreTiming, exists := analysis.GoodbyeReasonTimings["peer score too low"]; exists {
		if peerScoreTiming.Count > 2 { // Multiple peer score rejections
			suspicious = append(suspicious, fmt.Sprintf("Multiple peer score rejections: %d peers rejected us for low score (avg after %v)", 
				peerScoreTiming.Count, peerScoreTiming.AverageDuration.Round(time.Second)))
		}
	}
	
	if bannedTiming, exists := analysis.GoodbyeReasonTimings["client banned this node"]; exists {
		if bannedTiming.Count > 1 { // Multiple bans
			suspicious = append(suspicious, fmt.Sprintf("Multiple bans detected: %d peers banned this node (avg after %v)", 
				bannedTiming.Count, bannedTiming.AverageDuration.Round(time.Second)))
		}
	}
	
	// Check for very fast disconnects which might indicate immediate rejection
	if analysis.FastestDisconnect > 0 && analysis.FastestDisconnect < 5*time.Second {
		fastCount := 0
		for _, timing := range analysis.GoodbyeReasonTimings {
			if timing.MinDuration < 5*time.Second {
				fastCount += timing.Count
			}
		}
		if fastCount > analysis.TotalConnections/4 { // More than 25% of connections are very fast
			suspicious = append(suspicious, fmt.Sprintf("Many fast disconnects: %d peers disconnect within 5 seconds", fastCount))
		}
	}
	
	return suspicious
}


// generateDownscoreIndicators generates specific indicators suggesting peer downscoring
func (pst *PeerScoreTool) generateDownscoreIndicators(analysis ConnectionTiming) []string {
	indicators := make([]string, 0)
	
	// Direct peer scoring indicators
	if peerScoreTiming, exists := analysis.GoodbyeReasonTimings["peer score too low"]; exists {
		indicators = append(indicators, fmt.Sprintf("Direct peer score rejections: %d peers rejected us for low peer score", peerScoreTiming.Count))
		
		if peerScoreTiming.AverageDuration < 30*time.Second {
			indicators = append(indicators, "Fast peer score rejections suggest immediate reputation issues")
		}
	}
	
	if bannedTiming, exists := analysis.GoodbyeReasonTimings["client banned this node"]; exists {
		indicators = append(indicators, fmt.Sprintf("Node bans detected: %d peers banned this node", bannedTiming.Count))
	}
	
	// Pattern-based indicators
	for _, pattern := range analysis.ClientTimingPatterns {
		if pattern.GoodbyeReason == "peer score too low" || pattern.GoodbyeReason == "client banned this node" {
			indicators = append(indicators, fmt.Sprintf("Consistent %s pattern: %s", pattern.GoodbyeReason, pattern.Pattern))
		}
		
		// Very fast consistent disconnects might indicate reputation issues
		if pattern.AverageDuration < 10*time.Second && pattern.Occurrences >= 3 {
			indicators = append(indicators, fmt.Sprintf("Suspicious fast pattern: %s", pattern.Pattern))
		}
	}
	
	// Network verification issues might indicate configuration problems affecting score
	if networkTiming, exists := analysis.GoodbyeReasonTimings["unable to verify network"]; exists {
		if networkTiming.Count > 2 {
			indicators = append(indicators, fmt.Sprintf("Network verification failures: %d peers unable to verify network", networkTiming.Count))
		}
	}
	
	if irrelevantTiming, exists := analysis.GoodbyeReasonTimings["irrelevant network"]; exists {
		if irrelevantTiming.Count > 2 {
			indicators = append(indicators, fmt.Sprintf("Network relevance issues: %d peers consider our network irrelevant", irrelevantTiming.Count))
		}
	}
	
	// High proportion of error-level goodbyes suggests systematic issues
	errorGoodbyes := 0
	totalGoodbyes := 0
	for reason, timing := range analysis.GoodbyeReasonTimings {
		totalGoodbyes += timing.Count
		if pst.classifyGoodbyeSeverity(reason) == SeverityError {
			errorGoodbyes += timing.Count
		}
	}
	
	if totalGoodbyes > 0 {
		errorRate := float64(errorGoodbyes) / float64(totalGoodbyes) * 100
		if errorRate > 50 {
			indicators = append(indicators, fmt.Sprintf("High error rate: %.1f%% of disconnections are error-level", errorRate))
		}
	}
	
	// Pattern-based indicators are already included in the timing patterns
	
	return indicators
}

// analyzeClientSpecificScoringPatterns performs detailed client-specific analysis for scoring vulnerabilities  
func (pst *PeerScoreTool) analyzeClientSpecificScoringPatterns() map[string][]string {
	pst.mu.RLock()
	defer pst.mu.RUnlock()
	
	clientPatterns := make(map[string][]string)
	clientStats := make(map[string]map[string]int) // client -> reason -> count
	clientTimings := make(map[string]map[string][]time.Duration) // client -> reason -> durations
	
	// Collect client-specific data
	for _, peer := range pst.peers {
		if peer.ClientType == "" || peer.ClientType == "unknown" {
			continue
		}
		
		if clientStats[peer.ClientType] == nil {
			clientStats[peer.ClientType] = make(map[string]int)
			clientTimings[peer.ClientType] = make(map[string][]time.Duration)
		}
		
		// Track goodbye reasons and timings by client
		for _, goodbye := range peer.GoodbyeTimings {
			clientStats[peer.ClientType][goodbye.Reason]++
			
			if goodbye.DurationFromStart > 0 {
				if clientTimings[peer.ClientType][goodbye.Reason] == nil {
					clientTimings[peer.ClientType][goodbye.Reason] = make([]time.Duration, 0)
				}
				clientTimings[peer.ClientType][goodbye.Reason] = append(
					clientTimings[peer.ClientType][goodbye.Reason], 
					goodbye.DurationFromStart,
				)
			}
		}
	}
	
	// Analyze patterns for each client
	for clientType, reasons := range clientStats {
		patterns := make([]string, 0)
		
		// Check for high error rates per client
		totalGoodbyes := 0
		errorGoodbyes := 0
		
		for reason, count := range reasons {
			totalGoodbyes += count
			if pst.classifyGoodbyeSeverity(reason) == SeverityError {
				errorGoodbyes += count
			}
		}
		
		if totalGoodbyes > 0 {
			errorRate := float64(errorGoodbyes) / float64(totalGoodbyes) * 100
			if errorRate > 60 { // High error rate for this client
				patterns = append(patterns, fmt.Sprintf("High error rate: %.1f%% of %s disconnections are error-level (%d/%d)", 
					errorRate, clientType, errorGoodbyes, totalGoodbyes))
			}
		}
		
		// Check for specific scoring issues per client
		if count, exists := reasons["peer score too low"]; exists && count > 1 {
			if timings, hasTimings := clientTimings[clientType]["peer score too low"]; hasTimings && len(timings) > 0 {
				sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
				avgTime := time.Duration(0)
				for _, t := range timings {
					avgTime += t
				}
				avgTime /= time.Duration(len(timings))
				
				patterns = append(patterns, fmt.Sprintf("Peer score rejections: %d %s peers rejected us (avg after %v)", 
					count, clientType, avgTime.Round(time.Second)))
			}
		}
		
		if count, exists := reasons["client banned this node"]; exists && count > 0 {
			patterns = append(patterns, fmt.Sprintf("Node bans: %d %s peers banned this node", count, clientType))
		}
		
		// Check for suspicious timing patterns per client
		for reason, timings := range clientTimings[clientType] {
			if len(timings) < 3 {
				continue
			}
			
			sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
			
			// Calculate consistency
			total := time.Duration(0)
			for _, t := range timings {
				total += t
			}
			avg := total / time.Duration(len(timings))
			
			variance := float64(0)
			for _, t := range timings {
				diff := float64(t - avg)
				variance += diff * diff
			}
			variance /= float64(len(timings))
			stdDev := time.Duration(math.Sqrt(variance))
			
			// Very consistent timing might indicate algorithmic behavior
			if stdDev < avg/15 && len(timings) >= 4 { // StdDev < ~6.7% of average
				patterns = append(patterns, fmt.Sprintf("Highly consistent '%s' timing: %d occurrences at %v (±%v)", 
					reason, len(timings), avg.Round(time.Second), stdDev.Round(time.Millisecond)))
			}
			
			// Check for timeout-like patterns
			avgSeconds := avg.Seconds()
			timeoutIntervals := []float64{30, 60, 120, 240, 300} // Common timeout values
			for _, interval := range timeoutIntervals {
				if math.Abs(avgSeconds-interval) < 3 && len(timings) >= 3 { // Within 3 seconds of timeout
					patterns = append(patterns, fmt.Sprintf("Timeout pattern: %s '%s' consistently around %vs (%d occurrences)", 
						clientType, reason, interval, len(timings)))
				}
			}
		}
		
		// Check for client dominance in "too many peers" messages
		if count, exists := reasons["client has too many peers"]; exists {
			if float64(count)/float64(totalGoodbyes) > 0.8 { // More than 80% are "too many peers"
				patterns = append(patterns, fmt.Sprintf("Capacity issues: %d/%d (%d%%) of %s disconnections are 'too many peers'", 
					count, totalGoodbyes, int(float64(count)/float64(totalGoodbyes)*100), clientType))
			}
		}
		
		if len(patterns) > 0 {
			clientPatterns[clientType] = patterns
		}
	}
	
	return clientPatterns
}
