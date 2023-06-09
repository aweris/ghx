package job

import (
	"fmt"
	"os"

	"dagger.io/dagger"

	"github.com/spf13/cobra"

	"github.com/aweris/ghx/pkg/repository"
	statepkg "github.com/aweris/ghx/pkg/state"
)

// NewCommand  creates a new root command.
func NewCommand() *cobra.Command {
	// Flags for the Job command
	var (
		workflowName string
		jobName      string
	)

	cmd := &cobra.Command{
		Use:   "job",
		Short: "Add job to execute",
		Long:  "Sets workflow and job environment variables and configures to all steps in the job.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if workflowName == "" || jobName == "" {
				return fmt.Errorf("workflow and job name must be provided")
			}

			var opts []dagger.ClientOpt

			if os.Getenv("RUNNER_DEBUG") == "1" {
				opts = append(opts, dagger.WithLogOutput(os.Stdout))
			}

			client, err := dagger.Connect(cmd.Context(), opts...)
			if err != nil {
				return err
			}

			// TODO: temporary solution to load workflow from current directory. `gh` is missing in runner image
			workflows, err := repository.LoadWorkflows(cmd.Context(), client, ".")
			if err != nil {
				return err
			}

			workflow, ok := workflows[workflowName]
			if !ok {
				return fmt.Errorf("workflow %s not found", workflowName)
			}

			job, ok := workflow.Jobs[jobName]
			if !ok {
				return fmt.Errorf("job %s/%s not found", workflowName, jobName)
			}

			if job.Name == "" {
				job.Name = jobName
			}

			state, err := statepkg.GetState()
			if err != nil {
				return err
			}
			defer state.Close()

			err = state.AddWorkflowAndJob(workflow, job)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// Define flags for the Step command
	cmd.Flags().StringVar(&workflowName, "workflow", "", "Name of the workflow. If workflow doesn't have name, than it must be relative path to the workflow file")
	cmd.Flags().StringVar(&jobName, "job", "", "Name of the job")

	return cmd
}
