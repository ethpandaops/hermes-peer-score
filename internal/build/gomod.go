package build

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ethpandaops/hermes-peer-score/internal/config"
)

// GoModManager handles go.mod file operations for validation modes.
type GoModManager struct{}

// NewGoModManager creates a new GoModManager.
func NewGoModManager() *GoModManager {
	return &GoModManager{}
}

// UpdateForValidationMode updates go.mod to use the appropriate Hermes version for the given validation mode.
func (g *GoModManager) UpdateForValidationMode(validationMode config.ValidationMode) error {
	validationConfigs := config.GetValidationConfigs()
	validationConfig := validationConfigs[validationMode]

	// Define the replacement line based on validation mode
	var newReplaceLine string

	switch validationMode {
	case config.ValidationModeDelegated:
		newReplaceLine = "replace github.com/probe-lab/hermes => github.com/ethpandaops/hermes v0.0.4-0.20250513093811-320c1c3ee6e2"
	case config.ValidationModeIndependent:
		newReplaceLine = "replace github.com/probe-lab/hermes => github.com/ethpandaops/hermes v0.0.4-0.20250613124328-491d55340eb7"
	default:
		return fmt.Errorf("unknown validation mode: %s", validationMode)
	}

	// Read the current go.mod file
	file, err := os.Open("go.mod")
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	var lines []string

	scanner := bufio.NewScanner(file)
	replaceLine := regexp.MustCompile(`^replace\s+github\.com/probe-lab/hermes\s+=>\s+github\.com/ethpandaops/hermes\s+v.+$`)

	found := false

	for scanner.Scan() {
		line := scanner.Text()

		if replaceLine.MatchString(line) {
			lines = append(lines, newReplaceLine)
			found = true
		} else {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	// If no replace line was found, add it after the module declaration
	if !found {
		// Find module declaration and add replace line after it
		newLines := []string{}
		for i, line := range lines {
			newLines = append(newLines, line)
			if strings.HasPrefix(strings.TrimSpace(line), "module ") && i+1 < len(lines) {
				// Add empty line and replace directive after module
				newLines = append(newLines, "")
				newLines = append(newLines, newReplaceLine)
			}
		}

		lines = newLines
	}

	// Write the updated go.mod file
	output := strings.Join(lines, "\n")

	//nolint:gosec // controlled file write
	if err := os.WriteFile("go.mod", []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	fmt.Printf("Updated go.mod for %s validation mode\n", validationMode)
	fmt.Printf("Hermes version: %s\n", validationConfig.HermesVersion)

	return nil
}

// GetCurrentHermesVersion extracts the current Hermes version from go.mod.
func (g *GoModManager) GetCurrentHermesVersion() (string, error) {
	file, err := os.Open("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	replaceLine := regexp.MustCompile(`^replace\s+github\.com/probe-lab/hermes\s+=>\s+github\.com/ethpandaops/hermes\s+(v.+)$`)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := replaceLine.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	return "", fmt.Errorf("hermes replace directive not found in go.mod")
}

// ValidateForValidationMode checks if go.mod is correctly configured for the validation mode.
func (g *GoModManager) ValidateForValidationMode(validationMode config.ValidationMode) error {
	currentVersion, err := g.GetCurrentHermesVersion()
	if err != nil {
		return err
	}

	validationConfigs := config.GetValidationConfigs()
	expectedConfig := validationConfigs[validationMode]

	if !strings.Contains(currentVersion, expectedConfig.HermesVersion) {
		return fmt.Errorf("go.mod has wrong Hermes version for %s mode: got %s, expected %s",
			validationMode, currentVersion, expectedConfig.HermesVersion)
	}

	return nil
}
