package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dagger.io/dagger"

	"github.com/aweris/ghx/internal/log"
	"github.com/aweris/ghx/pkg/model"
)

const (
	ContainerRunnerPath  = "/home/runner/_temp/ghx" // path where the state and artifacts are stored in the container
	ContainerRunnerState = "state.json"             // name of the state file
)

type ExecStepStatus string

const (
	StatusSucceeded ExecStepStatus = "succeeded"
	StatusSkipped   ExecStepStatus = "skipped"
	StatusFailed    ExecStepStatus = "failed"
)

// Runner is the interface for the runner
type Runner interface {
	io.Closer

	// WithStep adds a step to the runner state to be executed with Execute(), if override is true, it will override
	// the step with the same id
	WithStep(step *model.Step, override bool)

	// WithJob adds a job to the runner state to be executed with Execute()
	WithJob(workflow *model.Workflow, job *model.Job)

	// Execute executes the steps configured previously with WithStep(), WithJob()
	Execute(ctx context.Context) error
}

var _ Runner = new(runner)

type runner struct {
	client *dagger.Client
	state  *State
	logger *log.Logger
}

// New creates a new runner
func New(client *dagger.Client) (Runner, error) {
	if err := ensureDir(ContainerRunnerPath); err != nil {
		return nil, err
	}

	state, err := readRunnerState()
	if err != nil {
		return nil, err
	}

	return &runner{client: client, state: state, logger: log.NewLogger()}, nil
}

func (r *runner) WithJob(workflow *model.Workflow, job *model.Job) {
	for k, v := range workflow.Environment {
		os.Setenv(k, v)
	}

	for k, v := range job.Environment {
		os.Setenv(k, v)
	}

	for _, step := range job.Steps {
		r.WithStep(step, false)
	}
}

// WithStep adds a step to the steps configuration file to be executed by the runner. If override is true, it will override
// the step with the same id. Method assumes override and step id validated by the caller
func (r *runner) WithStep(step *model.Step, override bool) {
	//TODO: add validation for step type
	if step.ID == "" {
		step.ID = fmt.Sprintf("%d", len(r.state.StepOrder))
	}

	r.state.AddNewStep(step, override)

	switch step.Type() {
	case model.StepTypeAction:
		r.logger.Infof("New step added", "type", model.StepTypeAction, "uses", step.Uses, "index", "step.ID")
	case model.StepTypeRun:
		r.logger.Infof("New step added", "type", model.StepTypeRun, "index", "step.ID")
	}
}

// Execute executes the steps configured previously with WithStep()
func (r *runner) Execute(ctx context.Context) error {
	if err := r.execSetup(ctx); err != nil {
		return err
	}

	// TODO: check conditionals like continue-on-error, skip etc...

	// Run stages
	for _, key := range r.state.StepOrder {
		result, _ := r.execStep(ctx, model.ActionStagePre, key)
		if result == StatusFailed {
			return fmt.Errorf("step %s failed at pre stage", key)
		}
	}

	for _, key := range r.state.StepOrder {
		result, _ := r.execStep(ctx, model.ActionStageMain, key)
		if result == StatusFailed {
			return fmt.Errorf("step %s failed at main stage", key)
		}
	}

	for _, key := range r.state.StepOrder {
		result, _ := r.execStep(ctx, model.ActionStagePost, key)
		if result == StatusFailed {
			return fmt.Errorf("step %s failed at post stage", key)
		}
	}

	return nil
}

// Close persist latest runner state to disk
func (r *runner) Close() error {
	return writeRunnerState(r.state)
}

func (r *runner) execSetup(ctx context.Context) error {
	for _, stepID := range r.state.StepOrder {
		srs := r.state.Children[stepID]
		if srs.Step.Uses == "" {
			continue
		}

		_, _, err := r.state.LoadAction(ctx, r.client, srs.Step.Uses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *runner) execStep(ctx context.Context, stage model.ActionStage, stepID string) (ExecStepStatus, error) {
	// get step run state
	srs := r.state.Children[stepID]
	if srs == nil {
		return StatusFailed, fmt.Errorf("step %s not found", stepID)
	}

	switch srs.Step.Type() {
	case model.StepTypeAction:
		return r.execStepAction(ctx, stage, srs)
	case model.StepTypeRun:
		return r.execStepRun(ctx, stage, srs)
	default:
		return StatusFailed, fmt.Errorf("not supported step type %s", srs.Step.Type())
	}
}

func (r *runner) execStepAction(ctx context.Context, stage model.ActionStage, srs *StepRunState) (ExecStepStatus, error) {
	path, action, err := r.state.GetAction(ctx, r.client, srs.Step.Uses)
	if err != nil {
		return StatusFailed, err
	}

	// add action to step run state
	srs.Action = action

	var runs string

	switch stage {
	case model.ActionStagePre:
		runs = action.Runs.Pre
	case model.ActionStageMain:
		runs = action.Runs.Main
	case model.ActionStagePost:
		runs = action.Runs.Post
	default:
		return StatusFailed, fmt.Errorf("not supported stage %s", stage)
	}

	// if runs is empty for pre or post, this is a no-op step
	if runs == "" && stage != model.ActionStageMain {
		return StatusSkipped, nil
	}

	// if runs is empty for main, this is a failure
	if runs == "" && stage == model.ActionStageMain {
		srs.Result.Conclusion = model.StepStatusFailure
		srs.Result.Outcome = model.StepStatusFailure

		return StatusFailed, fmt.Errorf("no runs for step %s", srs.Step.ID)
	}

	if stage == model.ActionStageMain {
		r.logger.Info(fmt.Sprintf("Run %s", srs.Step.Uses))
	} else {
		// capitalize first letter of stage to keep consistency with github actions
		str := string(stage)
		r.logger.Info(fmt.Sprintf("%s run %s", strings.ToUpper(str[:1])+str[1:], srs.Step.Uses))
	}

	// start new log group to separate action logs
	r.logger.StartGroup()

	err = execCommand(ctx, stage, []string{"node", fmt.Sprintf("%s/%s", path, runs)}, srs, r.logger)
	if err != nil {
		srs.Result.Conclusion = model.StepStatusFailure
		srs.Result.Outcome = model.StepStatusFailure
		return StatusFailed, err
	}

	// end log group
	r.logger.EndGroup()

	srs.Result.Conclusion = model.StepStatusSuccess
	srs.Result.Outcome = model.StepStatusSuccess

	return StatusSucceeded, nil
}

func (r *runner) execStepRun(ctx context.Context, stage model.ActionStage, srs *StepRunState) (ExecStepStatus, error) {
	// step run only happens for main stage
	if stage != model.ActionStageMain {
		return StatusSkipped, nil
	}

	// script directory
	path := filepath.Join(ContainerRunnerPath, "scripts", srs.Step.ID)

	// ensure the directory exists
	//nolint:gosec // it's ok to create a directory with 0755 permissions. This directory contains scripts that will be
	// executed by the user
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return StatusFailed, err
	}

	// write the script to disk
	//nolint:gosec // This is a script, it's supposed to be executable by the user
	err := os.WriteFile(path, []byte(fmt.Sprintf("#!/bin/bash\n%s", srs.Step.Run)), 0755)
	if err != nil {
		return StatusFailed, err
	}

	// execute the script
	err = execCommand(ctx, stage, []string{"bash", "--noprofile", "--norc", "-e", "-o", "pipefail", path}, srs, r.logger)
	if err != nil {
		srs.Result.Conclusion = model.StepStatusFailure
		srs.Result.Outcome = model.StepStatusFailure
		return StatusFailed, err
	}

	srs.Result.Conclusion = model.StepStatusSuccess
	srs.Result.Outcome = model.StepStatusSuccess
	return StatusSucceeded, nil
}
