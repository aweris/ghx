package actions

import (
	"dagger.io/dagger"
)

// ActionStage represents the stage of an action. It can be pre, main or post.
type ActionStage string

const (
	ActionStagePre  ActionStage = "pre"
	ActionStageMain ActionStage = "main"
	ActionStagePost ActionStage = "post"
)

// Action represents the metadata of an action. It contains all the information needed to run the action.
// The metadata is loaded from the action.yml | action.yaml file in the action repository.
//
// See more details at https://docs.github.com/en/actions/creating-actions/metadata-syntax-for-github-actions
type Action struct {
	Name        *String                 `yaml:"name"`        // Name is the name of the action.
	Author      *String                 `yaml:"author"`      // Author is the author of the action.
	Description *String                 `yaml:"description"` // Description is the description of the action.
	Inputs      map[string]ActionInput  `yaml:"inputs"`      // Inputs is a map of input names to their definitions.
	Outputs     map[string]ActionOutput `yaml:"outputs"`     // Outputs is a map of output names to their definitions.
	Runs        ActionRuns              `yaml:"runs"`        // Runs is the definition of how the action is run.
	Branding    Branding                `yaml:"branding"`    // Branding is the branding information for the action.
	Directory   *dagger.Directory       `yaml:"-"`           // Directory is the directory where source files for the action are located.
}

// ActionInput represents an input for a GitHub Action.
type ActionInput struct {
	Description        *String `yaml:"description"`        // Description is the description of the input.
	Default            *String `yaml:"default"`            // Default is the default value of the input.
	Required           *Bool   `yaml:"required"`           // Required is whether the input is required.
	DeprecationMessage *String `yaml:"deprecationMessage"` // DeprecationMessage is the message to display when the input is used.
}

// ActionOutput represents an output for a GitHub Action.
type ActionOutput struct {
	Description *String `yaml:"description"` // Description is the description of the output.
	Value       *String `yaml:"value"`       // Value is the value of the output. This is only used by composite actions.
}

// ActionRunsUsing represents the method used to run a GitHub Action.
type ActionRunsUsing string

var (
	ActionRunsUsingComposite ActionRunsUsing = "composite"
	ActionRunsUsingDocker    ActionRunsUsing = "docker"
	ActionRunsUsingNode16    ActionRunsUsing = "node16"
	ActionRunsUsingNode12    ActionRunsUsing = "node12"
)

// ActionRuns represents the definition of how a GitHub Action is run.
type ActionRuns struct {
	Using          ActionRunsUsing    `yaml:"using"`           // Using is the method used to run the action.
	Env            map[string]*String `yaml:"env"`             // Env is a map of environment variables to their values.
	Main           *String            `yaml:"main"`            // Main is the path to the main entrypoint for the action. This is only used by javascript actions.
	Pre            *String            `yaml:"pre"`             // Pre is the path to the pre entrypoint for the action. This is only used by javascript actions.
	PreIf          *String            `yaml:"pre-if"`          // PreIf is the condition for running the pre entrypoint. This is only used by javascript actions.
	Post           *String            `yaml:"post"`            // Post is the path to the post entrypoint for the action. This is only used by javascript actions.
	PostIf         *String            `yaml:"post-if"`         // PostIf is the condition for running the post entrypoint. This is only used by javascript actions.
	Steps          []Step             `yaml:"steps"`           // Steps is the list of steps to run for the action. This is only used by composite actions.
	Image          *String            `yaml:"image"`           // Image is the image used to run the action. This is only used by docker actions.
	PreEntrypoint  *String            `yaml:"pre-entrypoint"`  // PreEntrypoint is the pre-entrypoint used to run the action. This is only used by docker actions.
	Entrypoint     *String            `yaml:"entrypoint"`      // Entrypoint is the entrypoint used to run the action. This is only used by docker actions.
	PostEntrypoint *String            `yaml:"post-entrypoint"` // PostEntrypoint is the post-entrypoint used to run the action. This is only used by docker actions.
	Args           []*String          `yaml:"args"`            // Args is the arguments used to run the action. This is only used by docker actions.
}

// Branding represents the branding information for a GitHub Action.
type Branding struct {
	Color *String `yaml:"color"` // Color is the color of the action.
	Icon  *String `yaml:"icon"`  // Icon is the icon of the action.
}
