package templates

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

//go:embed *.html *.css
var templateFS embed.FS

// Manager handles template loading, parsing, and rendering.
type Manager struct {
	templates map[string]*template.Template
	logger    logrus.FieldLogger
}

// NewManager creates a new template manager.
func NewManager(logger logrus.FieldLogger) *Manager {
	return &Manager{
		templates: make(map[string]*template.Template),
		logger:    logger.WithField("component", "template_manager"),
	}
}

// LoadTemplates loads all templates from the embedded filesystem.
func (m *Manager) LoadTemplates() error {
	m.logger.Info("Loading HTML templates")

	// Load templates from embedded filesystem
	err := fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Create template name from filename
		templateName := strings.TrimSuffix(filepath.Base(path), ".html")

		// Parse template with helper functions
		tmpl, err := template.New(templateName).Funcs(m.getTemplateFuncs()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		m.templates[templateName] = tmpl
		m.logger.WithField("template", templateName).Debug("Loaded template")

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	m.logger.WithField("template_count", len(m.templates)).Info("Templates loaded successfully")

	return nil
}

// LoadTemplatesFromDir loads templates from a directory (for development/testing).
func (m *Manager) LoadTemplatesFromDir(dir string) error {
	m.logger.WithField("directory", dir).Info("Loading templates from directory")

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Create template name from filename
		templateName := strings.TrimSuffix(filepath.Base(path), ".html")

		// Parse template with helper functions
		tmpl, err := template.New(templateName).Funcs(m.getTemplateFuncs()).ParseFiles(path)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		m.templates[templateName] = tmpl
		m.logger.WithField("template", templateName).Debug("Loaded template from file")

		return nil
	})
}

// RenderReport renders the main report template with the given data.
func (m *Manager) RenderReport(data interface{}) (string, error) {
	return m.RenderTemplate("report", data)
}

// RenderTemplate renders a template with the given name and data.
func (m *Manager) RenderTemplate(templateName string, data interface{}) (string, error) {
	tmpl, exists := m.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var output strings.Builder
	if err := tmpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return output.String(), nil
}

// GetTemplate returns the raw template content for a given name.
func (m *Manager) GetTemplate(templateName string) (string, error) {
	if !strings.HasSuffix(templateName, ".html") {
		templateName += ".html"
	}

	content, err := templateFS.ReadFile(templateName)
	if err != nil {
		return "", fmt.Errorf("template %s not found: %w", templateName, err)
	}

	return string(content), nil
}

// GetAvailableTemplates returns a list of all available template names.
func (m *Manager) GetAvailableTemplates() []string {
	templates := make([]string, 0, len(m.templates))
	for name := range m.templates {
		templates = append(templates, name)
	}

	return templates
}

// getTemplateFuncs returns template helper functions.
func (m *Manager) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDuration": func(seconds float64) string {
			if seconds < 60 {
				return fmt.Sprintf("%.1fs", seconds)
			} else if seconds < 3600 {
				return fmt.Sprintf("%.1fm", seconds/60)
			} else {
				return fmt.Sprintf("%.1fh", seconds/3600)
			}
		},
		"formatPercent": func(value, total int) string {
			if total == 0 {
				return "0%"
			}

			percent := float64(value) / float64(total) * 100

			return fmt.Sprintf("%.1f%%", percent)
		},
		"formatScore": func(score float64) string {
			return fmt.Sprintf("%.3f", score)
		},
		"shortPeerID": func(peerID string) string {
			if len(peerID) <= 12 {
				return peerID
			}

			return peerID[:12]
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"gt": func(a, b interface{}) bool {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va > vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va > vb
				}
			}

			return false
		},
		"lt": func(a, b interface{}) bool {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va < vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va < vb
				}
			}

			return false
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"printf": fmt.Sprintf,
		"contains": func(s, substr string) bool {
			return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
		},
		"title": strings.Title, //nolint:staticcheck // ok.
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		//nolint:predeclared // ok.
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
	}
}
