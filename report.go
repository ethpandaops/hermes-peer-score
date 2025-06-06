package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	if err := generateHTMLReport(); err != nil {
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
func printReportSummary(_ context.Context, log logrus.FieldLogger, report PeerScoreReport) {
	log.Infof("Peer score test results:")
	log.Infof("Test Duration: %v\n", report.Duration)
	log.Infof("Total Connections: %d\n", report.TotalConnections)
	log.Infof("Successful Handshakes: %d\n", report.SuccessfulHandshakes)
	log.Infof("Failed Handshakes: %d\n", report.FailedHandshakes)
	log.Infof("Report saved to: %s\n", *outputFile)
}
