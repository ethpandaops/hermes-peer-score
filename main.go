package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Configuration and command-line flags.
var (
	duration      = flag.Duration("duration", 2*time.Minute, "Test duration for peer scoring")
	outputFile    = flag.String("output", "peer-score-report.json", "Output file for results")
	prysmHost     = flag.String("prysm-host", "", "Prysm host connection string (required)")
	prysmHTTPPort = flag.Int("prysm-http-port", 443, "Prysm HTTP port")
	prysmGRPCPort = flag.Int("prysm-grpc-port", 443, "Prysm gRPC port")
	generateTestHTML = flag.Bool("test-html", false, "Generate test HTML report with sample data")
)

func main() {
	flag.Parse()

	// Check if we should generate test HTML
	if *generateTestHTML {
		testHTML()
		return
	}

	// Validate required parameters.
	if *prysmHost == "" {
		log.Fatal("prysm-host is required")
	}

	// Initialize peer score tool configuration.
	config := buildPeerScoreConfig()
	tool := NewPeerScoreTool(config)

	// Log connection settings for debugging.
	logConnectionSettings()

	// Set up graceful shutdown handling.
	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	// Execute the peer scoring test.
	runPeerScoreTest(ctx, tool)

	// Generate and save reports.
	generateReports(tool)
}

// buildPeerScoreConfig constructs the configuration for the peer score tool.
func buildPeerScoreConfig() PeerScoreConfig {
	hermesArgs := buildHermesArgs()

	return PeerScoreConfig{
		HermesPath:     "./hermes",
		TestDuration:   *duration,
		ReportInterval: 1 * time.Minute,
		HermesArgs:     hermesArgs,
	}
}

// buildHermesArgs constructs the command-line arguments for the Hermes process.
func buildHermesArgs() []string {
	args := []string{
		"--data.stream.type=callback",
		"--metrics=true",
		"eth",
		"--chain=mainnet",
		"--prysm.host=" + *prysmHost,
		"--prysm.port.http=" + strconv.Itoa(*prysmHTTPPort),
		"--prysm.port.grpc=" + strconv.Itoa(*prysmGRPCPort),
		"--devp2p.host=0.0.0.0",
		"--devp2p.port=31912",
		"--libp2p.host=0.0.0.0",
		"--libp2p.port=31912",
		"--subscription.topics=beacon_block",
	}

	// Add TLS flag if either HTTP or gRPC port is 443.
	if *prysmHTTPPort == 443 || *prysmGRPCPort == 443 {
		args = append(args, "--prysm.tls")
	}

	return args
}

// logConnectionSettings logs connection details with password redaction for security.
func logConnectionSettings() {
	redactedHost := redactPassword(*prysmHost)

	log.Printf("Connection settings:")
	log.Printf("  Prysm Host: %s", redactedHost)
	log.Printf("  HTTP Port: %d", *prysmHTTPPort)
	log.Printf("  gRPC Port: %d", *prysmGRPCPort)
	log.Printf("  TLS Enabled: %t", (*prysmHTTPPort == 443 || *prysmGRPCPort == 443))
}

// redactPassword redacts passwords from connection strings for secure logging.
func redactPassword(host string) string {
	if !strings.Contains(host, ":") || !strings.Contains(host, "@") {
		return host
	}

	// Format is user:pass@host, redact the password.
	parts := strings.Split(host, "@")
	if len(parts) != 2 {
		return host
	}

	userParts := strings.Split(parts[0], ":")
	if len(userParts) != 2 {
		return host
	}

	return userParts[0] + ":****@" + parts[1]
}

// setupGracefulShutdown configures signal handling for graceful shutdown.
func setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	return ctx, cancel
}

// runPeerScoreTest executes the main peer scoring test.
func runPeerScoreTest(ctx context.Context, tool *PeerScoreTool) {
	// Start the Hermes process.
	if err := tool.StartHermes(ctx); err != nil {
		log.Fatalf("Failed to start Hermes: %v", err)
	}

	defer func() {
		if err := tool.Stop(); err != nil {
			log.Printf("Error stopping tool: %v", err)
		}
	}()

	log.Printf("Running peer score tests for %v...", *duration)

	// Start periodic status reporting.
	go startStatusReporting(ctx, tool)

	// Wait for test completion or cancellation.
	select {
	case <-ctx.Done():
		log.Println("Test interrupted")
	case <-time.After(*duration):
		log.Println("Test duration completed")
	}
}

// startStatusReporting provides periodic updates on peer connection status.
func startStatusReporting(ctx context.Context, tool *PeerScoreTool) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reportCurrentStatus(tool)
		}
	}
}

// reportCurrentStatus logs the current peer connection statistics.
func reportCurrentStatus(tool *PeerScoreTool) {
	tool.mu.RLock()
	defer tool.mu.RUnlock()

	peerCount := len(tool.peers)
	handshaked := 0

	for _, peer := range tool.peers {
		if peer.HandshakeOK {
			handshaked++
		}
	}

	log.Printf("Status: %d peers connected, %d handshaked", peerCount, handshaked)
}

// generateReports creates both JSON and HTML reports from the test results.
func generateReports(tool *PeerScoreTool) {
	// Generate the final peer score report.
	report := tool.GenerateReport()

	// Save JSON report to file.
	if err := saveJSONReport(report); err != nil {
		log.Fatalf("Failed to save JSON report: %v", err)
	}

	// Generate HTML report from JSON.
	if err := generateHTMLReport(); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	}

	// Print summary to console.
	printReportSummary(report)
}

// saveJSONReport marshals and saves the report as JSON.
func saveJSONReport(report PeerScoreReport) error {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	//nolint:gosec // Controlled input.
	if err := os.WriteFile(*outputFile, reportJSON, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

// generateHTMLReport creates an HTML version of the JSON report.
func generateHTMLReport() error {
	htmlFile := strings.Replace(*outputFile, ".json", ".html", 1)

	return GenerateHTMLReport(*outputFile, htmlFile)
}

// printReportSummary displays a comprehensive summary of the test results.
func printReportSummary(report PeerScoreReport) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PEER SCORE REPORT")
	fmt.Println(strings.Repeat("=", 60))

	// Core metrics.
	fmt.Printf("Overall Score: %.1f%%\n", report.OverallScore)
	fmt.Printf("Test Duration: %v\n", report.Duration)
	fmt.Printf("Total Connections: %d\n", report.TotalConnections)
	fmt.Printf("Successful Handshakes: %d\n", report.SuccessfulHandshakes)
	fmt.Printf("Failed Handshakes: %d\n", report.FailedHandshakes)
	fmt.Printf("Goodbye Messages: %d\n", report.GoodbyeMessages)

	// Goodbye reasons breakdown.
	if len(report.GoodbyeReasons) > 0 {
		fmt.Println("Goodbye Reasons:")

		for reason, count := range report.GoodbyeReasons {
			fmt.Printf("  %s: %d\n", reason, count)
		}
	}

	// Client diversity metrics.
	fmt.Printf("Unique Clients: %d\n", report.UniqueClients)
	fmt.Println("Client Distribution:")

	for client, count := range report.PeersByClient {
		fmt.Printf("  %s: %d\n", client, count)
	}

	// Error reporting.
	if len(report.Errors) > 0 {
		fmt.Printf("Errors Encountered: %d\n", len(report.Errors))

		for i, err := range report.Errors {
			fmt.Printf("  [%d] %s\n", i+1, err)
		}
	}

	// Connection failure warning.
	if report.ConnectionFailed {
		fmt.Println("WARNING: Connection to beacon node failed!")
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Report saved to: %s\n", *outputFile)
}
