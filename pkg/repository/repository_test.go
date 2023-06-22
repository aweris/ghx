package repository_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"dagger.io/dagger"

	"github.com/aweris/ghx/pkg/repository"
)

func TestLoadWorkflows(t *testing.T) {
	ctx := context.Background()

	client, err := dagger.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Get the current file's directory
	//nolint:dogsled // only need filename here and it's okay to allow blank identifier's for the other return values
	_, filename, _, _ := runtime.Caller(0)

	// Get the root directory by going up two levels
	rootDir := filepath.Join(filepath.Dir(filename), "../..")

	// Load workflows from custom directory
	workflows, err := repository.LoadWorkflows(ctx, client, rootDir, "pkg/repository/testdata/workflows")
	if err != nil {
		t.Fatal(err)
	}

	if len(workflows) != 1 {
		t.Errorf("expected 1 workflows, got %d", len(workflows))
	}

	workflow, ok := workflows["test-workflow"]
	if !ok {
		t.Errorf("workflow not found")
	}

	if workflow.Name != "test-workflow" {
		t.Errorf("expected workflow name to be test-workflow, got %s", workflow.Name)
	}

	if workflow.Path != "pkg/repository/testdata/workflows/test.yaml" {
		t.Errorf("expected workflow path to be pkg/repository/testdata/workflows/test.yaml, got %s", workflow.Path)
	}
}
