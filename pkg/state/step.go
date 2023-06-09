package state

import "github.com/aweris/ghx/pkg/model"

type StepState struct {
	Step   *model.Step       // step metadata
	Result *model.StepResult // result of the step
	State  map[string]string // state of the step
}

// NewStepState creates a new step state with the given step
func NewStepState(step *model.Step) *StepState {
	return &StepState{
		Step:   step,
		Result: &model.StepResult{Outputs: make(map[string]string)},
		State:  make(map[string]string),
	}
}
