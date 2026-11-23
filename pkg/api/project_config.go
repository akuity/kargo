package api

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ProjectConfigGetter returns a Project's ProjectConfig, or nil, nil when the
// Project has none.
type ProjectConfigGetter interface {
	Get(ctx context.Context, project string) (*kargoapi.ProjectConfig, error)
}
