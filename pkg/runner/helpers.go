package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aweris/ghx/internal/log"
	"github.com/aweris/ghx/pkg/actions"
	"github.com/aweris/ghx/pkg/config"
	"github.com/aweris/ghx/pkg/model"
	statepkg "github.com/aweris/ghx/pkg/state"
)

// getStepEnv returns the environment variables for the step to load in cmd exec
func getStepEnv(state *statepkg.State, ss *statepkg.StepState, stage model.ActionStage) ([]string, error) {
	// getting the current environment first
	env := os.Environ()

	// adding the environment variables of the workflow and job to the environment. for duplicate keys, the last one wins
	for k, v := range state.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// adding the state, input and environment variables to the environment. for duplicate keys, the last one wins
	// so getting the current environment first is important

	for k, v := range ss.State {
		env = append(env, fmt.Sprintf("STATE_%s=%s", k, v))
	}

	if ss.Step.Type() == model.StepTypeAction {
		as, ok := state.GetActionState(ss.Step.Uses)
		if !ok {
			return nil, fmt.Errorf("action state not found for action %s", ss.Step.Uses)
		}

		// get context for the github actions expressions
		ac := state.GetActionsContext()

		// add inputs to the environment
		for k, v := range ss.Step.With {
			// TODO: temporary solution evaluate github expressions here. Need to implement a proper solution

			// convert value to Evaluable String type
			str := actions.NewString(v)

			// evaluate the expression
			res, err := str.Eval(ac)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate default value for input %s: %v", k, err)
			}

			env = append(env, fmt.Sprintf("INPUT_%s=%s", strings.ToUpper(k), res))
		}

		// add default values for inputs that are not defined in the step config
		for k, v := range as.Metadata.Inputs {
			if _, ok := ss.Step.With[k]; ok {
				continue
			}

			if v.Default == "" {
				continue
			}
			// TODO: temporary solution evaluate github expressions here. Need to implement a proper solution

			// convert value to Evaluable String type
			str := actions.NewString(v.Default)

			// evaluate the expression
			res, err := str.Eval(ac)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate default value for input %s: %v", k, err)
			}

			env = append(env, fmt.Sprintf("INPUT_%s=%s", strings.ToUpper(k), res))
		}
	}

	for k, v := range ss.Step.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// create directory and files for file commands and add them to the environment as well

	dir := filepath.Join("steps", ss.Step.ID, string(stage), "file_commands")

	var err error

	env, err = appendFileCommandPathToEnv(env, "GITHUB_ENV", filepath.Join(dir, "env"))
	if err != nil {
		return nil, err
	}

	env, err = appendFileCommandPathToEnv(env, "GITHUB_PATH", filepath.Join(dir, "path"))
	if err != nil {
		return nil, err
	}

	env, err = appendFileCommandPathToEnv(env, "GITHUB_STEP_SUMMARY", filepath.Join(dir, "step_summary"))
	if err != nil {
		return nil, err
	}

	env, err = appendFileCommandPathToEnv(env, "GITHUB_ACTION_OUTPUT", filepath.Join(dir, "output"))
	if err != nil {
		return nil, err
	}

	return env, nil
}

// appendFileCommandPathToEnv creates a file and appends the path to the environment
func appendFileCommandPathToEnv(env []string, key string, path string) ([]string, error) {
	// ensure the file exists
	if err := config.EnsureFile(path); err != nil {
		return nil, err
	}

	env = append(env, fmt.Sprintf("%s=%s", key, config.GetPath(path)))

	return env, nil
}

// valuesFromFile extracts file command values from a file. The method supports:
// - single line key values (e.g. key=value)
// - multi-line key values (e.g. key=<<$EOF...$EOF)
// - single line values without keys like GITHUB_PATH values to add container $PATH
//
// The method returns a map of key values. Single line values without keys are added to the map as key with an empty
// value (e.g. "/root/go/bin": "").
func valuesFromFile(filePath string) (map[string]string, error) {
	keyValues := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentKey string
	var valueBuilder strings.Builder
	var inMultiLineValue bool
	var endMarker string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if the line contains "<<" indicating the start of a multi-line value
		if strings.Contains(line, "<<") {
			if inMultiLineValue {
				return nil, fmt.Errorf("unexpected '<<' in line: %s", line)
			}

			parts := strings.SplitN(line, "<<", 2)
			currentKey = strings.TrimSpace(parts[0])
			valueBuilder.Reset()
			inMultiLineValue = true

			// Extract the end marker from the line
			endMarker = strings.TrimSpace(parts[1])

			continue
		}

		// Check if there is active multi-line value processing
		if inMultiLineValue {
			// Check if the line is the end of the multi-line value
			if strings.TrimSpace(line) == endMarker {
				inMultiLineValue = false
				value := strings.TrimSpace(strings.TrimSuffix(valueBuilder.String(), "\n"))
				keyValues[currentKey] = value
			} else {
				valueBuilder.WriteString(line)
			}

			continue
		}

		// Single line format: "key=value"
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			keyValues[key] = value
		} else if len(parts) == 1 { // Single line format: "key" like in path values
			key := strings.TrimSpace(parts[0])
			keyValues[key] = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keyValues, nil
}

func processGithubWorkflowCommands(cmd *model.Command, ss *statepkg.StepState, logger *log.Logger) error {
	switch cmd.Name {
	case "group":
		logger.Info(cmd.Value)
		logger.StartGroup()
	case "endgroup":
		logger.EndGroup()
	case "debug":
		logger.Debug(cmd.Value)
	case "error":
		logger.Errorf(cmd.Value, "file", cmd.Parameters["file"], "line", cmd.Parameters["line"], "col", cmd.Parameters["col"], "endLine", cmd.Parameters["endLine"], "endCol", cmd.Parameters["endCol"], "title", cmd.Parameters["title"])
	case "warning":
		logger.Warnf(cmd.Value, "file", cmd.Parameters["file"], "line", cmd.Parameters["line"], "col", cmd.Parameters["col"], "endLine", cmd.Parameters["endLine"], "endCol", cmd.Parameters["endCol"], "title", cmd.Parameters["title"])
	case "notice":
		logger.Noticef(cmd.Value, "file", cmd.Parameters["file"], "line", cmd.Parameters["line"], "col", cmd.Parameters["col"], "endLine", cmd.Parameters["endLine"], "endCol", cmd.Parameters["endCol"], "title", cmd.Parameters["title"])
	case "set-env":
		if err := os.Setenv(cmd.Parameters["name"], cmd.Value); err != nil {
			return err
		}
	case "set-output":
		ss.Result.Outputs[cmd.Parameters["name"]] = cmd.Value
	case "save-state":
		ss.State[cmd.Parameters["name"]] = cmd.Value
	case "add-mask":
		logger.Info(fmt.Sprintf("[add-mask] %s", cmd.Value))
	case "add-matcher":
		logger.Info(fmt.Sprintf("[add-matcher] %s", cmd.Value))
	case "add-path":
		path := os.Getenv("PATH")
		path = fmt.Sprintf("%s:%s", path, cmd.Value)
		if err := os.Setenv("PATH", path); err != nil {
			return err
		}
	}

	return nil
}

func processFileCommands(ss *statepkg.StepState, stage model.ActionStage) error {
	dir := config.GetPath("steps", ss.Step.ID, string(stage), "file_commands")

	env, err := valuesFromFile(filepath.Join(dir, "env"))
	if err != nil {
		return err
	}

	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}

	paths, err := valuesFromFile(filepath.Join(dir, "path"))
	if err != nil {
		return err
	}

	path := os.Getenv("PATH")

	for p := range paths {
		path = fmt.Sprintf("%s:%s", path, p)
	}

	if err := os.Setenv("PATH", path); err != nil {
		return err
	}

	output, err := valuesFromFile(filepath.Join(dir, "output"))
	if err != nil {
		return err
	}

	if ss.Result.Outputs == nil {
		ss.Result.Outputs = make(map[string]string)
	}

	for k, v := range output {
		ss.Result.Outputs[k] = v
	}

	// TODO: not sure what should I do with step summary, so I'm ignoring it for now

	return nil
}
