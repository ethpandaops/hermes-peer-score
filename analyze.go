package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// ClaudeAPIClient handles communication with the Claude API
type ClaudeAPIClient struct {
	APIKey  string
	BaseURL string
	Model   string
}

// ClaudeRequest represents the request structure for OpenRouter API
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in the Claude API request
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response from OpenRouter API
type ClaudeResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
	ID      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ReportSummary contains key metrics extracted from the peer score report
type ReportSummary struct {
	Overview             OverviewMetrics         `json:"overview"`
	ConnectionMetrics    ConnectionMetrics       `json:"connection_metrics"`
	ClientDistribution   map[string]int          `json:"client_distribution"`
	TopDisconnectReasons []DisconnectReasonCount `json:"top_disconnect_reasons"`
	PeerBehaviorSummary  PeerBehaviorSummary     `json:"peer_behavior_summary"`
	TestConfiguration    TestConfigSummary       `json:"test_configuration"`
}

// OverviewMetrics contains high-level test metrics
type OverviewMetrics struct {
	TestDuration         string  `json:"test_duration"`
	TotalPeers           int     `json:"total_peers"`
	TotalConnections     int     `json:"total_connections"`
	SuccessfulHandshakes int     `json:"successful_handshakes"`
	FailedHandshakes     int     `json:"failed_handshakes"`
	SuccessRate          float64 `json:"success_rate"`
}

// ConnectionMetrics contains connection-related statistics
type ConnectionMetrics struct {
	AvgConnectionDuration    string  `json:"avg_connection_duration"`
	MedianConnectionDuration string  `json:"median_connection_duration"`
	ShortConnections         int     `json:"short_connections_under_30s"`
	LongConnections          int     `json:"long_connections_over_5min"`
	ReconnectionRate         float64 `json:"reconnection_rate"`
}

// DisconnectReasonCount represents disconnect reason statistics
type DisconnectReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// PeerBehaviorSummary contains peer behavior analysis
type PeerBehaviorSummary struct {
	PeersWithScores        int     `json:"peers_with_scores"`
	AvgMessagesPerPeer     float64 `json:"avg_messages_per_peer"`
	PeersWithMeshEvents    int     `json:"peers_with_mesh_events"`
	MostActivePeerID       string  `json:"most_active_peer_id"`
	MostActivePeerMsgCount int     `json:"most_active_peer_message_count"`
}

// TestConfigSummary contains test configuration details
type TestConfigSummary struct {
	TestDuration   string `json:"test_duration"`
	ReportInterval string `json:"report_interval"`
	MaxPeers       int    `json:"max_peers"`
	PrysmHost      string `json:"prysm_host"`
}

// NewClaudeAPIClient creates a new Claude API client
func NewClaudeAPIClient(apiKey string) *ClaudeAPIClient {
	return &ClaudeAPIClient{
		APIKey:  apiKey,
		BaseURL: "https://openrouter.ai/api/v1/chat/completions",
		Model:   "anthropic/claude-sonnet-4", // OpenRouter model identifier
	}
}

// AnalyzeReport sends the report summary to Claude for analysis
func (c *ClaudeAPIClient) AnalyzeReport(summary *ReportSummary) (string, error) {
	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %w", err)
	}

	systemPrompt := `You are an expert in peer-to-peer networking and Ethereum beacon chain analysis, specifically analyzing peer behavior for the Hermes network monitoring tool.

Your primary focus is investigating peer connection stability and churn patterns to identify potential issues with Hermes or network configuration that need attention.

IMPORTANT NOTE: "Stream reset errors" are typically normal and expected behavior that occurs after a goodbye/disconnect event. These should NOT be considered concerning or problematic - they are part of the normal connection cleanup process.

Analyze the data with these priorities:
1. **Peer Churn Analysis** - Identify excessive disconnections, short-lived connections, and reconnection patterns
2. **Connection Stability Issues** - Look for signs of network instability or Hermes-specific connection problems  
3. **Disconnect Reason Patterns** - Investigate goodbye codes and reasons that might indicate Hermes behavior issues (excluding stream reset errors which are normal)
4. **Client Interaction Problems** - Identify if certain client types have worse interactions with Hermes
5. **Performance Bottlenecks** - Spot patterns suggesting Hermes resource constraints or configuration issues

Provide actionable insights for the Hermes development team, focusing on:
- Potential bugs or issues in Hermes code
- Configuration changes that might improve peer retention
- Network behavior patterns that suggest underlying problems
- Specific client compatibility issues

IMPORTANT: Provide your response as clean HTML using Tailwind CSS classes. Use these specific classes:
- Headers: h2 with "text-xl font-semibold text-gray-900 mt-6 mb-3", h3 with "text-lg font-semibold text-gray-900 mt-4 mb-2"
- Paragraphs: p with "text-gray-700 mb-3"
- Lists: ul with "list-disc ml-6 space-y-1 mb-3", li with "text-gray-700"
- Bold text: strong with "font-semibold"
- Code/metrics: span with "bg-gray-100 px-1 py-0.5 rounded text-sm font-mono"

Do not include any markdown formatting - return only HTML.`

	userPrompt := fmt.Sprintf(`Analyze this Hermes peer score report data for connection stability and churn issues:

%s

Please provide HTML sections for:
1. **Peer Churn Assessment** - Are connections too short-lived? High reconnection rates?
2. **Disconnect Pattern Analysis** - What are the main reasons peers are leaving? Any concerning patterns?
3. **Client Compatibility Issues** - Do certain Ethereum clients have worse connection stability with Hermes?
4. **Hermes Performance Indicators** - Signs of resource constraints, configuration problems, or code issues?
5. **Actionable Recommendations** - Specific changes to improve Hermes peer retention and stability

Focus on diagnosing potential problems with Hermes itself rather than general network health. Use proper HTML structure with the specified Tailwind classes.`, string(summaryJSON))

	request := ClaudeRequest{
		Model:     c.Model,
		MaxTokens: 2000,
		Messages: []ClaudeMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(responseBody, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(claudeResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenRouter API")
	}

	return claudeResp.Choices[0].Message.Content, nil
}

// SummarizeReport extracts key metrics from the full peer score report
func SummarizeReport(report *PeerScoreReport) *ReportSummary {
	summary := &ReportSummary{
		Overview: OverviewMetrics{
			TestDuration:         report.Duration.String(),
			TotalPeers:           len(report.Peers),
			TotalConnections:     report.TotalConnections,
			SuccessfulHandshakes: report.SuccessfulHandshakes,
			FailedHandshakes:     report.FailedHandshakes,
		},
		ClientDistribution: make(map[string]int),
		TestConfiguration: TestConfigSummary{
			TestDuration:   report.Config.TestDuration.String(),
			ReportInterval: report.Config.ReportInterval.String(),
			MaxPeers:       report.Config.ToolConfig.MaxPeers,
			PrysmHost:      report.Config.ToolConfig.HostWithRedactedSecrets(),
		},
	}

	// Calculate success rate
	if report.TotalConnections > 0 {
		summary.Overview.SuccessRate = float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
	}

	// Analyze connection metrics and peer behavior
	var (
		connectionDurations    []time.Duration
		totalMessages          int
		peersWithScores        int
		peersWithMeshEvents    int
		mostActivePeerID       string
		mostActivePeerMsgCount int
		disconnectReasons      = make(map[string]int)
		reconnections          int
	)

	for peerID, peer := range report.Peers {
		// Client distribution
		if peer.ClientType != "" {
			summary.ClientDistribution[peer.ClientType]++
		}

		// Track total messages and find most active peer
		totalMessages += peer.TotalMessageCount
		if peer.TotalMessageCount > mostActivePeerMsgCount {
			mostActivePeerMsgCount = peer.TotalMessageCount
			mostActivePeerID = peerID
		}

		// Count reconnections (peers with multiple sessions)
		if len(peer.ConnectionSessions) > 1 {
			reconnections++
		}

		for _, session := range peer.ConnectionSessions {
			// Connection duration analysis
			connectionDurations = append(connectionDurations, session.ConnectionDuration)

			// Count peers with scores and mesh events
			if len(session.PeerScores) > 0 {
				peersWithScores++
			}
			if len(session.MeshEvents) > 0 {
				peersWithMeshEvents++
			}

			// Analyze disconnect reasons
			for _, goodbye := range session.GoodbyeEvents {
				disconnectReasons[goodbye.Reason]++
			}
		}
	}

	// Calculate connection metrics
	if len(connectionDurations) > 0 {
		// Sort durations for median calculation
		sort.Slice(connectionDurations, func(i, j int) bool {
			return connectionDurations[i] < connectionDurations[j]
		})

		// Calculate average
		var totalDuration time.Duration
		for _, d := range connectionDurations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(connectionDurations))
		summary.ConnectionMetrics.AvgConnectionDuration = avgDuration.String()

		// Calculate median
		medianIdx := len(connectionDurations) / 2
		summary.ConnectionMetrics.MedianConnectionDuration = connectionDurations[medianIdx].String()

		// Count short and long connections
		for _, d := range connectionDurations {
			if d < 30*time.Second {
				summary.ConnectionMetrics.ShortConnections++
			}
			if d > 5*time.Minute {
				summary.ConnectionMetrics.LongConnections++
			}
		}
	}

	// Calculate reconnection rate
	if len(report.Peers) > 0 {
		summary.ConnectionMetrics.ReconnectionRate = float64(reconnections) / float64(len(report.Peers)) * 100
	}

	// Peer behavior summary
	summary.PeerBehaviorSummary = PeerBehaviorSummary{
		PeersWithScores:        peersWithScores,
		PeersWithMeshEvents:    peersWithMeshEvents,
		MostActivePeerID:       mostActivePeerID,
		MostActivePeerMsgCount: mostActivePeerMsgCount,
	}

	if len(report.Peers) > 0 {
		summary.PeerBehaviorSummary.AvgMessagesPerPeer = float64(totalMessages) / float64(len(report.Peers))
	}

	// Top disconnect reasons (limit to top 10)
	type reasonCount struct {
		reason string
		count  int
	}
	var reasons []reasonCount
	for reason, count := range disconnectReasons {
		reasons = append(reasons, reasonCount{reason, count})
	}
	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].count > reasons[j].count
	})

	// Take top 10
	maxReasons := 10
	if len(reasons) < maxReasons {
		maxReasons = len(reasons)
	}
	for i := 0; i < maxReasons; i++ {
		summary.TopDisconnectReasons = append(summary.TopDisconnectReasons, DisconnectReasonCount{
			Reason: reasons[i].reason,
			Count:  reasons[i].count,
		})
	}

	return summary
}

// AnalyzePeerScoreReport performs AI analysis of a peer score report
func AnalyzePeerScoreReport(inputFile, outputFile, apiKey string) error {
	// Read and parse the report
	reportData, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read report file: %w", err)
	}

	var report PeerScoreReport
	if err := json.Unmarshal(reportData, &report); err != nil {
		return fmt.Errorf("failed to parse report JSON: %w", err)
	}

	// Create summary for analysis
	summary := SummarizeReport(&report)

	// Initialize Claude API client
	client := NewClaudeAPIClient(apiKey)

	// Get AI analysis
	analysis, err := client.AnalyzeReport(summary)
	if err != nil {
		return fmt.Errorf("failed to get AI analysis: %w", err)
	}

	// Prepare output content
	outputContent := fmt.Sprintf(`# Peer Score Report Analysis

Generated: %s
Report File: %s

## AI Analysis

%s

## Report Summary Data

### Overview
- Test Duration: %s
- Total Peers: %d
- Total Connections: %d
- Successful Handshakes: %d
- Failed Handshakes: %d
- Success Rate: %.1f%%

### Connection Metrics
- Average Connection Duration: %s
- Median Connection Duration: %s
- Short Connections (<30s): %d
- Long Connections (>5min): %d
- Reconnection Rate: %.1f%%

### Client Distribution
%s

### Top Disconnect Reasons
%s

### Peer Behavior
- Peers with Scores: %d
- Average Messages per Peer: %.1f
- Peers with Mesh Events: %d
- Most Active Peer: %s (%d messages)

---
*Analysis generated using Claude AI*
`,
		time.Now().Format(time.RFC3339),
		inputFile,
		analysis,
		summary.Overview.TestDuration,
		summary.Overview.TotalPeers,
		summary.Overview.TotalConnections,
		summary.Overview.SuccessfulHandshakes,
		summary.Overview.FailedHandshakes,
		summary.Overview.SuccessRate,
		summary.ConnectionMetrics.AvgConnectionDuration,
		summary.ConnectionMetrics.MedianConnectionDuration,
		summary.ConnectionMetrics.ShortConnections,
		summary.ConnectionMetrics.LongConnections,
		summary.ConnectionMetrics.ReconnectionRate,
		formatClientDistribution(summary.ClientDistribution),
		formatDisconnectReasons(summary.TopDisconnectReasons),
		summary.PeerBehaviorSummary.PeersWithScores,
		summary.PeerBehaviorSummary.AvgMessagesPerPeer,
		summary.PeerBehaviorSummary.PeersWithMeshEvents,
		summary.PeerBehaviorSummary.MostActivePeerID,
		summary.PeerBehaviorSummary.MostActivePeerMsgCount,
	)

	// Write output file
	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// formatClientDistribution formats client distribution for display
func formatClientDistribution(clients map[string]int) string {
	if len(clients) == 0 {
		return "No client data available"
	}

	var lines []string
	for client, count := range clients {
		lines = append(lines, fmt.Sprintf("- %s: %d", client, count))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// formatDisconnectReasons formats disconnect reasons for display
func formatDisconnectReasons(reasons []DisconnectReasonCount) string {
	if len(reasons) == 0 {
		return "No disconnect reasons recorded"
	}

	var lines []string
	for _, reason := range reasons {
		lines = append(lines, fmt.Sprintf("- %s: %d", reason.Reason, reason.Count))
	}
	return strings.Join(lines, "\n")
}
