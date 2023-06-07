package actions

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/aweris/ghx/pkg/expression"
)

var _ expression.VariableProvider = new(Context)

// TODO: add jobs and inputs context. Currently skipped because ghx not support re-usable workflows or workflow dispatch event

type Context struct {
	Github   *GithubContext         // Github context
	Env      map[string]string      // Environment variables from the workflow, job, and steps contexts
	Vars     map[string]string      // Variables context contains custom configuration variables set at the organization, repository, and environment levels.
	Job      *JobContext            // Job context
	Steps    map[string]*StepResult // Steps context to access the outputs of previous steps
	Runner   *RunnerContext         // Runner context
	Secrets  map[string]string      // Secrets context
	Strategy *StrategyContext       // Strategy context
	Matrix   map[string]string      // Matrix context
	Needs    map[string]string      // Needs context
}

func NewContext() (*Context, error) {
	var (
		err           error
		retentionDays int
	)

	if val := os.Getenv("GITHUB_RETENTION_DAYS"); val != "" {
		retentionDays, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("GITHUB_RETENTION_DAYS %s is not a valid integer", val)
		}
	}

	//nolint:goconst // keeping "true" here as string makes it easier to read and understand. No need to create a const.
	return &Context{
		Github: &GithubContext{
			CI:                os.Getenv("CI") == "true",
			Actions:           os.Getenv("GITHUB_ACTIONS") == "true",
			Action:            os.Getenv("GITHUB_ACTION"),
			ActionPath:        os.Getenv("GITHUB_ACTION_PATH"),
			ActionRepository:  os.Getenv("GITHUB_ACTION_REPOSITORY"),
			Actor:             os.Getenv("GITHUB_ACTOR"),
			ActorID:           os.Getenv("GITHUB_ACTOR_ID"),
			ApiURL:            os.Getenv("GITHUB_API_URL"),
			BaseRef:           os.Getenv("GITHUB_BASE_REF"),
			Env:               os.Getenv("GITHUB_ENV"),
			EventName:         os.Getenv("GITHUB_EVENT_NAME"),
			EventPath:         os.Getenv("GITHUB_EVENT_PATH"),
			GraphqlURL:        os.Getenv("GITHUB_GRAPHQL_URL"),
			HeadRef:           os.Getenv("GITHUB_HEAD_REF"),
			Job:               os.Getenv("GITHUB_JOB"),
			Path:              os.Getenv("GITHUB_PATH"),
			Ref:               os.Getenv("GITHUB_REF"),
			RefName:           os.Getenv("GITHUB_REF_NAME"),
			RefProtected:      os.Getenv("GITHUB_REF_PROTECTED") == "true",
			RefType:           os.Getenv("GITHUB_REF_TYPE"),
			Repository:        os.Getenv("GITHUB_REPOSITORY"),
			RepositoryID:      os.Getenv("GITHUB_REPOSITORY_ID"),
			RepositoryOwner:   os.Getenv("GITHUB_REPOSITORY_OWNER"),
			RepositoryOwnerID: os.Getenv("GITHUB_REPOSITORY_OWNER_ID"),
			RetentionDays:     retentionDays,
			RunAttempt:        os.Getenv("GITHUB_RUN_ATTEMPT"),
			RunID:             os.Getenv("GITHUB_RUN_ID"),
			RunNumber:         os.Getenv("GITHUB_RUN_NUMBER"),
			ServerURL:         os.Getenv("GITHUB_SERVER_URL"),
			SHA:               os.Getenv("GITHUB_SHA"),
			Workflow:          os.Getenv("GITHUB_WORKFLOW"),
			WorkflowRef:       os.Getenv("GITHUB_WORKFLOW_REF"),
			WorkflowSHA:       os.Getenv("GITHUB_WORKFLOW_SHA"),
			Workspace:         os.Getenv("GITHUB_WORKSPACE"),
			Token:             os.Getenv("GITHUB_TOKEN"),
		},
		Env:   make(map[string]string),
		Vars:  make(map[string]string),
		Job:   &JobContext{},
		Steps: make(map[string]*StepResult),
		Runner: &RunnerContext{
			Name:      os.Getenv("RUNNER_NAME"),
			OS:        os.Getenv("RUNNER_OS"),
			Arch:      os.Getenv("RUNNER_ARCH"),
			Temp:      os.Getenv("RUNNER_TEMP"),
			ToolCache: os.Getenv("RUNNER_TOOL_CACHE"),
			Debug:     os.Getenv("RUNNER_DEBUG"),
		},
		Secrets:  make(map[string]string),
		Strategy: &StrategyContext{},
		Matrix:   make(map[string]string),
		Needs:    make(map[string]string),
	}, nil
}
func (c *Context) GetVariable(name string) (interface{}, error) {
	switch name {
	case "github":
		return c.Github, nil
	case "env":
		return c.Env, nil
	case "vars":
		return c.Vars, nil
	case "job":
		return c.Job, nil
	case "steps":
		return c.Steps, nil
	case "runner":
		return c.Runner, nil
	case "secrets":
		return c.Secrets, nil
	case "strategy":
		return c.Strategy, nil
	case "matrix":
		return c.Matrix, nil
	case "needs":
		return c.Needs, nil
	case "infinity":
		return math.Inf(1), nil
	case "nan":
		return math.NaN(), nil
	default:
		return nil, fmt.Errorf("unknown variable: %s", name)
	}
}

// GithubContext contains information about the workflow run and the event that triggered the run.
// All fields in this section are based on the GitHub context documentation. Not all of them meaningful
// for gale, but we include them all for completeness.
//
// See more: https://docs.github.com/en/actions/learn-github-actions/contexts#github-context
type GithubContext struct {

	// CI is true when GitHub Actions is running the workflow. You can use this variable to differentiate when
	// tests are being run locally or by GitHub Actions.
	CI bool `json:"ci"`

	// The name of the action currently running, or the id of a step. GitHub removes special characters, and
	// uses the name __run when the current step runs a script without an id. If you use the same action more
	// than once in the same job, the name will include a suffix with the sequence number with underscore before it.
	// For example, the first script you run will have the name __run, and the second script will be named __run_2.
	// Similarly, the second invocation of actions/checkout will be actionscheckout2.
	Action string `json:"action"`

	// The path where an action is located. This property is only supported in composite actions. You can use this
	// path to access files located in the same repository as the action, for example by changing directories to the
	// path: cd ${{ github.action_path }} .
	ActionPath string `json:"action_path"`

	// For a step executing an action, this is the ref of the action being executed. For example, v2
	ActionRef string `json:"action_ref"`

	// For a step executing an action, this is the owner and repository name of the action. For example,
	// actions/checkout.
	ActionRepository string `json:"action_repository"`

	// For a composite action, the current result of the composite action.
	ActionStatus string `json:"action_status"`

	// Always set to true when GitHub Actions is running the workflow. You can use this variable to differentiate
	// when tests are being run locally or by GitHub Actions.
	Actions bool `json:"actions"`

	// The username of the user that triggered the initial workflow run. If the workflow run is a re-run,
	// this value may differ from github.triggering_actor. Any workflow re-runs will use the privileges of
	// github.actor, even if the actor initiating the re-run (github.triggering_actor) has different privileges.
	Actor string `json:"actor"`

	// The account ID of the person or app that triggered the initial workflow run. For example, 1234567. Note
	// that this is different from the actor username.
	ActorID string `json:"actor_id"`

	// The URL of the GitHub REST API.
	// nolint: stylecheck,revive // var-naming: struct field ApiURL should be APIURL - this is reducing readability
	ApiURL string `json:"api_url"`

	// The base_ref or target branch of the pull request in a workflow run. This property is only available when
	// the event that triggers a workflow run is either pull_request or pull_request_target.
	BaseRef string `json:"base_ref"`

	// Path on the runner to the file that sets environment variables from workflow commands. This file is unique
	// to the current step and is a different file for each step in a job.
	Env string `json:"env"`

	// The full event webhook payload. You can access individual properties of the event using this context. This
	// object is identical to the webhook payload of the event that triggered the workflow run, and is different
	// for each event.
	Event map[string]interface{} `json:"event"`

	// The name of the event that triggered the workflow run.
	EventName string `json:"event_name"`

	// The path to the file on the runner that contains the full event webhook payload.
	EventPath string `json:"event_path"`

	// The URL of the GitHub GraphQL API.
	GraphqlURL string `json:"graphql_url"`

	// The head_ref or source branch of the pull request in a workflow run. This property is only available when the
	// event that triggers a workflow run is either pull_request or pull_request_target.
	HeadRef string `json:"head_ref"`

	// The job_id of the current job. Note: This context property is set by the Actions runner, and is only
	// available within the execution steps of a job. Otherwise, the value of this property will be null.
	Job string `json:"job"`

	// For jobs using a reusable workflow, the commit SHA for the reusable workflow file.
	JobWorkflowSHA string `json:"job_workflow_sha"`

	// Path on the runner to the file that sets system PATH variables from workflow commands. This file is
	// unique to the current step and is a different file for each step in a job.
	Path string `json:"path"`

	// The fully-formed ref of the branch or tag that triggered the workflow run. For workflows triggered by push,
	// this is the branch or tag ref that was pushed. For workflows triggered by pull_request, this is the pull request
	// merge branch. For workflows triggered by release, this is the release tag created. For other triggers, this is
	// the branch or tag ref that triggered the workflow run. This is only set if a branch or tag is available for the
	// event type. The ref given is fully-formed, meaning that for branches the format is refs/heads/<branch_name>,
	// for pull requests it is refs/pull/<pr_number>/merge, and for tags it is refs/tags/<tag_name>. For example,
	// refs/heads/feature-branch-1.
	Ref string `json:"ref"`

	// The short ref name of the branch or tag that triggered the workflow run. This value matches the branch or
	// tag name shown on GitHub. For example, feature-branch-1.
	RefName string `json:"ref_name"`

	// true if branch protections are configured for the ref that triggered the workflow run.
	RefProtected bool `json:"ref_protected"`

	// The type of ref that triggered the workflow run. Valid values are branch or tag.
	RefType string `json:"ref_type"`

	// The owner and repository name. For example, octocat/Hello-World.
	Repository string `json:"repository"`

	// The ID of the repository. For example, 123456789. Note that this is different from the repository name.
	RepositoryID string `json:"repository_id"`

	// The repository owner's username. For example, octocat.
	RepositoryOwner string `json:"repository_owner"`

	// The repository owner's account ID. For example, 1234567. Note that this is different from the owner's name.
	RepositoryOwnerID string `json:"repository_owner_id"`

	// The Git URL to the repository. For example, git://github.com/octocat/hello-world.git.
	RepositoryURL string `json:"repositoryUrl"`

	// The number of days that workflow run logs and artifacts are kept.
	RetentionDays int `json:"retention_days"`

	// A unique number for each workflow run within a repository. This number does not change if you re-run
	// the workflow run.
	RunID string `json:"run_id"`

	// A unique number for each run of a particular workflow in a repository. This number begins at 1 for
	// the workflow's first run, and increments with each new run. This number does not change if you re-run
	// the workflow run.
	RunNumber string `json:"run_number"`

	// A unique number for each attempt of a particular workflow run in a repository. This number begins at 1 for
	// the workflow run's first attempt, and increments with each re-run.
	RunAttempt string `json:"run_attempt"`

	// The source of a secret used in a workflow. Possible values are None, Actions, Codespaces, or Dependabot.
	SecretSource string `json:"secret_source"`

	// The URL of the GitHub server. For example: https://github.com.
	ServerURL string `json:"server_url"`

	// The commit SHA that triggered the workflow. The value of this commit SHA depends on the event that triggered
	// the workflow.
	SHA string `json:"sha"`

	// A token to authenticate on behalf of the GitHub App installed on your repository. This is
	// functionally equivalent to the GITHUB_TOKEN secret.
	Token string `json:"token"`

	// The username of the user that initiated the workflow run. If the workflow run is a re-run, this value
	// may differ from github.actor. Any workflow re-runs will use the privileges of github.actor, even if the actor
	// initiating the re-run (github.triggering_actor) has different privileges.
	TriggeringActor string `json:"triggering_actor"`

	// The name of the workflow. If the workflow file doesn't specify a name, the value of this property is the
	// full path of the workflow file in the repository.
	Workflow string `json:"workflow"`

	// The ref path to the workflow. For example,
	// octocat/hello-world/.github/workflows/my-workflow.yml@refs/heads/my_branch.
	WorkflowRef string `json:"workflow_ref"`

	// The commit SHA for the workflow file.
	WorkflowSHA string `json:"workflow_sha"`

	// The default working directory on the runner for steps, and the default location of your repository when
	// using the checkout action.
	Workspace string
}

type JobContext struct {
	Container JobContainer           // The container in which the job is running.
	Services  map[string]JobServices // The services running in the job.
	Status    string                 // The current status of the job. Possible values are queued, in_progress, or completed.
}

type JobContainer struct {
	ID      string // The ID of the container
	Network string // The ID of the container network. The runner creates the network used by all containers in a job.
}

// TODO: add ports type and custom unmarshaler to handle the different types of ports config

type JobServices struct {
	ID      string      // The ID of the service container.
	Network string      // The ID of the service container network. The runner creates the network used by all containers in a job.
	Ports   interface{} // The exposed ports of the service container.
}

// RunnerContext contains information about the runner that is executing the current job.
// All fields in this section are based on the Runner context documentation. Not all of them meaningful
// for gale, but we include them all for completeness.
//
// See more: https://docs.github.com/en/actions/learn-github-actions/contexts#runner-context
type RunnerContext struct {
	// The name of the runner executing the job.
	Name string `json:"name"`

	// The operating system of the runner executing the job. Possible values are Linux, Windows, or macOS.
	OS string `json:"os"`

	// The architecture of the runner executing the job. Possible values are X86, X64, ARM, or ARM64.
	Arch string `json:"arch"`

	// The path to a temporary directory on the runner. This directory is emptied at the beginning and end of
	// each job. Note that files will not be removed if the runner's user account does not have permission to
	// delete them.
	Temp string `json:"temp"`

	// The path to the directory containing preinstalled tools for GitHub-hosted runners.
	ToolCache string `json:"tool_cache"`

	// This is set only if debug logging is enabled, and always has the value of 1. It can be useful as an
	// indicator to enable additional debugging or verbose logging in your own job steps.
	Debug string `json:"debug"`
}

type StrategyContext struct {
	FailFast    bool // FailFast is whether to stop the job when one matrix combination fails.
	JobIndex    int  // JobIndex is the index of the current job in the matrix.
	JobTotal    int  // JobTotal is the total number of jobs in the matrix.
	MaxParallel int  // MaxParallel is the maximum number of jobs to run concurrently.
}

type NeedsContext struct {
	Outputs map[string]string // Outputs is a map of job names to their outputs.
	Result  string            // Result is the result of the job that this job depends on.
}
