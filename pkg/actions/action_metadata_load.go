package actions

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"dagger.io/dagger"

	"gopkg.in/yaml.v3"
)

// LoadActionFromSource loads an action from given source. Source can be a local directory or a remote repository.
func LoadActionFromSource(ctx context.Context, client *dagger.Client, src string) (*Action, error) {
	var dir *dagger.Directory

	dir, dirErr := getActionDirectory(client, src)
	if dirErr != nil {
		return nil, dirErr
	}

	file, findActionErr := findActionMetadataFileName(ctx, dir)
	if findActionErr != nil {
		return nil, findActionErr
	}

	content, contentErr := dir.File(file).Contents(ctx)
	if contentErr != nil {
		return nil, fmt.Errorf("failed to read %s/%s: %v", src, file, contentErr)
	}

	var action Action

	if unmarshalErr := yaml.Unmarshal([]byte(content), &action); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal %s/%s: %v", src, file, unmarshalErr)
	}

	action.Directory = dir

	return &action, nil
}

// getActionDirectory returns the directory of the action from given source.
func getActionDirectory(client *dagger.Client, src string) (*dagger.Directory, error) {
	// if path is relative, use the host to resolve the path
	if strings.HasPrefix(src, "./") || filepath.IsAbs(src) || strings.HasPrefix(src, "/") {
		return client.Host().Directory(src), nil
	}

	// if path is not a relative path, it must be a remote repository in the format "{owner}/{repo}/{path}@{ref}"
	// if {path} is not present in the input string, an empty string is returned for the path component.

	actionRepo, actionPath, actionRef, err := parseRepoRef(src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repo ref %s: %v", src, err)
	}

	// if path is empty, use the root of the repo as the action directory
	if actionPath == "" {
		actionPath = "."
	}

	// TODO: handle enterprise github instances as well
	// TODO: handle ref type (branch, tag, commit) currently only tags are supported
	return client.Git(path.Join("github.com", actionRepo)).Tag(actionRef).Tree().Directory(actionPath), nil
}

// findActionMetadataFileName finds the action.yml or action.yaml file in the root of the action directory.
func findActionMetadataFileName(ctx context.Context, dir *dagger.Directory) (string, error) {
	// list all entries in the root of the action directory
	entries, entriesErr := dir.Entries(ctx)
	if entriesErr != nil {
		return "", fmt.Errorf("failed to list entries for: %v", entriesErr)
	}

	file := ""

	// find action.yml or action.yaml exists in the root of the action repo
	for _, entry := range entries {
		if entry == "action.yml" || entry == "action.yaml" {
			file = entry
			break
		}
	}

	// if action.yml or action.yaml does not exist, return an error
	if file == "" {
		return "", fmt.Errorf("action.yml or action.yaml not found in the root of the action directory")
	}

	return file, nil
}

// parseRepoRef parses a string in the format "{owner}/{repo}/{path}@{ref}" and returns the parsed components.
// If {path} is not present in the input string, an empty string is returned for the path component.
func parseRepoRef(input string) (repo string, path string, ref string, err error) {
	regex := regexp.MustCompile(`^([^/]+)/([^/@]+)(?:/([^@]+))?@(.+)$`)
	matches := regex.FindStringSubmatch(input)

	if len(matches) == 0 {
		err = fmt.Errorf("invalid input format: %q", input)
		return
	}

	repo = strings.Join([]string{matches[1], matches[2]}, "/")
	path = matches[3]
	ref = matches[4]

	return
}
