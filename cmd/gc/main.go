// Ground Control CLI - AI Agent Task Orchestration
package main

import (
	"fmt"
	"os"

	"github.com/mmariani/ground-control/internal/cmd"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/mmariani/ground-control/internal/tui"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	// Find data directory
	dataDir := cmd.GetDataDir()
	store := data.NewStore(dataDir)

	// Root command
	rootCmd := &cobra.Command{
		Use:   "gc",
		Short: "Ground Control - AI Agent Task Orchestration",
		Long: `Ground Control is a task management system where AI agents orchestrate work flow.
The Taskmaster agent manages priorities and routing, specialized agents execute work,
and humans make decisions at checkpoints.`,
		Version: version,
	}

	// Add commands
	rootCmd.AddCommand(cmd.NewTasksCmd(store))
	rootCmd.AddCommand(cmd.NewDumpCmd(store))
	rootCmd.AddCommand(cmd.NewCreateCmd(store))
	rootCmd.AddCommand(cmd.NewProcessCmd(store))
	rootCmd.AddCommand(cmd.NewOrcCmd(store))
	rootCmd.AddCommand(cmd.NewCompleteCmd(store))
	rootCmd.AddCommand(cmd.NewStandupCmd(store))
	rootCmd.AddCommand(cmd.NewStatusCmd(store))
	rootCmd.AddCommand(cmd.NewConsultCmd(store))
	rootCmd.AddCommand(cmd.NewDelegateCmd(store))
	rootCmd.AddCommand(cmd.NewHandoffCmd(store))
	rootCmd.AddCommand(cmd.NewSprintCmd(store))
	rootCmd.AddCommand(cmd.NewArtifactCmd(store))
	rootCmd.AddCommand(cmd.NewAppCmd(store))
	rootCmd.AddCommand(cmd.NewBrainstormCmd(store))
	rootCmd.AddCommand(cmd.NewSessionsCmd(store))
	rootCmd.AddCommand(cmd.NewHistoryCmd(store))
	rootCmd.AddCommand(cmd.NewSelfLearnCmd(store))
	rootCmd.AddCommand(cmd.NewIngestCmd(store))
	rootCmd.AddCommand(newTUICmd(store))

	// Silence Cobra's automatic error printing - we handle it ourselves
	rootCmd.SilenceErrors = true

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newTUICmd creates the tui command.
func newTUICmd(store *data.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long:  "Start the interactive terminal user interface for Ground Control.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(store)
		},
	}
}
