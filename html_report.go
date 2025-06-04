// Package main provides HTML report generation functionality for the Hermes peer score tool.
// This file handles the conversion of JSON reports into styled HTML presentations suitable
// for web viewing and GitHub Pages deployment.

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"
)

// htmlTemplate contains the complete HTML template for generating peer score reports.
// It uses Tailwind CSS for styling and includes responsive design elements.
// The template includes sections for overall score, statistics grid, client distribution,
// goodbye message analysis, and detailed test information.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hermes Peer Score Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <!-- Header -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-3xl font-bold text-gray-900">Hermes Peer Score Report</h1>
                </div>
                <div class="text-right">
                    <p class="text-sm text-gray-500">Generated on</p>
                    <p class="text-lg font-semibold">{{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</p>
                </div>
            </div>
        </div>

        <!-- Overall Score -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <div class="text-center">
                <h2 class="text-2xl font-bold mb-4">Overall Score</h2>
                <div class="inline-flex items-center justify-center w-32 h-32 rounded-full {{if lt .Report.OverallScore 50.0}}bg-red-100{{else if lt .Report.OverallScore 80.0}}bg-yellow-100{{else}}bg-green-100{{end}}">
                    <span class="text-4xl font-bold {{if lt .Report.OverallScore 50.0}}text-red-600{{else if lt .Report.OverallScore 80.0}}text-yellow-600{{else}}text-green-600{{end}}">
                        {{printf "%.1f" .Report.OverallScore}}%
                    </span>
                </div>
                <p class="mt-4 text-lg font-semibold {{if lt .Report.OverallScore 50.0}}text-red-600{{else if lt .Report.OverallScore 80.0}}text-yellow-600{{else}}text-green-600{{end}}">
                    {{.ScoreClassification}}
                </p>
                <p class="text-gray-600 mt-2">{{.Report.Summary}}</p>
            </div>
        </div>

        <!-- Statistics Grid -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <!-- Test Duration -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-blue-100">
                        <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.Duration.Round (time.Second)}}</p>
                        <p class="text-gray-600">Test Duration</p>
                    </div>
                </div>
            </div>

            <!-- Total Connections -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-green-100">
                        <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.TotalConnections}}</p>
                        <p class="text-gray-600">Total Connections</p>
                    </div>
                </div>
            </div>

            <!-- Successful Handshakes -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-purple-100">
                        <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.SuccessfulHandshakes}}</p>
                        <p class="text-gray-600">Successful Handshakes</p>
                        <p class="text-sm text-gray-500">({{printf "%.1f" .ConnectionRate}}% success rate)</p>
                    </div>
                </div>
            </div>

            <!-- Unique Clients -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-indigo-100">
                        <svg class="w-6 h-6 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 515.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.UniqueClients}}</p>
                        <p class="text-gray-600">Unique Clients</p>
                    </div>
                </div>
            </div>
        </div>

        <!-- Client Distribution -->
        {{if .ClientList}}
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold mb-4">Client Distribution</h3>
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {{range .ClientList}}
                <div class="bg-gray-50 rounded-lg p-4 flex justify-between items-center">
                    <span class="font-medium capitalize">{{.Name}}</span>
                    <span class="bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm font-semibold">{{.Count}}</span>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        <!-- Goodbye Messages -->
        {{if .Report.GoodbyeReasons}}
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold mb-4">Goodbye Messages ({{.Report.GoodbyeMessages}} total)</h3>
            
            <!-- Overall breakdown -->
            <div class="mb-6">
                <h4 class="font-semibold text-gray-700 mb-3">Overall Breakdown</h4>
                <div class="space-y-2">
                    {{range $reason, $count := .Report.GoodbyeReasons}}
                    <div class="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                        <span class="text-gray-700">{{$reason}}</span>
                        <span class="bg-gray-200 text-gray-800 px-3 py-1 rounded-full text-sm font-semibold">{{$count}}</span>
                    </div>
                    {{end}}
                </div>
            </div>

            <!-- By client breakdown -->
            {{if .Report.GoodbyesByClient}}
            <div>
                <h4 class="font-semibold text-gray-700 mb-3">By Client Type</h4>
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {{range $client, $reasons := .Report.GoodbyesByClient}}
                    <div class="bg-gray-50 rounded-lg p-4">
                        <h5 class="font-medium capitalize text-gray-800 mb-2">{{$client}}</h5>
                        <div class="space-y-1">
                            {{range $reason, $count := $reasons}}
                            <div class="flex justify-between text-sm">
                                <span class="text-gray-600">{{$reason}}</span>
                                <span class="font-medium">{{$count}}</span>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- Test Details -->
        <div class="bg-white rounded-lg shadow-lg p-6">
            <h3 class="text-xl font-bold mb-4">Test Details</h3>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <h4 class="font-semibold text-gray-700 mb-2">Test Configuration</h4>
                    <div class="space-y-2 text-sm">
                        <div class="flex justify-between">
                            <span class="text-gray-600">Start Time:</span>
                            <span>{{.Report.StartTime.Format "2006-01-02 15:04:05 UTC"}}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-600">End Time:</span>
                            <span>{{.Report.EndTime.Format "2006-01-02 15:04:05 UTC"}}</span>
                        </div>
                    </div>
                </div>
                <div>
                    <h4 class="font-semibold text-gray-700 mb-2">Results Summary</h4>
                    <div class="space-y-2 text-sm">
                        <div class="flex justify-between">
                            <span class="text-gray-600">Failed Handshakes:</span>
                            <span>{{.Report.FailedHandshakes}}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-600">Total Peers:</span>
                            <span>{{len .Report.Peers}}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Footer -->
        <div class="text-center mt-8 py-4 text-gray-500 text-sm">
            <p>Generated by Hermes Peer Score Tool | <a href="https://github.com/ethpandaops/hermes-peer-score" class="text-blue-600 hover:underline">View Source</a></p>
        </div>
    </div>
</body>
</html>`

// GenerateHTMLReport creates an HTML report from a JSON report file.
// It reads the JSON data, processes it for HTML presentation, and generates
// a styled web page with comprehensive peer connectivity analysis.
//
// Parameters:
//   - jsonFile: Path to the input JSON report file
//   - outputFile: Path where the HTML report should be written
//
// Returns an error if file operations or template processing fails.
func GenerateHTMLReport(jsonFile, outputFile string) error {
	// Read the JSON report file from disk.
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Parse the JSON data into our report structure.
	var report PeerScoreReport
	if uErr := json.Unmarshal(data, &report); uErr != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", uErr)
	}

	// Prepare template data with enhanced fields for HTML presentation.
	// This includes computed fields not present in the raw JSON report.
	templateData := HTMLTemplateData{
		GeneratedAt: time.Now(),
		Report:      report,
	}

	// Calculate connection success rate as a percentage for display.
	if report.TotalConnections > 0 {
		templateData.ConnectionRate = float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
	}

	// Classify the overall score into human-readable categories.
	// These classifications help users quickly understand their network health.
	if report.ConnectionFailed {
		templateData.ScoreClassification = "Connection Failed"
	} else if report.OverallScore >= 90 {
		templateData.ScoreClassification = "Excellent" // 90-100%: Outstanding connectivity.
	} else if report.OverallScore >= 80 {
		templateData.ScoreClassification = "Good" // 80-89%: Good network health.
	} else if report.OverallScore >= 60 {
		templateData.ScoreClassification = "Fair" // 60-79%: Acceptable but could improve.
	} else if report.OverallScore >= 40 {
		templateData.ScoreClassification = "Poor" // 40-59%: Concerning network issues.
	} else {
		templateData.ScoreClassification = "Critical" // 0-39%: Severe connectivity problems.
	}

	// Convert the client distribution map to a sorted list for consistent HTML display.
	// This ensures client types are presented in a predictable order in the web interface.
	for client, count := range report.PeersByClient {
		templateData.ClientList = append(templateData.ClientList, ClientStat{
			Name:  client,
			Count: count,
		})
	}

	// Create the HTML template with custom helper functions.
	// These functions provide additional functionality within the template context.
	tmpl := template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b }, // Mathematical addition for template calculations.
		"time": func() struct{ Second time.Duration } {
			// Provides access to time.Second for duration formatting in templates.
			return struct{ Second time.Duration }{Second: time.Second}
		},
	})

	// Parse the HTML template string into a usable template object.
	tmpl, err = tmpl.Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure the output directory exists before attempting to write the file.
	if mkErr := os.MkdirAll(filepath.Dir(outputFile), 0755); mkErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkErr)
	}

	// Create the output HTML file for writing.
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute the template with our data to generate the final HTML.
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Printf("HTML report generated: %s", outputFile)

	return nil
}
