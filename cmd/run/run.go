package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"dagger.io/dagger"

	"github.com/spf13/cobra"

	runnerpkg "github.com/aweris/ghx/pkg/runner"
)

// NewCommand  creates a new root command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs all configured steps",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := run(cmd.Context())

			// dagger and buildkit doesn't allow running commands if container is failed.
			//
			// failure of `run` is operational, so we need to return 0 exit code to dagger.
			// To workaround this, we write exit code to a file end of the command and finish the command with 0 exit code.
			//
			// TODO: we can remove this workaround after https://github.com/dagger/dagger/issues/5124 is fixed probably.
			exitCode := 0

			if err != nil {
				exitCode = 1
				fmt.Printf("Error executing command: %v\n", err)
			}

			path := filepath.Join(runnerpkg.ContainerRunnerPath, "exit-code")

			fmt.Printf("Writing exit code %d to %s\n", exitCode, path)

			return os.WriteFile(path, []byte(fmt.Sprintf("%d", exitCode)), 0600)
		},
	}

	return cmd
}

func run(ctx context.Context) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}
	defer client.Close()

	runner, err := runnerpkg.New(client)
	if err != nil {
		return err
	}
	defer runner.Close()

	return runner.Execute(ctx)
}
