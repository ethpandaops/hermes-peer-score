package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type PeerScoreConfig struct {
	HermesPath     string        `yaml:"hermes_path"`
	TestDuration   time.Duration `yaml:"test_duration"`
	ReportInterval time.Duration `yaml:"report_interval"`
	HermesArgs     []string      `yaml:"hermes_args"`
}

type PeerStats struct {
	PeerID       string    `json:"peer_id"`
	ClientType   string    `json:"client_type"`
	ConnectedAt  time.Time `json:"connected_at"`
	HandshakeOK  bool      `json:"handshake_ok"`
	GoodbyeCount int       `json:"goodbye_count"`
	LastGoodbye  string    `json:"last_goodbye"`
	MessageCount int       `json:"message_count"`
}

// Removed old test struct definitions - now using simplified PeerScoreReport

type PeerScoreReport struct {
	Timestamp            time.Time             `json:"timestamp"`
	Config               PeerScoreConfig       `json:"config"`
	StartTime            time.Time             `json:"start_time"`
	EndTime              time.Time             `json:"end_time"`
	Duration             time.Duration         `json:"duration"`
	TotalConnections     int                   `json:"total_connections"`
	SuccessfulHandshakes int                   `json:"successful_handshakes"`
	FailedHandshakes     int                   `json:"failed_handshakes"`
	GoodbyeMessages      int                   `json:"goodbye_messages"`
	GoodbyeReasons       map[string]int        `json:"goodbye_reasons"`
	PeersByClient        map[string]int        `json:"peers_by_client"`
	UniqueClients        int                   `json:"unique_clients"`
	Peers                map[string]*PeerStats `json:"peers"`
	OverallScore         float64               `json:"overall_score"`
	Summary              string                `json:"summary"`
	Errors               []string              `json:"errors"`
	ConnectionFailed     bool                  `json:"connection_failed"`
}

type PeerScoreTool struct {
	config     PeerScoreConfig
	hermesCmd  *exec.Cmd
	logRegexes map[string]*regexp.Regexp
	mu         sync.RWMutex
	peers      map[string]*PeerStats
	startTime  time.Time

	// Global counters for reporting
	totalGoodbyes  int
	goodbyeReasons map[string]int

	// Error tracking
	errors           []string
	connectionFailed bool
}

func NewPeerScoreTool(config PeerScoreConfig) *PeerScoreTool {
	tool := &PeerScoreTool{
		config:           config,
		peers:            make(map[string]*PeerStats),
		goodbyeReasons:   make(map[string]int),
		errors:           make([]string, 0),
		connectionFailed: false,
		logRegexes: map[string]*regexp.Regexp{
			"connected":   regexp.MustCompile(`Connected with peer.*peer_id=(\w+)`),
			"handshake":   regexp.MustCompile(`Performed successful handshake.*peer_id=(\w+).*agent=([^,\s]+)`),
			"goodbye":     regexp.MustCompile(`Received goodbye message.*peer_id=(\w+).*msg="([^"]+)"`),
			"disconnect":  regexp.MustCompile(`Disconnected from handshaked peer.*peer_id=(\w+)`),
			"attestation": regexp.MustCompile(`beacon_attestation.*from.*(\w+)`),
		},
	}

	return tool
}

func (pst *PeerScoreTool) StartHermes(ctx context.Context) error {
	// Start Hermes directly with the provided arguments
	pst.hermesCmd = exec.CommandContext(ctx, pst.config.HermesPath, pst.config.HermesArgs...)

	// Capture stdout and stderr for log parsing
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

	// Start enhanced log parsing
	parser := NewEnhancedLogParser(pst)
	go parser.StartParsing(ctx, stdout)
	go parser.StartParsing(ctx, stderr)

	pst.startTime = time.Now()
	log.Printf("Started Hermes with PID %d", pst.hermesCmd.Process.Pid)

	return nil
}

func (pst *PeerScoreTool) GenerateReport() PeerScoreReport {
	log.Println("Generating peer score report...")

	endTime := time.Now()
	duration := endTime.Sub(pst.startTime)

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
		PeersByClient:    make(map[string]int),
		Peers:            make(map[string]*PeerStats),
		Errors:           make([]string, len(pst.errors)),
		ConnectionFailed: pst.connectionFailed,
	}

	// Copy goodbye reasons
	for reason, count := range pst.goodbyeReasons {
		report.GoodbyeReasons[reason] = count
	}

	// Copy errors
	copy(report.Errors, pst.errors)

	// Process peer data
	for id, peer := range pst.peers {
		// Copy peer data
		report.Peers[id] = peer

		// Count handshakes
		if peer.HandshakeOK {
			report.SuccessfulHandshakes++
		} else {
			report.FailedHandshakes++
		}

		// Count by client type
		if peer.ClientType != "" {
			report.PeersByClient[peer.ClientType]++
		}
	}

	// Count unique clients
	report.UniqueClients = len(report.PeersByClient)

	// Calculate overall score with goodbye penalty
	if report.ConnectionFailed {
		report.OverallScore = 0.0
	} else if report.TotalConnections > 0 {
		connectionScore := float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
		diversityScore := float64(min(report.UniqueClients, 4)) / 4.0 * 100 // Max score for 4+ clients

		// Calculate goodbye penalty for ERROR-level messages
		errorGoodbyes := 0
		for reason, count := range report.GoodbyeReasons {
			if pst.classifyGoodbyeSeverity(reason) == "ERROR" {
				errorGoodbyes += count
			}
		}

		// Penalty: 5 points per ERROR goodbye message
		goodbyePenalty := float64(errorGoodbyes) * 5.0

		baseScore := (connectionScore + diversityScore) / 2
		report.OverallScore = max(0.0, baseScore-goodbyePenalty)
	} else {
		report.OverallScore = 0.0
	}

	// Generate summary
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

func (pst *PeerScoreTool) Stop() error {
	if pst.hermesCmd != nil && pst.hermesCmd.Process != nil {
		log.Printf("Stopping Hermes (PID %d)", pst.hermesCmd.Process.Pid)
		return pst.hermesCmd.Process.Signal(syscall.SIGTERM)
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// HTMLTemplateData represents the data structure for the HTML template
type HTMLTemplateData struct {
	GeneratedAt         time.Time       `json:"generated_at"`
	Report              PeerScoreReport `json:"report"`
	ScoreClassification string          `json:"score_classification"`
	ConnectionRate      float64         `json:"connection_rate"`
	ClientList          []ClientStat    `json:"client_list"`
}

type ClientStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

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
                    <p class="text-gray-600 mt-2">Ethereum Network Connectivity Assessment</p>
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
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
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
            <div class="space-y-3">
                {{range $reason, $count := .Report.GoodbyeReasons}}
                <div class="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                    <span class="text-gray-700">{{$reason}}</span>
                    <span class="bg-gray-200 text-gray-800 px-3 py-1 rounded-full text-sm font-semibold">{{$count}}</span>
                </div>
                {{end}}
            </div>
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

func GenerateHTMLReport(jsonFile, outputFile string) error {
	// Read the JSON report
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var report PeerScoreReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Calculate additional data for the template
	templateData := HTMLTemplateData{
		GeneratedAt: time.Now(),
		Report:      report,
	}

	// Calculate connection rate
	if report.TotalConnections > 0 {
		templateData.ConnectionRate = float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
	}

	// Classify score
	if report.ConnectionFailed {
		templateData.ScoreClassification = "Connection Failed"
	} else if report.OverallScore >= 90 {
		templateData.ScoreClassification = "Excellent"
	} else if report.OverallScore >= 80 {
		templateData.ScoreClassification = "Good"
	} else if report.OverallScore >= 60 {
		templateData.ScoreClassification = "Fair"
	} else if report.OverallScore >= 40 {
		templateData.ScoreClassification = "Poor"
	} else {
		templateData.ScoreClassification = "Critical"
	}

	// Convert client map to sorted list
	for client, count := range report.PeersByClient {
		templateData.ClientList = append(templateData.ClientList, ClientStat{
			Name:  client,
			Count: count,
		})
	}

	// Create template with helper functions
	tmpl := template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})

	tmpl, err = tmpl.Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Printf("HTML report generated: %s", outputFile)
	return nil
}

// classifyGoodbyeSeverity categorizes goodbye reasons by severity
func (pst *PeerScoreTool) classifyGoodbyeSeverity(reason string) string {
	switch reason {
	case "client has too many peers":
		return "NORMAL"
	case "client shutdown":
		return "NORMAL"
	case "peer score too low":
		return "ERROR"
	case "client banned this node":
		return "ERROR"
	case "irrelevant network":
		return "ERROR"
	case "unable to verify network":
		return "ERROR"
	case "fault/error":
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func main() {
	var (
		duration      = flag.Duration("duration", 2*time.Minute, "Test duration")
		outputFile    = flag.String("output", "peer-score-report.json", "Output file for results")
		prysmHost     = flag.String("prysm-host", "", "Prysm host connection string")
		prysmHTTPPort = flag.Int("prysm-http-port", 443, "Prysm HTTP port")
		prysmGRPCPort = flag.Int("prysm-grpc-port", 443, "Prysm gRPC port")
	)
	flag.Parse()

	if *prysmHost == "" {
		log.Fatal("prysm-host is required")
	}

	// Build Hermes arguments with configurable HTTP and gRPC ports
	hermesArgs := []string{
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
		"--subscription.topics=beacon_attestation,beacon_block",
	}

	// Only add TLS flag if either HTTP or gRPC port is 443
	if *prysmHTTPPort == 443 || *prysmGRPCPort == 443 {
		hermesArgs = append(hermesArgs, "--prysm.tls")
	}

	config := PeerScoreConfig{
		HermesPath:     "./hermes",
		TestDuration:   *duration,
		ReportInterval: 1 * time.Minute,
		HermesArgs:     hermesArgs,
	}

	tool := NewPeerScoreTool(config)

	// Log connection settings for debugging
	log.Printf("Connection settings:")
	log.Printf("  Prysm Host: %s", *prysmHost)
	log.Printf("  HTTP Port: %d", *prysmHTTPPort)
	log.Printf("  gRPC Port: %d", *prysmGRPCPort)
	log.Printf("  TLS Enabled: %t", (*prysmHTTPPort == 443 || *prysmGRPCPort == 443))
	log.Printf("Hermes arguments: %v", config.HermesArgs)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start Hermes
	if err := tool.StartHermes(ctx); err != nil {
		log.Fatalf("Failed to start Hermes: %v", err)
	}
	defer tool.Stop()

	log.Printf("Running peer score tests for %v...", config.TestDuration)

	// Start periodic status reporting
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tool.mu.RLock()
				peerCount := len(tool.peers)
				handshaked := 0
				for _, peer := range tool.peers {
					if peer.HandshakeOK {
						handshaked++
					}
				}
				tool.mu.RUnlock()
				log.Printf("Status: %d peers connected, %d handshaked", peerCount, handshaked)
			}
		}
	}()

	// Wait for test duration or cancellation
	select {
	case <-ctx.Done():
		log.Println("Test interrupted")
	case <-time.After(config.TestDuration):
		log.Println("Test duration completed")
	}

	// Generate final report
	report := tool.GenerateReport()

	// Save JSON report to file
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal report: %v", err)
	}

	if err := os.WriteFile(*outputFile, reportJSON, 0644); err != nil {
		log.Fatalf("Failed to write report file: %v", err)
	}

	// Generate HTML report
	htmlFile := strings.Replace(*outputFile, ".json", ".html", 1)
	if err := GenerateHTMLReport(*outputFile, htmlFile); err != nil {
		log.Printf("Failed to generate HTML report: %v", err)
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PEER SCORE REPORT")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Overall Score: %.1f%%\n", report.OverallScore)
	fmt.Printf("Test Duration: %v\n", report.Duration)
	fmt.Printf("Total Connections: %d\n", report.TotalConnections)
	fmt.Printf("Successful Handshakes: %d\n", report.SuccessfulHandshakes)
	fmt.Printf("Failed Handshakes: %d\n", report.FailedHandshakes)
	fmt.Printf("Goodbye Messages: %d\n", report.GoodbyeMessages)
	if len(report.GoodbyeReasons) > 0 {
		fmt.Println("Goodbye Reasons:")
		for reason, count := range report.GoodbyeReasons {
			fmt.Printf("  %s: %d\n", reason, count)
		}
	}
	fmt.Printf("Unique Clients: %d\n", report.UniqueClients)
	fmt.Println("Client Distribution:")
	for client, count := range report.PeersByClient {
		fmt.Printf("  %s: %d\n", client, count)
	}
	if len(report.Errors) > 0 {
		fmt.Printf("Errors Encountered: %d\n", len(report.Errors))
		for i, err := range report.Errors {
			fmt.Printf("  [%d] %s\n", i+1, err)
		}
	}
	if report.ConnectionFailed {
		fmt.Println("WARNING: Connection to beacon node failed!")
	}
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Report saved to: %s\n", *outputFile)
}
