package reports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

// DefaultAIAnalyzer implements the AIAnalyzer interface
type DefaultAIAnalyzer struct {
	logger     logrus.FieldLogger
	httpClient *http.Client
}

// NewDefaultAIAnalyzer creates a new AI analyzer
func NewDefaultAIAnalyzer(logger logrus.FieldLogger) *DefaultAIAnalyzer {
	return &DefaultAIAnalyzer{
		logger: logger.WithField("component", "ai_analyzer"),
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // Increased timeout for DeepSeek
		},
	}
}

// AnalyzeReport generates AI analysis for the given report
func (ai *DefaultAIAnalyzer) AnalyzeReport(report *Report, apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("API key is required for AI analysis")
	}

	ai.logger.Info("Generating AI analysis for report")

	// Prepare data for AI analysis
	analysisData := ai.prepareAnalysisData(report)

	// Generate analysis using OpenRouter API
	analysis, err := ai.callOpenRouterAPI(analysisData, apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenRouter API: %w", err)
	}

	ai.logger.Info("AI analysis generated successfully")
	return analysis, nil
}

// GenerateInsights generates insights from processed data.
// Only used in tests.
func (ai *DefaultAIAnalyzer) GenerateInsights(data interface{}) (string, error) {
	// Generate basic insights from the provided data
	// Can be extended to use AI models or more sophisticated analysis

	insights := "## Report Insights\n\n"
	insights += "- Analysis of peer connections and network behavior\n"
	insights += "- Identification of potential issues or anomalies\n"
	insights += "- Recommendations for network optimization\n"

	return insights, nil
}

// prepareAnalysisData prepares the report data for AI analysis (enhanced like old implementation)
func (ai *DefaultAIAnalyzer) prepareAnalysisData(report *Report) map[string]interface{} {
	summary := map[string]interface{}{
		"overview": map[string]interface{}{
			"test_duration":         report.Duration.String(),
			"total_peers":           len(report.Peers),
			"total_connections":     report.TotalConnections,
			"successful_handshakes": report.SuccessfulHandshakes,
			"failed_handshakes":     report.FailedHandshakes,
			"success_rate":          float64(0),
		},
		"connection_metrics":     map[string]interface{}{},
		"client_distribution":    make(map[string]int),
		"top_disconnect_reasons": []map[string]interface{}{},
		"peer_behavior_summary":  map[string]interface{}{},
		"test_configuration": map[string]interface{}{
			"validation_mode": report.ValidationMode,
			"test_duration":   report.Duration.String(),
		},
	}

	// Calculate success rate
	if report.TotalConnections > 0 {
		summary["overview"].(map[string]interface{})["success_rate"] = float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
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

	for peerID, peerData := range report.Peers {
		if peer, ok := peerData.(map[string]interface{}); ok {
			// Client distribution
			if clientType, ok := peer["client_type"].(string); ok {
				summary["client_distribution"].(map[string]int)[clientType]++
			}

			// Get message count for this peer
			var peerMsgCount int
			if sessions, ok := peer["connection_sessions"].([]interface{}); ok {
				if len(sessions) > 1 {
					reconnections++
				}

				for _, sessionData := range sessions {
					if session, ok := sessionData.(map[string]interface{}); ok {
						// Message count
						if msgCount, ok := session["message_count"].(float64); ok {
							peerMsgCount += int(msgCount)
						}

						// Connection duration
						if duration, ok := session["connection_duration"].(float64); ok {
							connectionDurations = append(connectionDurations, time.Duration(duration)*time.Nanosecond)
						}

						// Count peers with scores and mesh events
						if scores, ok := session["peer_scores"].([]interface{}); ok && len(scores) > 0 {
							peersWithScores++
						}

						if meshEvents, ok := session["mesh_events"].([]interface{}); ok && len(meshEvents) > 0 {
							peersWithMeshEvents++
						}

						// Analyze goodbye events for disconnect reasons
						if goodbyes, ok := session["goodbye_events"].([]interface{}); ok {
							for _, goodbyeData := range goodbyes {
								if goodbye, ok := goodbyeData.(map[string]interface{}); ok {
									if reason, ok := goodbye["reason"].(string); ok {
										disconnectReasons[reason]++
									}
								}
							}
						}
					}
				}
			}

			totalMessages += peerMsgCount
			if peerMsgCount > mostActivePeerMsgCount {
				mostActivePeerMsgCount = peerMsgCount
				mostActivePeerID = peerID
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
		shortConnections := 0
		longConnections := 0

		for _, d := range connectionDurations {
			totalDuration += d
			if d < 30*time.Second {
				shortConnections++
			}
			if d > 5*time.Minute {
				longConnections++
			}
		}

		avgDuration := totalDuration / time.Duration(len(connectionDurations))
		medianIdx := len(connectionDurations) / 2

		summary["connection_metrics"] = map[string]interface{}{
			"avg_connection_duration":     avgDuration.String(),
			"median_connection_duration":  connectionDurations[medianIdx].String(),
			"short_connections_under_30s": shortConnections,
			"long_connections_over_5min":  longConnections,
			"reconnection_rate":           float64(reconnections) / float64(len(report.Peers)) * 100,
		}
	}

	// Peer behavior summary
	summary["peer_behavior_summary"] = map[string]interface{}{
		"peers_with_scores":          peersWithScores,
		"peers_with_mesh_events":     peersWithMeshEvents,
		"most_active_peer_id":        mostActivePeerID,
		"most_active_peer_msg_count": mostActivePeerMsgCount,
		"avg_messages_per_peer":      float64(0),
	}

	if len(report.Peers) > 0 {
		summary["peer_behavior_summary"].(map[string]interface{})["avg_messages_per_peer"] = float64(totalMessages) / float64(len(report.Peers))
	}

	// Top disconnect reasons (limit to top 5)
	type reasonCount struct {
		reason string
		count  int
	}

	reasons := make([]reasonCount, 0)
	for reason, count := range disconnectReasons {
		reasons = append(reasons, reasonCount{reason, count})
	}

	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].count > reasons[j].count
	})

	// Take top 5
	maxReasons := 5
	if len(reasons) < maxReasons {
		maxReasons = len(reasons)
	}

	topReasons := make([]map[string]interface{}, 0)
	for i := 0; i < maxReasons; i++ {
		topReasons = append(topReasons, map[string]interface{}{
			"reason": reasons[i].reason,
			"count":  reasons[i].count,
		})
	}
	summary["top_disconnect_reasons"] = topReasons

	return summary
}

// callOpenRouterAPI makes a request to the OpenRouter API for analysis
func (ai *DefaultAIAnalyzer) callOpenRouterAPI(data map[string]interface{}, apiKey string) (string, error) {
	// Get model from environment or use DeepSeek default
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "deepseek/deepseek-r1-0528" // Default fallback to DeepSeek
	}

	// Prepare the prompt for analysis
	systemPrompt, userPrompt := ai.buildAnalysisPrompts(data)

	// Prepare the API request
	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"max_tokens":  8000, // Increased for DeepSeek which has higher token limits
		"temperature": 0.7,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/ethpandaops/hermes-peer-score")
	req.Header.Set("X-Title", "Hermes Peer Score Tool")

	// Make the request
	resp, err := ai.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return response.Choices[0].Message.Content, nil
}

// buildAnalysisPrompts builds the system and user prompts for AI analysis (matching old implementation)
func (ai *DefaultAIAnalyzer) buildAnalysisPrompts(data map[string]interface{}) (string, string) {
	dataJSON, _ := json.MarshalIndent(data, "", "  ")

	systemPrompt := `You are an expert in peer-to-peer networking and Ethereum beacon chain analysis, specifically analyzing network monitoring data from the Hermes tool.

CRITICAL CONTEXT: Hermes is a GossipSub listener and network tracer that connects to an upstream Prysm beacon chain node to monitor network events. Hermes is NOT a full Ethereum client - it's a passive monitoring tool that subscribes to pubsub topics and traces protocol interactions. It "leeches" events from the network through its connection to Prysm.

Your analysis should focus on understanding why OTHER PEERS are disconnecting FROM Hermes, not the other way around. Hermes wants to maintain stable connections to observe network behavior, so disconnections represent a loss of monitoring capability.

IMPORTANT NOTES:

- "Stream reset errors" are normal cleanup events after disconnections
- "Client has too many peers" means OTHER clients are dropping Hermes because they've reached their peer limits
- Hermes participates in the gossipsub network to monitor, but is not implementing peer scoring itself

Analyze the data with these priorities:

1. **Monitoring Stability** - Why are peers dropping connections to Hermes? Is Hermes being seen as an undesirable peer?
2. **Network Participation** - Is Hermes successfully participating in gossipsub to maintain monitoring visibility?
3. **Peer Acceptance** - Are certain client types more/less likely to maintain connections with Hermes?
4. **Configuration Impact** - Do Hermes settings affect how other peers perceive and interact with it?
5. **Data Collection Quality** - Are short connections providing sufficient monitoring data?

Provide actionable insights for improving Hermes as a network monitoring tool, focusing on:
- How to make Hermes a more "attractive" peer that others want to keep connected
- Configuration changes to improve monitoring stability and data collection
- Understanding network dynamics that affect monitoring tools like Hermes

IMPORTANT: Provide your response as clean HTML using Tailwind CSS classes. Use these specific classes:

- Headers: h2 with "text-xl font-semibold text-gray-900 mt-6 mb-3", h3 with "text-lg font-semibold text-gray-900 mt-4 mb-2"
- Paragraphs: p with "text-gray-700 mb-3"
- Lists: ul with "list-disc ml-6 space-y-1 mb-3", li with "text-gray-700"
- Bold text: strong with "font-semibold"
- Code/metrics: span with "bg-gray-100 px-1 py-0.5 rounded text-sm font-mono"
- This HTML will be embedded via Javascript, so to avoid any issues, ensure basic, clean HTML is used only

Do not include any markdown formatting - return only HTML.`

	userPrompt := fmt.Sprintf(`Analyze this Hermes network monitoring data to understand why peers are disconnecting from our monitoring tool:

%s

Please provide HTML sections for:

1. **Monitoring Impact Assessment** - How do short connection durations affect Hermes's ability to collect network data?
2. **Peer Rejection Analysis** - Why are other clients dropping connections to Hermes? What patterns suggest Hermes is seen as undesirable?
3. **Client Behavior Patterns** - Which client types maintain better connections with Hermes monitoring? Any biases in peer selection?
4. **Network Integration Issues** - Is Hermes participating effectively in gossipsub without being too resource-intensive for other peers?
5. **Monitoring Optimization** - How can Hermes become a better network participant to maintain stable monitoring connections?

Focus on improving Hermes as a passive network monitoring tool that other peers want to stay connected to. Use proper HTML structure with the specified Tailwind classes.`, string(dataJSON))

	return systemPrompt, userPrompt
}

// SetHTTPClient allows setting a custom HTTP client (for testing)
func (ai *DefaultAIAnalyzer) SetHTTPClient(client *http.Client) {
	ai.httpClient = client
}
