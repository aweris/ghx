package with

import (
	"github.com/spf13/cobra"

	"github.com/aweris/ghx/cmd/with/job"
	"github.com/aweris/ghx/cmd/with/step"
)

// NewCommand  creates a new root command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "with",
		Short: "Adds new configuration to execute",
	}

	cmd.AddCommand(step.NewCommand())
	cmd.AddCommand(job.NewCommand())

	return cmd
}
