package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dagger.io/dagger"

	"github.com/aweris/ghx/pkg/actions"
	"github.com/aweris/ghx/pkg/model"
)

// State is the state of the runner
type State struct {
	ActionPathsBySource map[string]string        `json:"action-paths-by-source"` // map of action source to path of the action
	StepOrder           []string                 `json:"step-order"`             // order of the steps to make sure custom id is respected
	Children            map[string]*StepRunState `json:"children"`               // map of step id to state of the step
}

// LoadAction loads the action from the given source and stores it in the state
func (s *State) LoadAction(ctx context.Context, client *dagger.Client, source string) (string, *actions.Action, error) {
	action, err := actions.LoadActionFromSource(ctx, client, source)
	if err != nil {
		return "", nil, err
	}

	path := filepath.Join(ContainerRunnerPath, "actions", source)

	_, err = action.Directory.Export(ctx, path)
	if err != nil {
		return "", nil, err
	}

	s.ActionPathsBySource[source] = path

	return path, action, nil
}

// GetAction returns the action for the given source. If the action is not loaded yet, it will be loaded and stored in the state
func (s *State) GetAction(ctx context.Context, client *dagger.Client, source string) (string, *actions.Action, error) {
	path := s.ActionPathsBySource[source]
	if path == "" {
		return s.LoadAction(ctx, client, source)
	}

	action, err := actions.LoadActionFromSource(ctx, client, path)
	if err != nil {
		return "", nil, err
	}

	return path, action, nil
}

// AddNewStep adds a new step to the state
func (s *State) AddNewStep(step *model.Step, override bool) {
	// if the step is already in the state, we don't need to add it again
	if !override {
		s.StepOrder = append(s.StepOrder, step.ID)
	}

	s.Children[step.ID] = &StepRunState{
		Step:   step,
		Result: &model.StepResult{Outputs: make(map[string]string)},
		State:  make(map[string]string),
	}
}

// StepRunState is the state of a single step
type StepRunState struct {
	Step   *model.Step       `json:"step"`             // definition of the step
	Result *model.StepResult `json:"result"`           // result of the step
	State  map[string]string `json:"state"`            // state of the step
	Action *actions.Action   `json:"action,omitempty"` // action of the step if it is an action step
}

// GetStepDataDir returns the directory where the step related data is stored like state, logs, artifacts etc.
func (s *StepRunState) GetStepDataDir(stage actions.ActionStage) string {
	return filepath.Join(ContainerRunnerPath, "steps", s.Step.ID, string(stage))
}

// GetStepEnv returns the environment variables for the step to load in cmd exec
func (s *StepRunState) GetStepEnv(stage actions.ActionStage, ctx *actions.Context) ([]string, error) {
	// getting the current environment first
	env := os.Environ()

	// adding the state, input and environment variables to the environment. for duplicate keys, the last one wins
	// so getting the current environment first is important

	for k, v := range s.State {
		env = append(env, fmt.Sprintf("STATE_%s=%s", k, v))
	}

	if s.Step.Type() == model.StepTypeAction {
		// add inputs to the environment
		for k, v := range s.Step.With {
			env = append(env, fmt.Sprintf("INPUT_%s=%s", strings.ToUpper(k), v))
		}

		// add default values for inputs that are not defined in the step config
		for k, v := range s.Action.Inputs {
			if _, ok := s.Step.With[k]; ok {
				continue
			}

			if v.Default == nil {
				continue
			}

			val, err := v.Default.Eval(ctx)
			if err != nil {
				return nil, err
			}

			if val != "" {
				env = append(env, fmt.Sprintf("INPUT_%s=%s", strings.ToUpper(k), val))
			}
		}
	}

	for k, v := range s.Step.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// create directory and files for file commands and add them to the environment as well

	dir := filepath.Join(s.GetStepDataDir(stage), "file_commands")
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	env, err = addFileCommandToEnv(env, "GITHUB_ENV", filepath.Join(dir, "env"))
	if err != nil {
		return nil, err
	}

	env, err = addFileCommandToEnv(env, "GITHUB_PATH", filepath.Join(dir, "path"))
	if err != nil {
		return nil, err
	}

	env, err = addFileCommandToEnv(env, "GITHUB_STEP_SUMMARY", filepath.Join(dir, "step_summary"))
	if err != nil {
		return nil, err
	}

	env, err = addFileCommandToEnv(env, "GITHUB_ACTION_OUTPUT", filepath.Join(dir, "output"))
	if err != nil {
		return nil, err
	}

	return env, nil
}

func addFileCommandToEnv(env []string, key string, path string) ([]string, error) {
	if err := os.WriteFile(path, []byte{}, os.ModePerm); err != nil {
		return nil, err
	}

	env = append(env, fmt.Sprintf("%s=%s", key, path))

	return env, nil
}
