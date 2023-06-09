package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"dagger.io/dagger"

	"github.com/aweris/ghx/internal/log"
	"github.com/aweris/ghx/pkg/config"
	"github.com/aweris/ghx/pkg/model"
	statepkg "github.com/aweris/ghx/pkg/state"
)

type ExecStepStatus string

const (
	StatusSucceeded ExecStepStatus = "succeeded"
	StatusSkipped   ExecStepStatus = "skipped"
	StatusFailed    ExecStepStatus = "failed"
)

// Runner is the interface for the runner
type Runner interface {
	// Execute executes the steps configured previously with WithStep(), WithJob()
	Execute(ctx context.Context) error
}

var _ Runner = new(runner)

type runner struct {
	client *dagger.Client
	state  *statepkg.State
	logger *log.Logger
}

// New creates a new runner
func New(client *dagger.Client, state *statepkg.State) (Runner, error) {
	return &runner{client: client, state: state, logger: log.NewLogger()}, nil
}

// Execute executes the steps configured previously with WithStep()
func (r *runner) Execute(ctx context.Context) error {
	if err := r.setupJob(ctx); err != nil {
		return err
	}

	// TODO: check conditionals like continue-on-error, skip etc...

	// ids of the steps to run with execution order
	ids := r.state.GetStepOrder()

	// Run stages
	for _, stepID := range ids {
		// TODO: also add conditional check here with expression evaluation
		ss, _ := r.state.GetStepState(stepID)

		// skip run steps since they don't have pre runs
		if ss.Step.Type() == model.StepTypeRun {
			continue
		}

		as, _ := r.state.GetActionState(ss.Step.Uses)

		if as.Metadata.Runs.Pre == "" {
			continue
		}

		result, _ := r.execStep(ctx, ss, model.ActionStagePre)
		if result == StatusFailed {
			return fmt.Errorf("step %s failed at pre stage", stepID)
		}
	}

	for _, stepID := range ids {
		// TODO: also add conditional check here with expression evaluation
		ss, _ := r.state.GetStepState(stepID)

		result, _ := r.execStep(ctx, ss, model.ActionStageMain)
		if result == StatusFailed {
			return fmt.Errorf("step %s failed at main stage", stepID)
		}
	}

	for _, stepID := range ids {
		// TODO: also add conditional check here with expression evaluation
		ss, _ := r.state.GetStepState(stepID)

		// skip run steps since they don't have pre runs
		if ss.Step.Type() == model.StepTypeRun {
			continue
		}

		as, _ := r.state.GetActionState(ss.Step.Uses)

		if as.Metadata.Runs.Post == "" {
			continue
		}

		result, _ := r.execStep(ctx, ss, model.ActionStagePost)
		if result == StatusFailed {
			return fmt.Errorf("step %s failed at post stage", stepID)
		}
	}

	return nil
}

// setupJob performs the `Set up job` step from the Github Actions workflow run to prepare the job environment
func (r *runner) setupJob(ctx context.Context) error {
	r.logger.Info("Set up job")
	r.logger.StartGroup()
	defer r.logger.EndGroup()

	// ids of the steps to run with execution order
	ids := r.state.GetStepOrder()

	// Setup steps
	for _, stepID := range ids {
		ss, _ := r.state.GetStepState(stepID)

		switch ss.Step.Type() {
		case model.StepTypeAction:
			err := r.state.AddAction(ctx, r.client, ss.Step.Uses)
			if err != nil {
				return err
			}

			r.logger.Info(fmt.Sprintf("Download action repository '%s'", ss.Step.Uses))
		case model.StepTypeRun:
			path := filepath.Join("scripts", ss.Step.ID, "run.sh")
			content := []byte(fmt.Sprintf("#!/bin/bash\n%s", ss.Step.Run))

			err := config.WriteFile(path, content, 0755)
			if err != nil {
				return err
			}

			// make it debug level because it's not really important and it's visible in Github Actions logs
			r.logger.Debug(fmt.Sprintf("Write script to '%s' for step '%s'", path, ss.Step.ID))
		}
	}

	r.logger.Info(fmt.Sprintf("Complete job name: %s", r.state.JobName))

	return nil
}

func (r *runner) execStep(ctx context.Context, ss *statepkg.StepState, stage model.ActionStage) (ExecStepStatus, error) {
	r.logger.Info(ss.Step.LogMessage(stage))
	r.logger.StartGroup()
	defer r.logger.EndGroup()

	switch ss.Step.Type() {
	case model.StepTypeAction:
		return r.execStepAction(ctx, ss, stage)
	case model.StepTypeRun:
		return r.execStepRun(ctx, ss, stage)
	default:
		return StatusFailed, fmt.Errorf("not supported step type %s", ss.Step.Type())
	}
}

func (r *runner) execStepAction(ctx context.Context, ss *statepkg.StepState, stage model.ActionStage) (ExecStepStatus, error) {
	as, ok := r.state.GetActionState(ss.Step.Uses)
	if !ok {
		return StatusFailed, fmt.Errorf("action '%s' not found", ss.Step.Uses)
	}

	var runs string

	switch stage {
	case model.ActionStagePre:
		runs = as.Metadata.Runs.Pre
	case model.ActionStageMain:
		runs = as.Metadata.Runs.Main
	case model.ActionStagePost:
		runs = as.Metadata.Runs.Post
	default:
		return StatusFailed, fmt.Errorf("not supported stage %s", stage)
	}

	err := r.execCmd(ctx, ss, stage, []string{"node", fmt.Sprintf("%s/%s", as.Path, runs)})
	if err != nil {
		ss.Result.Conclusion = model.StepStatusFailure
		ss.Result.Outcome = model.StepStatusFailure

		return StatusFailed, err
	}

	ss.Result.Conclusion = model.StepStatusSuccess
	ss.Result.Outcome = model.StepStatusSuccess

	return StatusSucceeded, nil
}

func (r *runner) execStepRun(ctx context.Context, ss *statepkg.StepState, stage model.ActionStage) (ExecStepStatus, error) {
	// path of the run.sh to execute
	path := config.GetPath("scripts", ss.Step.ID, "run.sh")

	// execute the script
	err := r.execCmd(ctx, ss, stage, []string{"bash", "--noprofile", "--norc", "-e", "-o", "pipefail", path})
	if err != nil {
		ss.Result.Conclusion = model.StepStatusFailure
		ss.Result.Outcome = model.StepStatusFailure

		return StatusFailed, err
	}

	ss.Result.Conclusion = model.StepStatusSuccess
	ss.Result.Outcome = model.StepStatusSuccess
	return StatusSucceeded, nil
}

func (r *runner) execCmd(ctx context.Context, ss *statepkg.StepState, stage model.ActionStage, args []string) error {
	// get the step env
	env, err := getStepEnv(r.state, ss, stage)
	if err != nil {
		return err
	}

	//nolint:gosec // (G204) this is a command runner, we need to run arbitrary commands.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	commandsRaw := bytes.NewBuffer(nil)

	cmd.Stderr = io.MultiWriter(stderr, os.Stderr)
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	var commands []*model.Command

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			output := scanner.Text()

			// write to stdout as it is so we can keep original formatting
			stdout.WriteString(output)
			stdout.WriteString("\n") // scanner strips newlines

			isCommand, command := model.ParseCommand(output)

			// print the output if it is a regular output
			if !isCommand {
				r.logger.Info(output)

				continue
			}

			// add the command to the list of commands to keep it as artifact
			commands = append(commands, command)

			// write to commands raw so we can keep original formatting
			commandsRaw.WriteString(output)
			commandsRaw.WriteString("\n")

			// process the command
			if err := processGithubWorkflowCommands(command, ss, r.logger); err != nil {
				fmt.Println(err)
			}
		}
	}()

	cmdErr := cmd.Wait()

	if data := stdout.Bytes(); len(data) > 0 {
		config.WriteFile(filepath.Join("steps", ss.Step.ID, "logs", string(stage), "stdout.log"), data, 0600)
	}

	if data := stderr.Bytes(); len(data) > 0 {
		config.WriteFile(filepath.Join("steps", ss.Step.ID, "logs", string(stage), "stderr.log"), data, 0600)
	}

	if len(commands) > 0 {
		config.WriteJsonFile(filepath.Join("steps", ss.Step.ID, "logs", string(stage), "workflow_commands.json"), commands)
	}

	if data := commandsRaw.Bytes(); len(data) > 0 {
		config.WriteFile(filepath.Join("steps", ss.Step.ID, "logs", string(stage), "workflow_commands.log"), data, 0600)
	}

	if cmdErr != nil {
		return cmdErr
	}

	// process commands at the end of the command
	return processFileCommands(ss, stage)
}
