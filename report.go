package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// generateReports creates both JSON and HTML reports from the test results.
func generateReports(ctx context.Context, log logrus.FieldLogger, tool *PeerScoreTool) {
	// Generate the final peer score report.
	report := tool.GenerateReport()

	// Choose report format based on flags
	if *legacyFormat {
		// Generate legacy monolithic format
		generateLegacyReports(ctx, log, report)
	} else {
		// Generate optimized split format
		generateSplitReports(ctx, log, report)
	}

	// Print summary to console.
	printReportSummary(ctx, log, report)
}

// generateLegacyReports creates the old monolithic JSON and HTML files.
func generateLegacyReports(ctx context.Context, log logrus.FieldLogger, report PeerScoreReport) {
	// Save JSON report to file.
	if err := saveJSONReport(report); err != nil {
		log.Fatalf("Failed to save JSON report: %v", err)
	}

	// Generate HTML report from JSON.
	if err := generateHTMLReport(); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	}

	log.Infof("Legacy monolithic reports generated")
}

// generateSplitReports creates the optimized split-file format for better performance.
func generateSplitReports(ctx context.Context, log logrus.FieldLogger, report PeerScoreReport) {
	// Create report directory structure
	reportDir := strings.TrimSuffix(*outputFile, ".json")
	if reportDir == *outputFile {
		reportDir = "peer-score-report"
	}
	
	// Create directory structure
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		log.Fatalf("Failed to create report directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(reportDir, "peers"), 0755); err != nil {
		log.Fatalf("Failed to create peers directory: %v", err)
	}

	// Generate summary report
	summary := createSummaryReport(report, reportDir)
	summaryFile := filepath.Join(reportDir, "summary.json")
	if err := saveJSONToFile(summary, summaryFile); err != nil {
		log.Fatalf("Failed to save summary report: %v", err)
	}

	// Generate peer index
	peerIndex := createPeerIndex(report)
	indexFile := filepath.Join(reportDir, "peer-index.json")
	if err := saveJSONToFile(peerIndex, indexFile); err != nil {
		log.Fatalf("Failed to save peer index: %v", err)
	}

	// Generate individual peer files
	log.Infof("Generating individual peer data files...")
	for peerID, peer := range report.Peers {
		peerData := createPeerDetailedData(peerID, peer, report.PeerEventCounts[peerID])
		peerFile := filepath.Join(reportDir, "peers", fmt.Sprintf("%s.json", sanitizeFilename(peerID)))
		if err := saveJSONToFile(peerData, peerFile); err != nil {
			log.Printf("Failed to save peer data for %s: %v", peerID, err)
		}
	}

	// Generate optimized HTML report
	if err := generateSplitHTMLReport(reportDir); err != nil {
		log.Printf("Failed to generate split HTML report: %v", err)
	}

	log.Infof("Split reports generated in directory: %s", reportDir)
	log.Infof("Generated %d individual peer files", len(report.Peers))
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
	if *legacyFormat {
		log.Infof("Report saved to: %s\n", *outputFile)
	} else {
		reportDir := strings.TrimSuffix(*outputFile, ".json")
		if reportDir == *outputFile {
			reportDir = "peer-score-report"
		}
		log.Infof("Split reports saved to directory: %s\n", reportDir)
	}
}

// createSummaryReport creates a summary report from the full report.
func createSummaryReport(report PeerScoreReport, reportDir string) PeerScoreReportSummary {
	return PeerScoreReportSummary{
		Config:               report.Config,
		Timestamp:            report.Timestamp,
		StartTime:            report.StartTime,
		EndTime:              report.EndTime,
		Duration:             report.Duration,
		TotalConnections:     report.TotalConnections,
		SuccessfulHandshakes: report.SuccessfulHandshakes,
		FailedHandshakes:     report.FailedHandshakes,
		PeerCount:            len(report.Peers),
		ReportDirectory:      reportDir,
	}
}

// createPeerIndex creates an index of all peers with basic information.
func createPeerIndex(report PeerScoreReport) PeerIndex {
	peers := make([]PeerIndexEntry, 0, len(report.Peers))
	
	for peerID, peer := range report.Peers {
		totalEventCount := 0
		if events, exists := report.PeerEventCounts[peerID]; exists {
			for _, count := range events {
				totalEventCount += count
			}
		}

		entry := PeerIndexEntry{
			PeerID:            peerID,
			ClientType:        peer.ClientType,
			ClientAgent:       peer.ClientAgent,
			TotalConnections:  peer.TotalConnections,
			TotalMessageCount: peer.TotalMessageCount,
			FirstSeenAt:       peer.FirstSeenAt,
			LastSeenAt:        peer.LastSeenAt,
			HasDetailedData:   true,
			TotalEventCount:   totalEventCount,
		}
		peers = append(peers, entry)
	}

	// Sort peers by total event count (descending) for better UX
	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			if peers[i].TotalEventCount < peers[j].TotalEventCount {
				peers[i], peers[j] = peers[j], peers[i]
			}
		}
	}

	return PeerIndex{
		GeneratedAt: time.Now(),
		Peers:       peers,
	}
}

// createPeerDetailedData creates detailed data for a specific peer.
func createPeerDetailedData(peerID string, peer *PeerStats, eventCounts map[string]int) PeerDetailedData {
	return PeerDetailedData{
		PeerID:             peerID,
		ClientType:         peer.ClientType,
		ClientAgent:        peer.ClientAgent,
		ConnectionSessions: peer.ConnectionSessions,
		TotalConnections:   peer.TotalConnections,
		TotalMessageCount:  peer.TotalMessageCount,
		FirstSeenAt:        peer.FirstSeenAt,
		LastSeenAt:         peer.LastSeenAt,
		EventCounts:        eventCounts,
	}
}

// saveJSONToFile marshals any data structure to JSON and saves it to a file.
func saveJSONToFile(data interface{}, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// sanitizeFilename removes characters that could cause issues in filenames.
func sanitizeFilename(filename string) string {
	// Replace problematic characters with underscores
	replacements := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	for _, char := range replacements {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// generateSplitHTMLReport creates an optimized HTML report that loads data progressively.
func generateSplitHTMLReport(reportDir string) error {
	htmlFile := filepath.Join(reportDir, "index.html")
	
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hermes Peer Score Report - Optimized</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .loading { display: none; }
        .loading.show { display: block; }
        .peer-detail { display: none; }
        .peer-detail.show { display: block; }
        .score-positive { color: #10b981; }
        .score-negative { color: #ef4444; }
        .score-neutral { color: #6b7280; }
        .accordion-content {
            max-height: 0;
            overflow: hidden;
            transition: max-height 0.3s ease-out;
        }
        .accordion-content.active {
            max-height: 2000px;
            transition: max-height 0.3s ease-in;
        }
    </style>
</head>
<body class="bg-gray-50 min-h-screen">
    <div class="container mx-auto px-4 py-8 max-w-7xl">
        <!-- Header -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-6">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-3xl font-bold text-gray-900">Hermes Peer Score Report (Optimized)</h1>
                    <p class="text-gray-600 mt-2" id="generated-at">Loading...</p>
                </div>
                <div class="text-right" id="test-duration">
                    <div class="text-sm text-gray-500">Test Duration</div>
                    <div class="text-2xl font-semibold text-blue-600" id="duration-value">Loading...</div>
                </div>
            </div>
        </div>

        <!-- Summary Statistics -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6" id="summary-stats">
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Total Connections</div>
                <div class="text-2xl font-bold text-gray-900" id="total-connections">Loading...</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Successful Handshakes</div>
                <div class="text-2xl font-bold text-green-600" id="successful-handshakes">Loading...</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Failed Handshakes</div>
                <div class="text-2xl font-bold text-red-600" id="failed-handshakes">Loading...</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Unique Peers</div>
                <div class="text-2xl font-bold text-blue-600" id="unique-peers">Loading...</div>
            </div>
        </div>

        <!-- Test Configuration -->
        <div class="bg-white rounded-lg shadow-lg mb-6">
            <div class="p-6 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">Test Configuration</h2>
            </div>
            <div class="p-6" id="test-config">
                <div class="text-center text-gray-500">Loading configuration...</div>
            </div>
        </div>

        <!-- Peer List Section -->
        <div class="bg-white rounded-lg shadow-lg mb-6">
            <div class="p-6 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">Peer Analysis</h2>
                <p class="text-gray-600 mt-1">Click on any peer to load detailed information</p>
                <div class="mt-3">
                    <input type="text" id="peer-search" placeholder="Search peers..." 
                           class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500">
                </div>
            </div>
            <div class="p-6">
                <div id="peer-list" class="space-y-4">
                    <div class="text-center text-gray-500">Loading peer list...</div>
                </div>
            </div>
        </div>

        <!-- Peer Detail Modal/Section -->
        <div id="peer-detail-section" class="peer-detail bg-white rounded-lg shadow-lg mb-6">
            <div class="p-6 border-b border-gray-200">
                <div class="flex justify-between items-center">
                    <h2 class="text-xl font-semibold text-gray-900">Peer Details</h2>
                    <button onclick="closePeerDetail()" class="text-gray-500 hover:text-gray-700">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>
            </div>
            <div class="p-6" id="peer-detail-content">
                <!-- Peer details will be loaded here -->
            </div>
        </div>

        <!-- Loading indicator -->
        <div id="loading-indicator" class="loading fixed top-4 right-4 bg-blue-500 text-white px-4 py-2 rounded shadow-lg">
            <div class="flex items-center">
                <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Loading...
            </div>
        </div>
    </div>

    <script>
        let summaryData = null;
        let peerIndexData = null;
        let loadedPeerData = new Map();

        // Initialize the report on page load
        document.addEventListener('DOMContentLoaded', function() {
            loadSummaryData();
            loadPeerIndex();
            setupSearch();
        });

        // Show/hide loading indicator
        function showLoading() {
            document.getElementById('loading-indicator').classList.add('show');
        }

        function hideLoading() {
            document.getElementById('loading-indicator').classList.remove('show');
        }

        // Load summary data
        async function loadSummaryData() {
            try {
                showLoading();
                const response = await fetch('summary.json');
                summaryData = await response.json();
                populateSummary();
            } catch (error) {
                console.error('Failed to load summary data:', error);
            } finally {
                hideLoading();
            }
        }

        // Load peer index
        async function loadPeerIndex() {
            try {
                showLoading();
                const response = await fetch('peer-index.json');
                peerIndexData = await response.json();
                populatePeerList();
            } catch (error) {
                console.error('Failed to load peer index:', error);
            } finally {
                hideLoading();
            }
        }

        // Populate summary section
        function populateSummary() {
            if (!summaryData) return;

            document.getElementById('generated-at').textContent = 
                'Generated on ' + new Date(summaryData.timestamp).toLocaleString();
            document.getElementById('duration-value').textContent = 
                (summaryData.duration / 1e9).toFixed(1) + 's';
            document.getElementById('total-connections').textContent = summaryData.total_connections;
            document.getElementById('successful-handshakes').textContent = summaryData.successful_handshakes;
            document.getElementById('failed-handshakes').textContent = summaryData.failed_handshakes;
            document.getElementById('unique-peers').textContent = summaryData.peer_count;

            // Populate test configuration
            const configHtml = ` + "`" + `
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                        <div class="text-sm font-medium text-gray-500">Test Duration</div>
                        <div class="text-lg">${(summaryData.config.test_duration / 1e9).toFixed(1)} seconds</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-gray-500">Start Time</div>
                        <div class="text-lg">${new Date(summaryData.start_time).toLocaleString()}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-gray-500">End Time</div>
                        <div class="text-lg">${new Date(summaryData.end_time).toLocaleString()}</div>
                    </div>
                </div>
            ` + "`" + `;
            document.getElementById('test-config').innerHTML = configHtml;
        }

        // Populate peer list
        function populatePeerList() {
            if (!peerIndexData) return;

            const peerListHtml = peerIndexData.peers.map(peer => ` + "`" + `
                <div class="border border-gray-200 rounded-lg peer-item" data-peer-id="${peer.peer_id}">
                    <div class="p-4 hover:bg-gray-50 cursor-pointer" onclick="loadPeerDetails('${peer.peer_id}')">
                        <div class="flex items-center justify-between">
                            <div class="flex items-center space-x-4">
                                <h4 class="font-medium text-gray-900">${peer.peer_id.substring(0, 12)}...</h4>
                                <span class="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">${peer.client_type}</span>
                                <span class="text-sm text-gray-600">${peer.total_connections} sessions</span>
                                <span class="text-sm text-gray-600">${peer.total_event_count} events</span>
                                <span class="text-sm text-gray-600">${peer.total_message_count} messages</span>
                            </div>
                            <div class="text-right">
                                <div class="text-xs text-gray-500">
                                    ${peer.first_seen_at ? 'First: ' + new Date(peer.first_seen_at).toLocaleTimeString() : ''}
                                </div>
                                <div class="text-xs text-gray-500">
                                    ${peer.last_seen_at ? 'Last: ' + new Date(peer.last_seen_at).toLocaleTimeString() : ''}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            ` + "`" + `).join('');

            document.getElementById('peer-list').innerHTML = peerListHtml;
        }

        // Load detailed peer data
        async function loadPeerDetails(peerId) {
            try {
                showLoading();
                
                // Check if already loaded
                if (loadedPeerData.has(peerId)) {
                    displayPeerDetails(loadedPeerData.get(peerId));
                    return;
                }

                // Load from file
                const response = await fetch(` + "`" + `peers/${peerId}.json` + "`" + `);
                const peerData = await response.json();
                
                // Cache the data
                loadedPeerData.set(peerId, peerData);
                
                // Display the data
                displayPeerDetails(peerData);
            } catch (error) {
                console.error('Failed to load peer details:', error);
                alert('Failed to load peer details for ' + peerId);
            } finally {
                hideLoading();
            }
        }

        // Display peer details
        function displayPeerDetails(peerData) {
            const detailHtml = ` + "`" + `
                <div class="mb-6">
                    <h3 class="text-lg font-semibold text-gray-900 mb-4">Basic Information</h3>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <div class="text-sm font-medium text-gray-500">Peer ID</div>
                            <div class="text-sm font-mono break-all">${peerData.peer_id}</div>
                        </div>
                        <div>
                            <div class="text-sm font-medium text-gray-500">Client Agent</div>
                            <div class="text-sm">${peerData.client_agent}</div>
                        </div>
                        <div>
                            <div class="text-sm font-medium text-gray-500">Total Connections</div>
                            <div class="text-sm">${peerData.total_connections}</div>
                        </div>
                        <div>
                            <div class="text-sm font-medium text-gray-500">Total Messages</div>
                            <div class="text-sm">${peerData.total_message_count}</div>
                        </div>
                    </div>
                </div>

                ${peerData.event_counts && Object.keys(peerData.event_counts).length > 0 ? ` + "`" + `
                <div class="mb-6">
                    <h3 class="text-lg font-semibold text-gray-900 mb-4">Event Counts</h3>
                    <div class="bg-blue-50 rounded-lg p-4">
                        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                            ${Object.entries(peerData.event_counts).map(([event, count]) => ` + "`" + `
                                <div class="bg-white rounded p-3 text-center">
                                    <div class="text-lg font-semibold text-blue-600">${count}</div>
                                    <div class="text-xs text-gray-600">${event}</div>
                                </div>
                            ` + "`" + `).join('')}
                        </div>
                    </div>
                </div>
                ` + "`" + ` : ''}

                <div class="mb-6">
                    <h3 class="text-lg font-semibold text-gray-900 mb-4">Connection Sessions (${peerData.connection_sessions.length})</h3>
                    <div class="space-y-4">
                        ${peerData.connection_sessions.map((session, index) => ` + "`" + `
                            <div class="border border-gray-200 rounded-lg">
                                <div class="p-3 bg-gray-50 cursor-pointer" onclick="toggleSessionDetails('session-${index}')">
                                    <div class="flex items-center justify-between">
                                        <div class="flex items-center space-x-4">
                                            <span class="font-medium text-gray-900">Session ${index + 1}</span>
                                            <span class="text-sm text-gray-600">${(session.connection_duration / 1e9).toFixed(2)}s duration</span>
                                            <span class="text-sm text-gray-600">${session.message_count} messages</span>
                                            <span class="px-2 py-1 text-xs ${session.disconnected ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'} rounded">
                                                ${session.disconnected ? 'Disconnected' : 'Connected'}
                                            </span>
                                        </div>
                                        <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="session-${index}-arrow">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                        </svg>
                                    </div>
                                </div>
                                <div class="accordion-content" id="session-${index}">
                                    <div class="p-4 border-t border-gray-200">
                                        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                                            ${session.connected_at ? ` + "`" + `
                                            <div class="text-center p-3 bg-green-50 rounded">
                                                <div class="font-medium text-green-800">Connected</div>
                                                <div class="text-sm text-green-600">${new Date(session.connected_at).toLocaleTimeString()}</div>
                                            </div>
                                            ` + "`" + ` : ''}
                                            ${session.identified_at ? ` + "`" + `
                                            <div class="text-center p-3 bg-blue-50 rounded">
                                                <div class="font-medium text-blue-800">Identified</div>
                                                <div class="text-sm text-blue-600">${new Date(session.identified_at).toLocaleTimeString()}</div>
                                            </div>
                                            ` + "`" + ` : ''}
                                            ${session.disconnected_at ? ` + "`" + `
                                            <div class="text-center p-3 bg-red-50 rounded">
                                                <div class="font-medium text-red-800">Disconnected</div>
                                                <div class="text-sm text-red-600">${new Date(session.disconnected_at).toLocaleTimeString()}</div>
                                            </div>
                                            ` + "`" + ` : ''}
                                        </div>
                                        ${session.peer_scores && session.peer_scores.length > 0 ? ` + "`" + `
                                        <div class="mt-4">
                                            <h6 class="font-medium text-gray-800 mb-3">Peer Score Evolution (${session.peer_scores.length} snapshots)</h6>
                                            <div class="overflow-x-auto">
                                                <table class="min-w-full bg-white border border-gray-200 rounded text-xs">
                                                    <thead class="bg-gray-50">
                                                        <tr>
                                                            <th class="px-2 py-1 text-left font-medium text-gray-500">Time</th>
                                                            <th class="px-2 py-1 text-left font-medium text-gray-500">Total Score</th>
                                                            <th class="px-2 py-1 text-left font-medium text-gray-500">App Score</th>
                                                            <th class="px-2 py-1 text-left font-medium text-gray-500">IP Colocation</th>
                                                            <th class="px-2 py-1 text-left font-medium text-gray-500">Behaviour</th>
                                                            <th class="px-2 py-1 text-left font-medium text-gray-500">Topics</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody class="divide-y divide-gray-100">
                                                        ${session.peer_scores.map(snapshot => ` + "`" + `
                                                        <tr class="hover:bg-gray-50">
                                                            <td class="px-2 py-1">${new Date(snapshot.timestamp).toLocaleTimeString()}</td>
                                                            <td class="px-2 py-1">
                                                                <span class="font-medium ${snapshot.score > 0 ? 'score-positive' : snapshot.score < 0 ? 'score-negative' : 'score-neutral'}">
                                                                    ${snapshot.score.toFixed(3)}
                                                                </span>
                                                            </td>
                                                            <td class="px-2 py-1">
                                                                <span class="${snapshot.app_specific_score > 0 ? 'score-positive' : snapshot.app_specific_score < 0 ? 'score-negative' : 'score-neutral'}">
                                                                    ${snapshot.app_specific_score.toFixed(3)}
                                                                </span>
                                                            </td>
                                                            <td class="px-2 py-1">
                                                                <span class="${snapshot.ip_colocation_factor > 0 ? 'score-positive' : snapshot.ip_colocation_factor < 0 ? 'score-negative' : 'score-neutral'}">
                                                                    ${snapshot.ip_colocation_factor.toFixed(3)}
                                                                </span>
                                                            </td>
                                                            <td class="px-2 py-1">
                                                                <span class="${snapshot.behaviour_penalty > 0 ? 'score-positive' : snapshot.behaviour_penalty < 0 ? 'score-negative' : 'score-neutral'}">
                                                                    ${snapshot.behaviour_penalty.toFixed(3)}
                                                                </span>
                                                            </td>
                                                            <td class="px-2 py-1">
                                                                ${snapshot.topics ? snapshot.topics.length + ' topics' : 'None'}
                                                            </td>
                                                        </tr>
                                                        ` + "`" + `).join('')}
                                                    </tbody>
                                                </table>
                                            </div>
                                        </div>
                                        ` + "`" + ` : ''}
                                    </div>
                                </div>
                            </div>
                        ` + "`" + `).join('')}
                    </div>
                </div>
            ` + "`" + `;

            document.getElementById('peer-detail-content').innerHTML = detailHtml;
            document.getElementById('peer-detail-section').classList.add('show');
            document.getElementById('peer-detail-section').scrollIntoView({ behavior: 'smooth' });
        }

        // Close peer detail section
        function closePeerDetail() {
            document.getElementById('peer-detail-section').classList.remove('show');
        }

        // Toggle session details
        function toggleSessionDetails(sessionId) {
            const content = document.getElementById(sessionId);
            const arrow = document.getElementById(sessionId + '-arrow');
            
            if (content.classList.contains('active')) {
                content.classList.remove('active');
                if (arrow) arrow.style.transform = 'rotate(0deg)';
            } else {
                content.classList.add('active');
                if (arrow) arrow.style.transform = 'rotate(180deg)';
            }
        }

        // Setup search functionality
        function setupSearch() {
            const searchInput = document.getElementById('peer-search');
            searchInput.addEventListener('input', function() {
                const searchTerm = this.value.toLowerCase();
                const peerItems = document.querySelectorAll('.peer-item');
                
                peerItems.forEach(item => {
                    const peerId = item.dataset.peerId.toLowerCase();
                    const text = item.textContent.toLowerCase();
                    if (peerId.includes(searchTerm) || text.includes(searchTerm)) {
                        item.style.display = 'block';
                    } else {
                        item.style.display = 'none';
                    }
                });
            });
        }
    </script>
</body>
</html>`
	
	return os.WriteFile(htmlFile, []byte(htmlContent), 0644)
}
