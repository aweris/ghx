package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DataHome is the path to the data home directory for ghx. This is the base directory where ghx stores
// all of its data
const DataHome = "/home/runner/_temp/ghx"

// GetPath returns the path to the given file or directory under the ghx data home directory.
func GetPath(paths ...string) string {
	return filepath.Join(append([]string{DataHome}, paths...)...)
}

// EnsureDir ensures that the given directory exists under the ghx data home directory.
func EnsureDir(dir string) error {
	if _, err := os.Stat(GetPath(dir)); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}

	return nil
}

// EnsureFile ensures that the given file exists under the ghx data home directory. If the file does not exist, it will
// be created.
//
// To ensure everything is working as expected, the method will also ensure that the directory of the file
// exists.
func EnsureFile(file string) error {
	path := GetPath(file)

	// just to be sure that the directory exists
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create the file if it doesn't exist
		file, err := os.Create(GetPath(file))
		if err != nil {
			return err
		}
		defer file.Close()
	}

	return nil
}

// WriteFile ensures that the given file exists under the ghx data home directory. If the file does not
// exist, it will be created and the given content will be written to it.
//
// To ensure everything is working as expected, the method will also ensure that the directory of the file
// exists.
func WriteFile(file string, content []byte, permissions os.FileMode) error {
	if err := EnsureFile(file); err != nil {
		return err
	}

	if permissions == 0 {
		permissions = 0600
	}

	return os.WriteFile(GetPath(file), content, permissions)
}

// ReadJsonFile reads the given path as a JSON file under the ghx data home directory and unmarshal it into the given value.
func ReadJsonFile[T any](file string, val *T) error {
	data, err := os.ReadFile(GetPath(file))
	if err != nil {
		return err
	}

	// if the file is empty, we don't need to do anything
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, val)
}

// WriteJsonFile writes the given value to the given path as a JSON file under the ghx data home directory.
func WriteJsonFile(file string, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return WriteFile(file, data, 0600)
}
