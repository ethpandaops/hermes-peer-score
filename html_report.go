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
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hermes Peer Score Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .accordion-content {
            max-height: 0;
            overflow: hidden;
            transition: max-height 0.3s ease-out;
        }
        .accordion-content.active {
            max-height: 2000px;
            transition: max-height 0.3s ease-in;
        }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .score-positive { color: #10b981; }
        .score-negative { color: #ef4444; }
        .score-neutral { color: #6b7280; }
    </style>
</head>
<body class="bg-gray-50 min-h-screen">
    <div class="container mx-auto px-4 py-8 max-w-7xl">
        <!-- Header -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-6">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-3xl font-bold text-gray-900">Hermes Peer Score Report</h1>
                    <p class="text-gray-600 mt-2">Generated on {{.GeneratedAt.Format "January 2, 2006 at 3:04 PM"}}</p>
                </div>
                <div class="text-right">
                    <div class="text-sm text-gray-500">Test Duration</div>
                    <div class="text-2xl font-semibold text-blue-600">{{printf "%.1f" .Report.Duration.Seconds}}s</div>
                </div>
            </div>
        </div>

        <!-- Summary Statistics -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Total Connections</div>
                <div class="text-2xl font-bold text-gray-900">{{.Report.TotalConnections}}</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Successful Handshakes</div>
                <div class="text-2xl font-bold text-green-600">{{.Report.SuccessfulHandshakes}}</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Failed Handshakes</div>
                <div class="text-2xl font-bold text-red-600">{{.Report.FailedHandshakes}}</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Unique Peers</div>
                <div class="text-2xl font-bold text-blue-600">{{len .Report.Peers}}</div>
            </div>
        </div>

        <!-- Test Configuration -->
        <div class="bg-white rounded-lg shadow-lg mb-6">
            <div class="p-6 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">Test Configuration</h2>
            </div>
            <div class="p-6">
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                        <div class="text-sm font-medium text-gray-500">Test Duration</div>
                        <div class="text-lg">{{printf "%.1f" .Report.Config.TestDuration.Seconds}} seconds</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-gray-500">Start Time</div>
                        <div class="text-lg">{{.Report.StartTime.Format "2006-01-02 15:04:05"}}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-gray-500">End Time</div>
                        <div class="text-lg">{{.Report.EndTime.Format "2006-01-02 15:04:05"}}</div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Peer Analysis Section -->
        <div class="bg-white rounded-lg shadow-lg mb-6">
            <div class="p-6 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">Detailed Peer Analysis</h2>
                <p class="text-gray-600 mt-1">Complete lifecycle, events, and scoring data for each discovered peer</p>
            </div>
            <div class="p-6">
                <div class="space-y-6">
                    {{range $peerID := sortPeersByEvents .Report.Peers .Report.PeerEventCounts}}
                    {{$peer := index $.Report.Peers $peerID}}
                    <div class="border border-gray-200 rounded-lg">
                        <!-- Peer Header -->
                        <div class="p-4 bg-gray-50 cursor-pointer" onclick="toggleAccordion('peer-{{$peerID}}')">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center space-x-4">
                                    <h4 class="font-medium text-gray-900">{{slice 0 12 $peerID}}...</h4>
                                    <span class="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">{{$peer.ClientType}}</span>
                                    <span class="text-sm text-gray-600">{{$peer.TotalConnections}} sessions</span>
                                    {{if index $.Report.PeerEventCounts $peerID}}
                                        {{$eventCount := 0}}
                                        {{range $eventType, $count := index $.Report.PeerEventCounts $peerID}}
                                            {{$eventCount = add $eventCount $count}}
                                        {{end}}
                                        <span class="text-sm text-gray-600">{{$eventCount}} events</span>
                                    {{end}}
                                </div>
                                <svg class="w-5 h-5 text-gray-500 transform transition-transform" id="peer-{{$peerID}}-arrow">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                </svg>
                            </div>
                        </div>
                        
                        <!-- Peer Details -->
                        <div class="accordion-content" id="peer-{{$peerID}}">
                            <div class="p-6 border-t border-gray-200">
                                <!-- Basic Info -->
                                <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                                    <div>
                                        <div class="text-sm font-medium text-gray-500">Full Peer ID</div>
                                        <div class="text-sm font-mono break-all">{{$peerID}}</div>
                                    </div>
                                    <div>
                                        <div class="text-sm font-medium text-gray-500">Client Agent</div>
                                        <div class="text-sm">{{$peer.ClientAgent}}</div>
                                    </div>
                                    {{if $peer.FirstSeenAt}}
                                    <div>
                                        <div class="text-sm font-medium text-gray-500">First Seen</div>
                                        <div class="text-sm">{{$peer.FirstSeenAt.Format "15:04:05.000"}}</div>
                                    </div>
                                    {{end}}
                                    {{if $peer.LastSeenAt}}
                                    <div>
                                        <div class="text-sm font-medium text-gray-500">Last Seen</div>
                                        <div class="text-sm">{{$peer.LastSeenAt.Format "15:04:05.000"}}</div>
                                    </div>
                                    {{end}}
                                </div>

                                <!-- Peer Events -->
                                {{if index $.Report.PeerEventCounts $peerID}}
                                <div class="mb-6">
                                    <h5 class="font-medium text-gray-900 mb-3 flex items-center">
                                        <span>Peer Events</span>
                                        <span class="ml-2 px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded">{{len (index $.Report.PeerEventCounts $peerID)}} types</span>
                                    </h5>
                                    <div class="bg-blue-50 rounded-lg p-4">
                                        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                                            {{range $eventType, $count := index $.Report.PeerEventCounts $peerID}}
                                            <div class="bg-white rounded p-3 text-center">
                                                <div class="text-lg font-semibold text-blue-600">{{$count}}</div>
                                                <div class="text-xs text-gray-600">{{$eventType}}</div>
                                            </div>
                                            {{end}}
                                        </div>
                                    </div>
                                </div>
                                {{end}}

                                <!-- Connection Sessions -->
                                <div class="mb-6">
                                    <h5 class="font-medium text-gray-900 mb-3 flex items-center">
                                        <span>Connection Sessions</span>
                                        <span class="ml-2 px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded">{{len $peer.ConnectionSessions}} sessions</span>
                                    </h5>
                                    <div class="space-y-4">
                                        {{range $sessionIdx, $session := $peer.ConnectionSessions}}
                                        <div class="border border-gray-200 rounded-lg">
                                            <!-- Session Header -->
                                            <div class="p-3 bg-gray-50 cursor-pointer" onclick="toggleAccordion('session-{{$peerID}}-{{$sessionIdx}}')">
                                                <div class="flex items-center justify-between">
                                                    <div class="flex items-center space-x-4">
                                                        <span class="font-medium text-gray-900">Session {{add $sessionIdx 1}}</span>
                                                        <span class="text-sm text-gray-600">{{printf "%.2f" $session.ConnectionDuration.Seconds}}s duration</span>
                                                        <span class="text-sm text-gray-600">{{$session.MessageCount}} messages</span>
                                                        {{if $session.PeerScores}}
                                                            <span class="text-sm text-gray-600">{{len $session.PeerScores}} score snapshots</span>
                                                        {{end}}
                                                        <span class="px-2 py-1 text-xs {{if $session.Disconnected}}bg-red-100 text-red-800{{else}}bg-green-100 text-green-800{{end}} rounded">
                                                            {{if $session.Disconnected}}Disconnected{{else}}Connected{{end}}
                                                        </span>
                                                    </div>
                                                    <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="session-{{$peerID}}-{{$sessionIdx}}-arrow">
                                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                                    </svg>
                                                </div>
                                            </div>
                                            
                                            <!-- Session Details -->
                                            <div class="accordion-content" id="session-{{$peerID}}-{{$sessionIdx}}">
                                                <div class="p-4 border-t border-gray-200">
                                                    <!-- Session Timeline -->
                                                    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
                                                        {{if $session.ConnectedAt}}
                                                        <div class="text-center p-3 bg-green-50 rounded">
                                                            <div class="font-medium text-green-800">Connected</div>
                                                            <div class="text-sm text-green-600">{{$session.ConnectedAt.Format "15:04:05.000"}}</div>
                                                        </div>
                                                        {{end}}
                                                        {{if $session.IdentifiedAt}}
                                                        <div class="text-center p-3 bg-blue-50 rounded">
                                                            <div class="font-medium text-blue-800">Identified</div>
                                                            <div class="text-sm text-blue-600">{{$session.IdentifiedAt.Format "15:04:05.000"}}</div>
                                                        </div>
                                                        {{end}}
                                                        {{if $session.DisconnectedAt}}
                                                        <div class="text-center p-3 bg-red-50 rounded">
                                                            <div class="font-medium text-red-800">Disconnected</div>
                                                            <div class="text-sm text-red-600">{{$session.DisconnectedAt.Format "15:04:05.000"}}</div>
                                                        </div>
                                                        {{end}}
                                                    </div>

                                                    <!-- Peer Scores for this session -->
                                                    {{if $session.PeerScores}}
                                                    <div class="mt-4">
                                                        <h6 class="font-medium text-gray-800 mb-3 flex items-center">
                                                            <span>Peer Score Evolution</span>
                                                            <span class="ml-2 px-2 py-1 text-xs bg-purple-100 text-purple-600 rounded">{{len $session.PeerScores}} snapshots</span>
                                                        </h6>
                                                        <div class="overflow-x-auto">
                                                            <table class="min-w-full bg-white border border-gray-200 rounded">
                                                                <thead class="bg-gray-50">
                                                                    <tr>
                                                                        <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Time</th>
                                                                        <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Total Score</th>
                                                                        <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">App Score</th>
                                                                        <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">IP Colocation</th>
                                                                        <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Behaviour</th>
                                                                        <th class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Topics</th>
                                                                    </tr>
                                                                </thead>
                                                                <tbody class="divide-y divide-gray-100">
                                                                    {{range $scoreIdx, $snapshot := $session.PeerScores}}
                                                                    <tr class="hover:bg-gray-50">
                                                                        <td class="px-3 py-2 text-xs">{{$snapshot.Timestamp.Format "15:04:05.000"}}</td>
                                                                        <td class="px-3 py-2 text-xs">
                                                                            <span class="font-medium {{if gt $snapshot.Score 0.0}}score-positive{{else if lt $snapshot.Score 0.0}}score-negative{{else}}score-neutral{{end}}">
                                                                                {{printf "%.3f" $snapshot.Score}}
                                                                            </span>
                                                                        </td>
                                                                        <td class="px-3 py-2 text-xs">
                                                                            <span class="{{if gt $snapshot.AppSpecificScore 0.0}}score-positive{{else if lt $snapshot.AppSpecificScore 0.0}}score-negative{{else}}score-neutral{{end}}">
                                                                                {{printf "%.3f" $snapshot.AppSpecificScore}}
                                                                            </span>
                                                                        </td>
                                                                        <td class="px-3 py-2 text-xs">
                                                                            <span class="{{if gt $snapshot.IPColocationFactor 0.0}}score-positive{{else if lt $snapshot.IPColocationFactor 0.0}}score-negative{{else}}score-neutral{{end}}">
                                                                                {{printf "%.3f" $snapshot.IPColocationFactor}}
                                                                            </span>
                                                                        </td>
                                                                        <td class="px-3 py-2 text-xs">
                                                                            <span class="{{if gt $snapshot.BehaviourPenalty 0.0}}score-positive{{else if lt $snapshot.BehaviourPenalty 0.0}}score-negative{{else}}score-neutral{{end}}">
                                                                                {{printf "%.3f" $snapshot.BehaviourPenalty}}
                                                                            </span>
                                                                        </td>
                                                                        <td class="px-3 py-2 text-xs">
                                                                            {{if $snapshot.Topics}}
                                                                                <button onclick="toggleTopicDetails('topics-{{$peerID}}-{{$sessionIdx}}-{{$scoreIdx}}')" class="text-blue-600 hover:text-blue-800 underline">
                                                                                    {{len $snapshot.Topics}} topics
                                                                                </button>
                                                                                <div id="topics-{{$peerID}}-{{$sessionIdx}}-{{$scoreIdx}}" class="hidden mt-2">
                                                                                    <div class="bg-gray-50 rounded p-2 max-h-40 overflow-y-auto">
                                                                                        {{range $topic := $snapshot.Topics}}
                                                                                        <div class="mb-2 p-2 bg-white rounded border text-xs">
                                                                                            <div class="font-medium text-gray-800 mb-1">{{$topic.Topic}}</div>
                                                                                            <div class="grid grid-cols-2 gap-1 text-xs">
                                                                                                <div><span class="text-gray-500">Mesh Time:</span> {{printf "%.1f" $topic.TimeInMesh.Seconds}}s</div>
                                                                                                <div><span class="text-gray-500">First Msgs:</span> {{printf "%.1f" $topic.FirstMessageDeliveries}}</div>
                                                                                                <div><span class="text-gray-500">Mesh Msgs:</span> {{printf "%.1f" $topic.MeshMessageDeliveries}}</div>
                                                                                                <div><span class="text-gray-500">Invalid:</span> {{printf "%.1f" $topic.InvalidMessageDeliveries}}</div>
                                                                                            </div>
                                                                                        </div>
                                                                                        {{end}}
                                                                                    </div>
                                                                                </div>
                                                                            {{else}}
                                                                                <span class="text-gray-400">None</span>
                                                                            {{end}}
                                                                        </td>
                                                                    </tr>
                                                                    {{end}}
                                                                </tbody>
                                                            </table>
                                                        </div>
                                                    </div>
                                                    {{end}}
                                                </div>
                                            </div>
                                        </div>
                                        {{end}}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>
    </div>

    <script>
        function toggleAccordion(id) {
            const content = document.getElementById(id);
            const arrow = document.getElementById(id + '-arrow');
            
            if (content.classList.contains('active')) {
                content.classList.remove('active');
                if (arrow) arrow.style.transform = 'rotate(0deg)';
            } else {
                content.classList.add('active');
                if (arrow) arrow.style.transform = 'rotate(180deg)';
            }
        }

        function toggleTopicDetails(id) {
            const element = document.getElementById(id);
            if (element) {
                element.classList.toggle('hidden');
            }
        }

        // Add smooth scrolling to peer sections
        function scrollToPeer(peerId) {
            const element = document.getElementById('peer-' + peerId);
            if (element) {
                element.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        }

        // Initialize page
        document.addEventListener('DOMContentLoaded', function() {
            // Add click handlers for better UX
            console.log('Hermes Peer Score Report loaded successfully');
        });
    </script>
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

	// Create the HTML template with custom helper functions.
	// These functions provide additional functionality within the template context.
	tmpl := template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b }, // Mathematical addition for template calculations.
		"time": func() struct{ Second, Millisecond time.Duration } {
			// Provides access to time units for duration formatting in templates.
			return struct{ Second, Millisecond time.Duration }{Second: time.Second, Millisecond: time.Millisecond}
		},
		"slice": func(start, end int, s string) string {
			// String slicing function for templates
			if start < 0 || start >= len(s) {
				return s
			}
			if end < 0 || end > len(s) {
				end = len(s)
			}
			if start >= end {
				return s
			}
			return s[start:end]
		},
		"sortPeersByEvents": func(peers map[string]*PeerStats, eventCounts map[string]map[string]int) []string {
			// Create a slice of peer IDs and sort by total event count (descending)
			type peerEventCount struct {
				peerID     string
				eventCount int
			}
			
			var peerCounts []peerEventCount
			for peerID := range peers {
				totalEvents := 0
				if events, exists := eventCounts[peerID]; exists {
					for _, count := range events {
						totalEvents += count
					}
				}
				peerCounts = append(peerCounts, peerEventCount{
					peerID:     peerID,
					eventCount: totalEvents,
				})
			}
			
			// Sort by event count (descending)
			for i := 0; i < len(peerCounts); i++ {
				for j := i + 1; j < len(peerCounts); j++ {
					if peerCounts[i].eventCount < peerCounts[j].eventCount {
						peerCounts[i], peerCounts[j] = peerCounts[j], peerCounts[i]
					}
				}
			}
			
			// Extract sorted peer IDs
			var sortedPeerIDs []string
			for _, pc := range peerCounts {
				sortedPeerIDs = append(sortedPeerIDs, pc.peerID)
			}
			
			return sortedPeerIDs
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
