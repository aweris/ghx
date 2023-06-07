package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/aweris/ghx/pkg/actions"
	"os"
	"path/filepath"
	"strings"

	"github.com/aweris/ghx/pkg/model"
)

// ensureDir ensures that the given directory exists.
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}

	return nil
}

// readRunnerState reads the state of the runner from the file system. if the state does not exist, it will be created.
func readRunnerState() (*State, error) {
	file := filepath.Join(ContainerRunnerPath, ContainerRunnerState)

	// ensure the file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// Create the file if it doesn't exist
		file, err := os.Create(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %v", err)
		}
		defer file.Close()
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	state := &State{
		ActionPathsBySource: make(map[string]string),
		StepOrder:           make([]string, 0),
		Children:            make(map[string]*StepRunState),
	}

	if len(data) > 0 {
		err := json.Unmarshal(data, state)
		if err != nil {
			return nil, err
		}
	}

	return state, nil
}

// writeRunnerState writes the state of the runner to the file system.
func writeRunnerState(state *State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(ContainerRunnerPath, ContainerRunnerState), data, 0600)
}

// exportLogArtifact exports a log artifact to the file system.
func exportStepLogs(srs *StepRunState, stage model.ActionStage, filename string, content []byte) error {
	path := filepath.Join(srs.GetStepDataDir(actions.ActionStage(stage)), "logs")

	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(path, filename), content, 0600)
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
