package state

import (
	"context"

	"dagger.io/dagger"

	"github.com/aweris/ghx/pkg/config"
	"github.com/aweris/ghx/pkg/model"
)

// ActionState represents a single action metadata and where it is stored
type ActionState struct {
	Source   string        // source of the action
	Path     string        // path of the action stored on disk
	Metadata *model.Action // metadata of the action
}

// LoadAction loads the action from the given source and stores it in the state
func LoadAction(ctx context.Context, client *dagger.Client, source string) (*ActionState, error) {
	action, err := model.LoadActionFromSource(ctx, client, source)
	if err != nil {
		return nil, err
	}

	path := config.GetPath("actions", source)

	if _, err = action.Directory.Export(ctx, path); err != nil {
		return nil, err
	}

	return &ActionState{Source: source, Path: path, Metadata: action}, nil
}
