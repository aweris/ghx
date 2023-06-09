package step

import (
	"encoding/json"
	"fmt"
	"os"

	"dagger.io/dagger"

	"github.com/spf13/cobra"

	"github.com/aweris/ghx/pkg/model"
	statepkg "github.com/aweris/ghx/pkg/state"
)

// NewCommand  creates a new root command.
func NewCommand() *cobra.Command {
	// Flags for the Step command
	var (
		stepID          string
		stepName        string
		stepUses        string
		stepEnvironment map[string]string
		stepWith        map[string]string
		stepRun         string
		stepShell       string
		stepJSON        string
		stepOverride    bool
	)

	cmd := &cobra.Command{
		Use:   "step",
		Short: "Add new step to execute",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := dagger.Connect(cmd.Context(), dagger.WithLogOutput(os.Stdout))
			if err != nil {
				return err
			}
			defer client.Close()

			state, err := statepkg.GetState()
			if err != nil {
				return err
			}
			defer state.Close()

			var step *model.Step
			if stepJSON != "" {
				var step model.Step

				err := json.Unmarshal([]byte(stepJSON), &step)
				if err != nil {
					return fmt.Errorf("failed to parse step json: %w", err)
				}

			} else {
				if stepOverride && stepID == "" {
					return fmt.Errorf("step id must be provided to override")
				}

				step = &model.Step{
					ID:          stepID,
					Name:        stepName,
					Uses:        stepUses,
					Environment: stepEnvironment,
					With:        stepWith,
					Run:         stepRun,
					Shell:       stepShell,
				}
			}

			if stepOverride && step.ID == "" {
				return fmt.Errorf("step id must be provided to override")
			}

			if stepOverride {
				state.OverrideStep(step.ID, step)
			} else {
				state.AddStep(step)
			}

			return nil
		},
	}

	// Define flags for the Step command
	cmd.Flags().StringVar(&stepID, "id", "", "Unique identifier of the step")
	cmd.Flags().StringVar(&stepName, "name", "", "Name of the step")
	cmd.Flags().StringVar(&stepUses, "uses", "", "Action to run for the step")
	cmd.Flags().StringToStringVar(&stepEnvironment, "env", map[string]string{}, "Environment variable names and values")
	cmd.Flags().StringToStringVar(&stepWith, "with", map[string]string{}, "Input names and values for the step")
	cmd.Flags().StringVar(&stepRun, "run", "", "Command to run for the step")
	cmd.Flags().StringVar(&stepShell, "shell", "", "Shell to use for the step")
	cmd.Flags().StringVar(&stepJSON, "json", "", "JSON string to use for the step. This will override all other step values")
	cmd.Flags().BoolVar(&stepOverride, "override", false, "Override step if already exists")

	return cmd
}
