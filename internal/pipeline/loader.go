package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadPipeline loads a pipeline definition by name.
// It looks for the file in config/pipelines/{name}.yaml.
// If the file doesn't exist, it returns a built-in definition.
func LoadPipeline(name string) (*PipelineDefinition, error) {
	// Try to load from config/pipelines/{name}.yaml
	configPath := filepath.Join("config", "pipelines", name+".yaml")

	if _, err := os.Stat(configPath); err == nil {
		// File exists, load it
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("reading pipeline file %s: %w", configPath, err)
		}

		var def PipelineDefinition
		if err := yaml.Unmarshal(data, &def); err != nil {
			return nil, fmt.Errorf("parsing pipeline file %s: %w", configPath, err)
		}

		return &def, nil
	}

	// File doesn't exist, fall back to built-in definitions
	return getBuiltInPipeline(name)
}

// LoadAllPipelines loads all pipeline definitions from config/pipelines/.
// It also includes built-in pipelines that don't have file overrides.
func LoadAllPipelines() (map[string]*PipelineDefinition, error) {
	pipelines := make(map[string]*PipelineDefinition)

	// Load built-in pipelines first
	builtIns := []string{"coding", "simple", "research", "ai-planning", "human-input"}
	for _, name := range builtIns {
		def, err := getBuiltInPipeline(name)
		if err != nil {
			return nil, fmt.Errorf("loading built-in pipeline %s: %w", name, err)
		}
		pipelines[name] = def
	}

	// Load custom pipelines from config/pipelines/ (override built-ins if present)
	configDir := filepath.Join("config", "pipelines")
	if entries, err := os.ReadDir(configDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}

			name := entry.Name()[:len(entry.Name())-5] // Remove .yaml extension
			def, err := LoadPipeline(name)
			if err != nil {
				return nil, fmt.Errorf("loading pipeline %s: %w", name, err)
			}
			pipelines[name] = def
		}
	}

	return pipelines, nil
}

// getBuiltInPipeline returns a built-in pipeline definition.
func getBuiltInPipeline(name string) (*PipelineDefinition, error) {
	switch name {
	case "coding":
		return &PipelineDefinition{
			Name:        "coding",
			Description: "Standard code implementation pipeline",
			Stages: []StageDefinition{
				{Name: "sanity"},
				{
					Name:      "coder",
					LoopsWith: []string{"reviewer", "tester"},
				},
				{Name: "reviewer"},
				{
					Name:   "tester",
					SkipIf: "no_tests",
				},
				{Name: "commit"},
			},
		}, nil

	case "simple":
		return &PipelineDefinition{
			Name:        "simple",
			Description: "Simple task pipeline without review",
			Stages: []StageDefinition{
				{Name: "sanity"},
				{Name: "coder"},
				{Name: "commit"},
			},
		}, nil

	case "research":
		return &PipelineDefinition{
			Name:        "research",
			Description: "Research task pipeline",
			Stages: []StageDefinition{
				{Name: "sanity"},
				{Name: "researcher"},
				{Name: "summary"},
			},
		}, nil

	case "ai-planning":
		return &PipelineDefinition{
			Name:        "ai-planning",
			Description: "AI planning task pipeline",
			Stages: []StageDefinition{
				{Name: "sanity"},
				{Name: "planner"},
				{Name: "human-review"},
			},
		}, nil

	case "human-input":
		return &PipelineDefinition{
			Name:        "human-input",
			Description: "Human input task pipeline",
			Stages: []StageDefinition{
				{Name: "notify"},
				{Name: "wait-human"},
				{Name: "capture-response"},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown built-in pipeline: %s", name)
	}
}
