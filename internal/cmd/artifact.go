package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mmariani/ground-control/internal/artifact"
	"github.com/mmariani/ground-control/internal/data"
	"github.com/spf13/cobra"
)

// NewArtifactCmd creates the artifact command.
func NewArtifactCmd(store *data.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Manage artifacts from templates",
		Long: `Generate documents from templates with interactive variable population.

Available commands:
  list      List available templates
  generate  Generate an artifact from a template`,
	}

	cmd.AddCommand(newArtifactListCmd(store))
	cmd.AddCommand(newArtifactGenerateCmd(store))

	return cmd
}

// newArtifactListCmd creates the artifact list subcommand.
func newArtifactListCmd(store *data.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			templates, err := artifact.ListTemplates(store.GetDataDir())
			if err != nil {
				return fmt.Errorf("listing templates: %w", err)
			}

			if len(templates) == 0 {
				fmt.Println(dimStyle.Render("No templates found."))
				return nil
			}

			fmt.Println(headerStyle.Render("Available Templates"))
			fmt.Println()
			for _, name := range templates {
				// Load template to get description
				tmpl, err := artifact.LoadTemplate(store.GetDataDir(), name)
				if err != nil {
					fmt.Printf("  %s %s\n", name, dimStyle.Render("(error loading)"))
					continue
				}
				fmt.Printf("  %s - %s\n", taskTitleStyle.Render(name), tmpl.Description)
			}
			fmt.Println()
			fmt.Println(dimStyle.Render("Use 'gc artifact generate <template>' to create an artifact."))

			return nil
		},
	}
}

// newArtifactGenerateCmd creates the artifact generate subcommand.
func newArtifactGenerateCmd(store *data.Store) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "generate <template>",
		Short: "Generate an artifact from a template",
		Long: `Interactively generate an artifact by populating template variables.

Example:
  gc artifact generate project_plan
  gc artifact generate project_plan --output plan.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]

			// Load template
			tmpl, err := artifact.LoadTemplate(store.GetDataDir(), templateName)
			if err != nil {
				return fmt.Errorf("loading template: %w", err)
			}

			fmt.Println(headerStyle.Render(fmt.Sprintf("Generate: %s", tmpl.Name)))
			fmt.Println(dimStyle.Render(tmpl.Description))
			fmt.Println()

			// Check for required input artifacts
			if len(tmpl.Requires) > 0 {
				fmt.Println(dimStyle.Render("Required inputs:"))
				for _, req := range tmpl.Requires {
					fmt.Printf("  - %s\n", req)
				}
				fmt.Println()
			}

			// Collect variables interactively
			variables := make(map[string]string)
			scanner := bufio.NewScanner(os.Stdin)

			for _, v := range tmpl.Variables {
				// Show guidance if available
				if guidance, ok := tmpl.Guidance[v.Name]; ok {
					fmt.Println(dimStyle.Render(guidance))
				}

				// Show variable info
				requiredMark := ""
				if v.Required {
					requiredMark = highStyle.Render(" *")
				}
				fmt.Printf("%s%s [%s]: ", taskTitleStyle.Render(v.Name), requiredMark, v.Description)

				// Handle choice type
				if v.Type == "choice" && len(v.Options) > 0 {
					fmt.Println()
					for i, opt := range v.Options {
						fmt.Printf("  %d. %s\n", i+1, opt)
					}
					fmt.Print(dimStyle.Render("Choose (number): "))
				}

				// Read input
				scanner.Scan()
				value := strings.TrimSpace(scanner.Text())

				// Use default if empty
				if value == "" && v.Default != "" {
					value = v.Default
				}

				// Validate required
				if v.Required && value == "" {
					return fmt.Errorf("required variable missing: %s", v.Name)
				}

				// Handle choice type
				if v.Type == "choice" && value != "" {
					// Convert number to option
					var choiceIdx int
					fmt.Sscanf(value, "%d", &choiceIdx)
					if choiceIdx > 0 && choiceIdx <= len(v.Options) {
						value = v.Options[choiceIdx-1]
					}
				}

				variables[v.Name] = value
			}

			// Generate artifact
			result, err := artifact.GenerateArtifact(tmpl, variables)
			if err != nil {
				return fmt.Errorf("generating artifact: %w", err)
			}

			fmt.Println()
			fmt.Println(headerStyle.Render("Generated Artifact"))
			fmt.Println()

			// Output to file or stdout
			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
				fmt.Println(lowStyle.Render(fmt.Sprintf("Written to: %s", outputPath)))
			} else {
				fmt.Println(result)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: stdout)")

	return cmd
}
