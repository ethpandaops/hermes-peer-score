package reports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
			Timeout: 30 * time.Second,
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

// GenerateInsights generates insights from processed data
func (ai *DefaultAIAnalyzer) GenerateInsights(data interface{}) (string, error) {
	// Generate basic insights from the provided data
	// Can be extended to use AI models or more sophisticated analysis
	
	insights := "## Report Insights\n\n"
	insights += "- Analysis of peer connections and network behavior\n"
	insights += "- Identification of potential issues or anomalies\n"
	insights += "- Recommendations for network optimization\n"
	
	return insights, nil
}

// prepareAnalysisData prepares the report data for AI analysis
func (ai *DefaultAIAnalyzer) prepareAnalysisData(report *Report) map[string]interface{} {
	// Create a summary of the report for AI analysis
	summary := map[string]interface{}{
		"validation_mode":       report.ValidationMode,
		"test_duration":        report.Duration.Seconds(),
		"total_connections":    report.TotalConnections,
		"successful_handshakes": report.SuccessfulHandshakes,
		"failed_handshakes":    report.FailedHandshakes,
		"unique_peers":         len(report.Peers),
		"timestamp":            report.Timestamp,
	}
	
	// Add peer statistics
	clientStats := make(map[string]int)
	connectionStats := make(map[string]int)
	scoreStats := make(map[string]interface{})
	
	var totalScores []float64
	
	for _, peerData := range report.Peers {
		if peer, ok := peerData.(map[string]interface{}); ok {
			// Count client types
			if clientType, ok := peer["client_type"].(string); ok {
				clientStats[clientType]++
			}
			
			// Analyze sessions
			if sessions, ok := peer["connection_sessions"].([]interface{}); ok {
				for _, sessionData := range sessions {
					if session, ok := sessionData.(map[string]interface{}); ok {
						// Connection status
						if disconnected, ok := session["disconnected"].(bool); ok {
							if disconnected {
								connectionStats["disconnected"]++
							} else {
								connectionStats["active"]++
							}
						}
						
						// Collect scores
						if scores, ok := session["peer_scores"].([]interface{}); ok {
							for _, scoreData := range scores {
								if score, ok := scoreData.(map[string]interface{}); ok {
									if scoreValue, ok := score["score"].(float64); ok {
										totalScores = append(totalScores, scoreValue)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Calculate score statistics
	if len(totalScores) > 0 {
		sum := 0.0
		min := totalScores[0]
		max := totalScores[0]
		
		for _, score := range totalScores {
			sum += score
			if score < min {
				min = score
			}
			if score > max {
				max = score
			}
		}
		
		scoreStats = map[string]interface{}{
			"average": sum / float64(len(totalScores)),
			"min":     min,
			"max":     max,
			"count":   len(totalScores),
		}
	}
	
	return map[string]interface{}{
		"summary":          summary,
		"client_stats":     clientStats,
		"connection_stats": connectionStats,
		"score_stats":      scoreStats,
	}
}

// callOpenRouterAPI makes a request to the OpenRouter API for analysis
func (ai *DefaultAIAnalyzer) callOpenRouterAPI(data map[string]interface{}, apiKey string) (string, error) {
	// Prepare the prompt for analysis
	prompt := ai.buildAnalysisPrompt(data)
	
	// Prepare the API request
	requestBody := map[string]interface{}{
		"model": "anthropic/claude-3-haiku",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  1000,
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

// buildAnalysisPrompt builds the prompt for AI analysis
func (ai *DefaultAIAnalyzer) buildAnalysisPrompt(data map[string]interface{}) string {
	dataJSON, _ := json.MarshalIndent(data, "", "  ")
	
	prompt := fmt.Sprintf(`Analyze this Ethereum peer scoring report and provide insights about network behavior, connection patterns, and potential issues or recommendations.

Report Data:
%s

Please provide:
1. Summary of network performance
2. Analysis of peer behavior patterns
3. Any anomalies or concerning patterns
4. Recommendations for optimization

Format your response in clear, professional language suitable for network operators.`, string(dataJSON))
	
	return prompt
}

// SetHTTPClient allows setting a custom HTTP client (for testing)
func (ai *DefaultAIAnalyzer) SetHTTPClient(client *http.Client) {
	ai.httpClient = client
}