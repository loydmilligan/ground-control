package artifact

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadTemplate loads a template by name from the templates directory.
func LoadTemplate(dataDir, name string) (*ArtifactTemplate, error) {
	templatesDir := filepath.Join(dataDir, "templates")
	path := filepath.Join(templatesDir, name+".template.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", name, err)
	}

	var template ArtifactTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", name, err)
	}

	return &template, nil
}

// ListTemplates returns all available template names.
func ListTemplates(dataDir string) ([]string, error) {
	templatesDir := filepath.Join(dataDir, "templates")

	// Check if templates directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("reading templates directory: %w", err)
	}

	var templates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check for .template.yaml suffix
		if len(name) > 14 && name[len(name)-14:] == ".template.yaml" {
			// Extract template name (remove .template.yaml)
			templates = append(templates, name[:len(name)-14])
		}
	}

	return templates, nil
}
