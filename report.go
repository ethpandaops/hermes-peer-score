package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// generateReports creates both JSON and HTML reports from the test results.
func generateReports(ctx context.Context, log logrus.FieldLogger, tool *PeerScoreTool) {
	// Generate the final peer score report.
	report := tool.GenerateReport()

	// Save JSON report to file.
	if err := saveJSONReport(report); err != nil {
		log.Fatalf("Failed to save JSON report: %v", err)
	}

	// Generate HTML report from JSON.
	if err := generateHTMLReport(log, report); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	}

	// Print summary to console.
	printReportSummary(ctx, log, report)
}

// saveJSONReport marshals and saves the report as JSON.
func saveJSONReport(report PeerScoreReport) error {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Always use timestamped filenames with fixed base name
	filename := GenerateTimestampedFilename(report.ValidationMode, "peer-score-report.json", report.Timestamp)

	//nolint:gosec // Controlled input.
	if err := os.WriteFile(filename, reportJSON, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	fmt.Printf("JSON report saved: %s\n", filename)

	return nil
}

// generateHTMLReport creates an HTML version of the report.
func generateHTMLReport(log logrus.FieldLogger, report PeerScoreReport) error {
	// Always use timestamped filenames with fixed base name
	htmlFile := GenerateTimestampedFilename(report.ValidationMode, "peer-score-report.html", report.Timestamp)

	// Get API key for AI analysis (optional)
	apiKey := *claudeAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	// Generate HTML report with optional AI analysis, passing report directly
	if apiKey != "" && !*skipAI {
		fmt.Printf("API key found - generating HTML with AI analysis\n")

		return GenerateHTMLReportFromReport(log, report, htmlFile, apiKey, "")
	} else {
		fmt.Printf("No API key or AI disabled - generating HTML without AI analysis\n")
	}

	return GenerateHTMLReportFromReport(log, report, htmlFile, "", "")
}

// printReportSummary displays a comprehensive summary of the test results.
func printReportSummary(_ context.Context, log logrus.FieldLogger, report PeerScoreReport) {
	log.Infof("Peer score test results:")
	log.Infof("Validation Mode: %s", report.ValidationMode)
	log.Infof("Test Duration: %v", report.Duration)
	log.Infof("Total Connections: %d", report.TotalConnections)
	log.Infof("Successful Handshakes: %d", report.SuccessfulHandshakes)
	log.Infof("Failed Handshakes: %d", report.FailedHandshakes)

	// Always use timestamped filenames with fixed base name
	filename := GenerateTimestampedFilename(report.ValidationMode, "peer-score-report.json", report.Timestamp)
	log.Infof("Report saved to: %s", filename)
}
