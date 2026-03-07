// Package artifact handles template-based artifact generation.
package artifact

// ArtifactTemplate defines a template for generating documents.
type ArtifactTemplate struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Document    string                 `yaml:"document"`
	Variables   []TemplateVariable     `yaml:"variables"`
	Guidance    map[string]string      `yaml:"guidance"`
	Requires    []string               `yaml:"requires"`
}

// TemplateVariable defines a variable that can be populated in a template.
type TemplateVariable struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"` // string, list, choice
	Description string   `yaml:"description"`
	Required    bool     `yaml:"required"`
	Options     []string `yaml:"options,omitempty"`
	Default     string   `yaml:"default,omitempty"`
}
