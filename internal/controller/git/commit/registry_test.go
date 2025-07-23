package commit

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
)

func TestSelectorRegistry(t *testing.T) {
	registry := selectorRegistry{}
	registry.register(
		kargoapi.CommitSelectionStrategySemVer,
		selectorRegistration{
			predicate: func(sub kargoapi.GitSubscription) bool {
				return sub.CommitSelectionStrategy ==
					kargoapi.CommitSelectionStrategySemVer
			},
			factory: func(
				kargoapi.GitSubscription,
				*git.RepoCredentials,
			) (Selector, error) {
				// No need for this factory function to work. We only care about testing
				// our ability to retrieve a factory function from the registry. We
				// won't be actually using it.
				return nil, nil
			},
		},
	)

	t.Run("duplicate registration", func(t *testing.T) {
		require.Panics(t, func() {
			registry.register(
				kargoapi.CommitSelectionStrategySemVer,
				selectorRegistration{},
			)
		})
	})

	t.Run("registration not found", func(t *testing.T) {
		_, err := registry.getSelectorFactory(kargoapi.GitSubscription{})
		require.Error(t, err)
	})

	t.Run("registration found", func(t *testing.T) {
		factory, err := registry.getSelectorFactory(
			kargoapi.GitSubscription{
				CommitSelectionStrategy: kargoapi.CommitSelectionStrategySemVer,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, factory)
	})
}
