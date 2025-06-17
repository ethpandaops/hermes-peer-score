package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/hermes-peer-score/constants"
)

// DefaultFileManager implements the FileManager interface.
type DefaultFileManager struct {
	logger logrus.FieldLogger
}

// NewDefaultFileManager creates a new file manager.
func NewDefaultFileManager(logger logrus.FieldLogger) *DefaultFileManager {
	return &DefaultFileManager{
		logger: logger.WithField("component", "file_manager"),
	}
}

// SaveJSON saves data as JSON to the specified filename.
func (fm *DefaultFileManager) SaveJSON(filename string, data interface{}) error {
	var jsonData []byte

	var err error

	switch v := data.(type) {
	case []byte:
		jsonData = v
	case string:
		jsonData = []byte(v)
	default:
		jsonData, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
	}

	if err := os.WriteFile(filename, jsonData, constants.DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", filename, err)
	}

	fm.logger.WithField("filename", filename).Debug("JSON file saved")

	return nil
}

// SaveHTML saves HTML content to the specified filename.
func (fm *DefaultFileManager) SaveHTML(filename string, content string) error {
	if err := os.WriteFile(filename, []byte(content), constants.DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write HTML file %s: %w", filename, err)
	}

	fm.logger.WithField("filename", filename).Debug("HTML file saved")

	return nil
}

// FileExists checks if a file exists at the given path.
func (fm *DefaultFileManager) FileExists(filename string) bool {
	_, err := os.Stat(filename)

	return !os.IsNotExist(err)
}

// GenerateFilename generates a filename with the base name and timestamp.
func (fm *DefaultFileManager) GenerateFilename(base string, timestamp time.Time) string {
	return fmt.Sprintf("%s-%s", base, timestamp.Format("2006-01-02_15-04-05"))
}

// ReadJSON reads and parses a JSON file.
func (fm *DefaultFileManager) ReadJSON(filename string, v interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", filename, err)
	}

	fm.logger.WithField("filename", filename).Debug("JSON file read")

	return nil
}

// WriteFile writes arbitrary data to a file.
func (fm *DefaultFileManager) WriteFile(filename string, data []byte) error {
	if err := os.WriteFile(filename, data, constants.DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	fm.logger.WithField("filename", filename).Debug("File written")

	return nil
}

// GetFileSize returns the size of a file in bytes.
func (fm *DefaultFileManager) GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file %s: %w", filename, err)
	}

	return info.Size(), nil
}

// DeleteFile removes a file.
func (fm *DefaultFileManager) DeleteFile(filename string) error {
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", filename, err)
	}

	fm.logger.WithField("filename", filename).Debug("File deleted")

	return nil
}
