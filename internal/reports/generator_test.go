package reports

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	
	"github.com/ethpandaops/hermes-peer-score/constants"
)

// MockFileManager for testing
type MockFileManager struct {
	files map[string][]byte
}

func NewMockFileManager() *MockFileManager {
	return &MockFileManager{
		files: make(map[string][]byte),
	}
}

func (m *MockFileManager) SaveJSON(filename string, data interface{}) error {
	switch v := data.(type) {
	case []byte:
		m.files[filename] = v
	case string:
		m.files[filename] = []byte(v)
	}
	return nil
}

func (m *MockFileManager) SaveHTML(filename string, content string) error {
	m.files[filename] = []byte(content)
	return nil
}

func (m *MockFileManager) FileExists(filename string) bool {
	_, exists := m.files[filename]
	return exists
}

func (m *MockFileManager) GenerateFilename(base string, timestamp time.Time) string {
	return base + "-" + timestamp.Format("2006-01-02_15-04-05")
}

// MockDataProcessor for testing
type MockDataProcessor struct{}

func (m *MockDataProcessor) ProcessPeerData(peers map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"peers": []interface{}{},
		"metadata": map[string]interface{}{
			"total_peers": len(peers),
		},
	}, nil
}

func (m *MockDataProcessor) CalculateSummaryStats(report *Report) (interface{}, error) {
	return map[string]interface{}{
		"test_duration":         report.Duration.Seconds(),
		"total_connections":     report.TotalConnections,
		"successful_handshakes": report.SuccessfulHandshakes,
		"failed_handshakes":     report.FailedHandshakes,
		"unique_peers":          len(report.Peers),
	}, nil
}

func (m *MockDataProcessor) FormatForTemplate(report *Report) (interface{}, error) {
	summary, _ := m.CalculateSummaryStats(report)
	return map[string]interface{}{
		"GeneratedAt":      time.Now(),
		"Summary":          summary,
		"ValidationMode":   report.ValidationMode,
		"ValidationConfig": report.ValidationConfig,
	}, nil
}

// MockAIAnalyzer for testing
type MockAIAnalyzer struct{}

func (m *MockAIAnalyzer) AnalyzeReport(report *Report, apiKey string) (string, error) {
	return "Mock AI analysis result", nil
}

func (m *MockAIAnalyzer) GenerateInsights(data interface{}) (string, error) {
	return "Mock insights", nil
}

func TestReportGenerator(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests
	
	// We can't easily test the real generator due to template loading,
	// so we'll test the components individually
	
	// Test file manager
	fm := NewMockFileManager()
	
	err := fm.SaveJSON("test.json", `{"test": "data"}`)
	if err != nil {
		t.Errorf("Expected no error saving JSON, got %v", err)
	}
	
	if !fm.FileExists("test.json") {
		t.Error("Expected file to exist after saving")
	}
	
	// Test data processor
	dp := NewDefaultDataProcessor(logger)
	
	report := &Report{
		ValidationMode:       "delegated",
		ValidationConfig:     map[string]interface{}{},
		Timestamp:            time.Now(),
		Duration:             2 * time.Minute,
		TotalConnections:     10,
		SuccessfulHandshakes: 8,
		FailedHandshakes:     2,
		Peers:                map[string]interface{}{},
	}
	
	summary, err := dp.CalculateSummaryStats(report)
	if err != nil {
		t.Errorf("Expected no error calculating summary, got %v", err)
	}
	
	if summary == nil {
		t.Error("Expected summary to be non-nil")
	}
	
	// Test template data formatting
	templateData, err := dp.FormatForTemplate(report)
	if err != nil {
		t.Errorf("Expected no error formatting for template, got %v", err)
	}
	
	if templateData == nil {
		t.Error("Expected template data to be non-nil")
	}
}

func TestDataProcessor(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	
	dp := NewDefaultDataProcessor(logger)
	
	// Test peer data processing
	peers := map[string]interface{}{
		"peer1": map[string]interface{}{
			"client_type":         constants.Lighthouse,
			"client_agent":        "lighthouse/v1.0.0",
			"connection_sessions": []interface{}{},
		},
		"peer2": map[string]interface{}{
			"client_type":         constants.Prysm,
			"client_agent":        "prysm/v2.0.0",
			"connection_sessions": []interface{}{},
		},
	}
	
	processed, err := dp.ProcessPeerData(peers)
	if err != nil {
		t.Errorf("Expected no error processing peers, got %v", err)
	}
	
	if processed == nil {
		t.Error("Expected processed data to be non-nil")
	}
	
	// Test short peer ID formatting
	shortID := dp.formatShortPeerID("very-long-peer-id-that-should-be-shortened")
	if len(shortID) != 12 {
		t.Errorf("Expected short ID length 12, got %d", len(shortID))
	}
	
	shortOriginal := dp.formatShortPeerID("short")
	if shortOriginal != "short" {
		t.Errorf("Expected short ID to remain unchanged, got %s", shortOriginal)
	}
}

func TestAIAnalyzer(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	
	analyzer := NewDefaultAIAnalyzer(logger)
	
	// Test insights generation (no API call)
	insights, err := analyzer.GenerateInsights(map[string]interface{}{
		"test": "data",
	})
	
	if err != nil {
		t.Errorf("Expected no error generating insights, got %v", err)
	}
	
	if insights == "" {
		t.Error("Expected insights to be non-empty")
	}
	
	// Test analysis data preparation
	report := &Report{
		ValidationMode:       "delegated",
		Duration:             2 * time.Minute,
		TotalConnections:     10,
		SuccessfulHandshakes: 8,
		FailedHandshakes:     2,
		Peers:                map[string]interface{}{},
		Timestamp:            time.Now(),
	}
	
	data := analyzer.prepareAnalysisData(report)
	if data == nil {
		t.Error("Expected analysis data to be non-nil")
	}
	
	summary, ok := data["summary"].(map[string]interface{})
	if !ok {
		t.Error("Expected summary to be a map")
	}
	
	if summary["validation_mode"] != "delegated" {
		t.Error("Expected validation mode to be delegated")
	}
}