package state

import (
	"context"
	"fmt"
	"github.com/aweris/ghx/pkg/actions"
	"io"
	"strconv"

	"dagger.io/dagger"

	"github.com/aweris/ghx/pkg/config"
	"github.com/aweris/ghx/pkg/model"
)

var _ io.Closer = new(State)

type State struct {
	JobName   string                  `json:"job-name"`   // name of the job
	Actions   map[string]*ActionState `json:"actions"`    // map of action source to state of the action
	Env       map[string]string       `json:"env"`        // environment variables of the workflow and job
	StepOrder []string                `json:"step-order"` // order of the steps to make sure custom id is respected
	Steps     map[string]*StepState   `json:"steps"`      // map of step id to state of the step
}

// GetState returns the state of the runner from the state file
func GetState() (*State, error) {
	// Ensure initialize proper empty state
	s := &State{
		Actions: make(map[string]*ActionState),
		Env:     make(map[string]string),
		Steps:   make(map[string]*StepState),
	}

	if err := config.EnsureFile("state.json"); err != nil {
		return nil, err
	}

	if err := config.ReadJSONFile("state.json", s); err != nil {
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}

	return s, nil
}

// Close writes the state of the runner to the state file
func (s *State) Close() error {
	return config.WriteJSONFile("state.json", s)
}

// AddAction adds a new action to the state
func (s *State) AddAction(ctx context.Context, client *dagger.Client, source string) error {
	action, err := LoadAction(ctx, client, source)
	if err != nil {
		return err
	}

	s.Actions[source] = action

	return nil
}

// GetActionState returns the state of the action with the given source
func (s *State) GetActionState(source string) (*ActionState, bool) {
	as, ok := s.Actions[source]

	return as, ok
}

// AddWorkflowAndJob adds a new job to the state
func (s *State) AddWorkflowAndJob(workflow *model.Workflow, job *model.Job) error {
	s.JobName = job.Name

	// Add the workflow and job environment variables to the state while ensuring that the job environment variables
	// override the workflow environment variables

	for k, v := range workflow.Environment {
		s.Env[k] = v
	}

	for k, v := range job.Environment {
		s.Env[k] = v
	}

	// Add the steps of the job to the state
	for _, step := range job.Steps {
		if err := s.AddStep(step); err != nil {
			return err
		}
	}

	return nil
}

// AddStep adds a new step to the state
func (s *State) AddStep(step *model.Step) error {
	// if the step has no id, assign it a new one
	if step.ID == "" {
		step.ID = strconv.Itoa(len(s.Steps))
	}

	// if the step already exists, return an error
	if _, ok := s.Steps[step.ID]; ok {
		return fmt.Errorf("step with id %s already exists", step.ID)
	}

	// append the step to the step order
	s.StepOrder = append(s.StepOrder, step.ID)

	// add the step to the state
	s.Steps[step.ID] = NewStepState(step)

	return nil
}

// OverrideStep overrides the step with the given id with the given step
func (s *State) OverrideStep(id string, step *model.Step) error {
	// if the step already exists, return an error
	if _, ok := s.Steps[id]; !ok {
		return fmt.Errorf("step with id %s does not exist", id)
	}

	// override the step
	s.Steps[id] = NewStepState(step)

	return nil
}

// GetStepState returns the step for the given id. If the step is not loaded yet, it will return nil and false
func (s *State) GetStepState(id string) (*StepState, bool) {
	ss, ok := s.Steps[id]

	return ss, ok
}

// GetStepOrder returns the ids of the steps in the order they are added to the state which is the order
func (s *State) GetStepOrder() []string {
	return s.StepOrder
}

func (s *State) GetActionsContext() *actions.Context {
	// load the actions context from the environment variables
	ac := actions.NewContextFromEnv()

	ac.Env = s.Env

	for _, ss := range s.Steps {
		ac.Steps[ss.Step.ID] = ss.Result
	}

	return ac
}
