package server

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
)

// fakeProjectConfigStore serves ProjectConfigs from memory, for the handlers
// that read them. The operations no such handler uses are left to the
// embedded interface, which is nil, so calling them fails the test.
type fakeProjectConfigStore struct {
	projectconfigstore.Store
	// configs holds the ProjectConfigs, keyed by Project name.
	configs map[string]projectconfigstore.Config
	// err, when set, fails every call.
	err error
}

func (s *fakeProjectConfigStore) Get(
	_ context.Context,
	project string,
	projectUID string,
) (projectconfigstore.Config, error) {
	if s.err != nil {
		return projectconfigstore.Config{}, s.err
	}
	cfg, ok := s.configs[project]
	if !ok || cfg.ProjectUID != projectUID {
		return projectconfigstore.Config{}, fmt.Errorf("ProjectConfig %q: %w", project, database.ErrNotFound)
	}
	return cfg, nil
}

// withProjectConfig returns a serverSetup that gives the server a store
// holding a ProjectConfig with the given spec for the Project with the given
// name and UID.
func withProjectConfig(
	project *kargoapi.Project,
	spec kargoapi.ProjectConfigSpec,
) func(*testing.T, *server) {
	return func(t *testing.T, s *server) {
		data, err := json.Marshal(spec)
		require.NoError(t, err)
		s.projectConfigs = &fakeProjectConfigStore{
			configs: map[string]projectconfigstore.Config{
				project.Name: {ProjectConfigRow: database.ProjectConfigRow{
					ProjectName: project.Name,
					ProjectUID:  string(project.UID),
					Spec:        data,
				}},
			},
		}
	}
}
