package pipeline

// PipelineDefinition defines a pipeline that can be loaded from YAML.
type PipelineDefinition struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Stages      []StageDefinition `yaml:"stages"`
}

// StageDefinition defines a single stage in a pipeline.
type StageDefinition struct {
	Name      string   `yaml:"name"`
	SkipIf    string   `yaml:"skip_if,omitempty"`    // Condition to skip
	LoopsWith []string `yaml:"loops_with,omitempty"` // Stages it loops with
	Pipeline  string   `yaml:"pipeline,omitempty"`   // Sub-pipeline to call
}
