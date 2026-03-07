package artifact

import (
	"fmt"
	"strings"
)

// GenerateArtifact populates a template with the provided variables.
func GenerateArtifact(template *ArtifactTemplate, variables map[string]string) (string, error) {
	// Validate required variables
	for _, v := range template.Variables {
		if v.Required {
			if val, ok := variables[v.Name]; !ok || val == "" {
				return "", fmt.Errorf("required variable missing: %s", v.Name)
			}
		}
	}

	// Replace placeholders
	result := template.Document
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Check for unreplaced placeholders
	if strings.Contains(result, "{{") && strings.Contains(result, "}}") {
		start := strings.Index(result, "{{")
		end := strings.Index(result[start:], "}}") + start + 2
		unreplaced := result[start:end]
		return "", fmt.Errorf("unreplaced placeholder found: %s", unreplaced)
	}

	return result, nil
}

// GetVariableDefaults returns a map of variable names to their default values.
func GetVariableDefaults(template *ArtifactTemplate) map[string]string {
	defaults := make(map[string]string)
	for _, v := range template.Variables {
		if v.Default != "" {
			defaults[v.Name] = v.Default
		}
	}
	return defaults
}
