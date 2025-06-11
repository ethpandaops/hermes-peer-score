package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/constants"
	"github.com/ethpandaops/hermes-peer-score/internal/reports/templates"
)

// DefaultGenerator implements the Generator interface
type DefaultGenerator struct {
	templateManager *templates.Manager
	fileManager     FileManager
	dataProcessor   DataProcessor
	aiAnalyzer      AIAnalyzer
	logger          logrus.FieldLogger
}

// NewGenerator creates a new report generator
func NewGenerator(logger logrus.FieldLogger) (*DefaultGenerator, error) {
	templateManager := templates.NewManager(logger)
	
	// Load templates
	if err := templateManager.LoadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}
	
	return &DefaultGenerator{
		templateManager: templateManager,
		fileManager:     NewDefaultFileManager(logger),
		dataProcessor:   NewDefaultDataProcessor(logger),
		aiAnalyzer:      NewDefaultAIAnalyzer(logger),
		logger:          logger.WithField("component", "report_generator"),
	}, nil
}

// GenerateJSON generates a JSON report and saves it to a file
func (g *DefaultGenerator) GenerateJSON(report *Report) (string, error) {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}
	
	// Generate timestamped filename
	filename := g.generateTimestampedFilename(report.ValidationMode, constants.DefaultJSONReportFile, report.Timestamp)
	
	if err := g.fileManager.SaveJSON(filename, reportJSON); err != nil {
		return "", fmt.Errorf("failed to save JSON report: %w", err)
	}
	
	g.logger.WithField("filename", filename).Info("JSON report generated successfully")
	return filename, nil
}

// GenerateHTML generates an HTML report and saves it to a file
func (g *DefaultGenerator) GenerateHTML(report *Report) (string, error) {
	return g.generateHTMLReport(report, "")
}

// GenerateHTMLWithAI generates an HTML report with AI analysis
func (g *DefaultGenerator) GenerateHTMLWithAI(report *Report, apiKey string) (string, error) {
	// Generate AI analysis first
	aiAnalysis, err := g.aiAnalyzer.AnalyzeReport(report, apiKey)
	if err != nil {
		g.logger.WithError(err).Warn("Failed to generate AI analysis, proceeding without it")
		aiAnalysis = ""
	}
	
	return g.generateHTMLReport(report, aiAnalysis)
}

// generateHTMLReport is the common HTML generation logic
func (g *DefaultGenerator) generateHTMLReport(report *Report, aiAnalysis string) (string, error) {
	// Process data for template
	templateData, err := g.dataProcessor.FormatForTemplate(report)
	if err != nil {
		return "", fmt.Errorf("failed to format data for template: %w", err)
	}
	
	// Generate filename first to use in template
	htmlFilename := g.generateTimestampedFilename(report.ValidationMode, constants.DefaultHTMLReportFile, report.Timestamp)
	dataFilename := g.generateTimestampedFilename(report.ValidationMode, constants.DefaultDataJSFile, report.Timestamp)
	
	// Add AI analysis and data file if provided
	if reportData, ok := templateData.(map[string]interface{}); ok {
		reportData["AIAnalysis"] = aiAnalysis
		reportData["DataFile"] = dataFilename
	}
	
	// Render template
	htmlContent, err := g.templateManager.RenderReport(templateData)
	if err != nil {
		return "", fmt.Errorf("failed to render HTML template: %w", err)
	}
	
	// HTML filename was already generated above
	
	if err := g.fileManager.SaveHTML(htmlFilename, htmlContent); err != nil {
		return "", fmt.Errorf("failed to save HTML report: %w", err)
	}
	
	// Generate data file for JavaScript (filename was already generated above)
	if err := g.generateDataFile(report, dataFilename); err != nil {
		g.logger.WithError(err).Warn("Failed to generate data file")
	}
	
	g.logger.WithFields(logrus.Fields{
		"html_file": htmlFilename,
		"data_file": dataFilename,
	}).Info("HTML report generated successfully")
	
	return htmlFilename, nil
}

// generateDataFile creates a JavaScript data file for the HTML report
func (g *DefaultGenerator) generateDataFile(report *Report, filename string) error {
	// Process the full report data for JavaScript consumption
	processedData, err := g.dataProcessor.ProcessPeerData(report.Peers)
	if err != nil {
		return fmt.Errorf("failed to process peer data: %w", err)
	}
	
	// Extract the peers array from the processed data
	var peersArray interface{}
	if processedMap, ok := processedData.(map[string]interface{}); ok {
		if peers, exists := processedMap["peers"]; exists {
			peersArray = peers
		} else {
			peersArray = []interface{}{} // Empty array fallback
		}
	} else {
		peersArray = processedData // Use as-is if not a map
	}
	
	// Create the complete data structure including event counts
	jsData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"format_version": "1.0",
			"processed_at":   report.Timestamp.Format(time.RFC3339),
			"total_peers":    len(report.Peers),
		},
		"peers":            peersArray,
		"peerEventCounts":  report.PeerEventCounts,
	}
	
	dataJSON, err := json.MarshalIndent(jsData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	// Wrap in JavaScript variable
	jsContent := fmt.Sprintf("window.reportData = %s;", string(dataJSON))
	
	//nolint:gosec // Controlled input
	if err := os.WriteFile(filename, []byte(jsContent), constants.DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}
	
	return nil
}

// GenerateHTMLFromJSON generates HTML report from existing JSON file
func (g *DefaultGenerator) GenerateHTMLFromJSON(jsonFile, outputFile string) error {
	return g.GenerateHTMLFromJSONWithAI(jsonFile, outputFile, "")
}

// GenerateHTMLFromJSONWithAI generates HTML report from existing JSON file with optional AI analysis
func (g *DefaultGenerator) GenerateHTMLFromJSONWithAI(jsonFile, outputFile, apiKey string) error {
	// Check if input file exists
	if !g.fileManager.FileExists(jsonFile) {
		return fmt.Errorf("input JSON file does not exist: %s", jsonFile)
	}
	
	// Read and parse JSON file
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}
	
	var report Report
	if err := json.Unmarshal(jsonData, &report); err != nil {
		return fmt.Errorf("failed to parse JSON report: %w", err)
	}
	
	// Generate AI analysis if API key provided
	var aiAnalysis string
	if apiKey != "" {
		analysis, err := g.aiAnalyzer.AnalyzeReport(&report, apiKey)
		if err != nil {
			g.logger.WithError(err).Warn("Failed to generate AI analysis")
		} else {
			aiAnalysis = analysis
		}
	}
	
	// Process data for template
	templateData, err := g.dataProcessor.FormatForTemplate(&report)
	if err != nil {
		return fmt.Errorf("failed to format data for template: %w", err)
	}
	
	// Generate data filename
	dataFilename := g.generateTimestampedFilename(report.ValidationMode, constants.DefaultDataJSFile, report.Timestamp)
	
	// Add AI analysis and data file to template data
	if reportData, ok := templateData.(map[string]interface{}); ok {
		reportData["AIAnalysis"] = aiAnalysis
		reportData["DataFile"] = dataFilename
	}
	
	// Render template
	htmlContent, err := g.templateManager.RenderReport(templateData)
	if err != nil {
		return fmt.Errorf("failed to render HTML template: %w", err)
	}
	
	// Save HTML file
	if err := g.fileManager.SaveHTML(outputFile, htmlContent); err != nil {
		return fmt.Errorf("failed to save HTML file: %w", err)
	}
	
	// Generate data file for JavaScript
	if err := g.generateDataFile(&report, dataFilename); err != nil {
		g.logger.WithError(err).Warn("Failed to generate data file")
	}
	
	g.logger.WithFields(logrus.Fields{
		"input":  jsonFile,
		"output": outputFile,
	}).Info("HTML report generated from JSON")
	
	return nil
}

// generateTimestampedFilename creates a filename with timestamp and validation mode
func (g *DefaultGenerator) generateTimestampedFilename(validationMode, baseFilename string, timestamp time.Time) string {
	// Extract extension and name parts
	ext := filepath.Ext(baseFilename)
	nameWithoutExt := strings.TrimSuffix(baseFilename, ext)
	
	// Insert validation mode and timestamp before the extension
	return fmt.Sprintf("%s-%s-%s%s", nameWithoutExt, validationMode, timestamp.Format("2006-01-02_15-04-05"), ext)
}

// SetTemplateManager allows injecting a different template manager (for testing)
func (g *DefaultGenerator) SetTemplateManager(tm *templates.Manager) {
	g.templateManager = tm
}

// SetFileManager allows injecting a different file manager (for testing)
func (g *DefaultGenerator) SetFileManager(fm FileManager) {
	g.fileManager = fm
}

// SetDataProcessor allows injecting a different data processor (for testing)
func (g *DefaultGenerator) SetDataProcessor(dp DataProcessor) {
	g.dataProcessor = dp
}

// SetAIAnalyzer allows injecting a different AI analyzer (for testing)
func (g *DefaultGenerator) SetAIAnalyzer(ai AIAnalyzer) {
	g.aiAnalyzer = ai
}