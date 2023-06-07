package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aweris/ghx/pkg/actions"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aweris/ghx/internal/log"
	"github.com/aweris/ghx/pkg/model"
)

// execCommand executes a command and processes the output for github workflow commands.
func execCommand(ctx context.Context, stage actions.ActionStage, args []string, srs *StepRunState, logger *log.Logger, actx *actions.Context) error {
	// get the step env
	env, err := srs.GetStepEnv(stage, actx)
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
				logger.Info(output)

				continue
			}

			// add the command to the list of commands to keep it as artifact
			commands = append(commands, command)

			// write to commands raw so we can keep original formatting
			commandsRaw.WriteString(output)
			commandsRaw.WriteString("\n")

			// process the command
			if err := processGithubWorkflowCommands(command, srs, logger); err != nil {
				fmt.Println(err)
			}
		}
	}()

	cmdErr := cmd.Wait()

	if data := stdout.Bytes(); len(data) > 0 {
		_ = exportStepLogs(srs, model.ActionStage(stage), "stdout.log", data)
	}

	if data := stderr.Bytes(); len(data) > 0 {
		_ = exportStepLogs(srs, model.ActionStage(stage), "stderr.log", data)
	}

	if len(commands) > 0 {
		data, err := json.Marshal(commands)
		if err != nil {
			fmt.Printf("failed to marshal commands to json: %v\n", err)
		}

		_ = exportStepLogs(srs, model.ActionStage(stage), "workflow_commands.json", data)
	}

	if data := commandsRaw.Bytes(); len(data) > 0 {
		_ = exportStepLogs(srs, model.ActionStage(stage), "workflow_commands.log", data)
	}

	if cmdErr != nil {
		return cmdErr
	}

	// process commands at the end of the command
	return processFileCommands(srs, stage)
}

func processGithubWorkflowCommands(cmd *model.Command, srs *StepRunState, logger *log.Logger) error {
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
		srs.Result.Outputs[cmd.Parameters["name"]] = cmd.Value
	case "save-state":
		srs.State[cmd.Parameters["name"]] = cmd.Value
	case "add-mask":
		fmt.Printf("[add-mask] %s\n", cmd.Value)
	case "add-matcher":
		fmt.Printf("[add-matcher] %s\n", cmd.Value)
	case "add-path":
		path := os.Getenv("PATH")
		path = fmt.Sprintf("%s:%s", path, cmd.Value)
		if err := os.Setenv("PATH", path); err != nil {
			return err
		}
	}

	return nil
}

func processFileCommands(srs *StepRunState, stage actions.ActionStage) error {
	dir := filepath.Join(srs.GetStepDataDir(stage), "file_commands")

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

	if srs.Result.Outputs == nil {
		srs.Result.Outputs = make(map[string]string)
	}

	for k, v := range output {
		srs.Result.Outputs[k] = v
	}

	// TODO: not sure what should I do with step summary, so I'm ignoring it for now

	return nil
}
