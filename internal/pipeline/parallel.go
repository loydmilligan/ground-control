package pipeline

// ParallelConfig holds parallelization configuration.
type ParallelConfig struct {
	MaxConcurrentAgents int  `yaml:"max_concurrent_agents"`
	MaxConcurrentTasks  int  `yaml:"max_concurrent_tasks"`
	Enabled             bool `yaml:"enabled"`
}

// DefaultParallelConfig returns the default parallelization configuration.
func DefaultParallelConfig() *ParallelConfig {
	return &ParallelConfig{
		MaxConcurrentAgents: 3,
		MaxConcurrentTasks:  5,
		Enabled:             true,
	}
}
