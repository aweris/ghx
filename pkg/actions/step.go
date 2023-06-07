package actions

// Steps represents a list of steps
type Steps []*Step

// Step represents a single task in a job context at GitHub Actions workflow
// For more information about workflows, see: https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idsteps
type Step struct {
	ID               *String            `yaml:"id,omitempty"`                // ID is the unique identifier of the step.
	If               *Bool              `yaml:"if,omitempty"`                // If is the conditional expression to run the step.
	Name             *String            `yaml:"name"`                        // Name is the name of the step.
	ContinueOnError  *Bool              `yaml:"continue-on-error,omitempty"` // ContinueOnError is whether to continue job execution when this step fails.
	TimeoutMinutes   *Float             `yaml:"timeout-minutes,omitempty"`   // TimeoutMinutes is the maximum number of minutes to run the step.
	Environment      map[string]*String `yaml:"env,omitempty"`               // Environment maps environment variable names to their values.
	Uses             *String            `yaml:"uses,omitempty"`              // Uses is the action to run for the step.
	With             map[string]*String `yaml:"with,omitempty"`              // With maps input names to their values for the step.
	Run              *String            `yaml:"run,omitempty"`               // Run is the command to run for the step.
	Shell            *String            `yaml:"shell,omitempty"`             // Shell is the shell to use for the step.
	WorkingDirectory *String            `yaml:"working-directory,omitempty"` // WorkingDirectory represents optional 'working-directory' field. Nil means nothing specified.
}

// StepType represents the type of step
type StepType string

const (
	StepTypeAction  StepType = "action"
	StepTypeRun     StepType = "run"
	StepTypeUnknown StepType = "unknown"
)

func (s *Step) Type() StepType {
	if s.Uses != nil {
		return StepTypeAction
	}

	if s.Run != nil {
		return StepTypeRun
	}

	return StepTypeUnknown
}

// StepStatus represents the status of a step
type StepStatus string

const (
	StepStatusSuccess StepStatus = "success"
	StepStatusFailure StepStatus = "failure"
	StepStatusSkipped StepStatus = "skipped"
)

// StepResult represents the result of a step
type StepResult struct {
	Outputs    map[string]string // Outputs maps output names to their values for the step.
	Conclusion StepStatus        // Conclusion is the conclusion of the step.
	Outcome    StepStatus        // Outcome is the outcome of the step.
}
