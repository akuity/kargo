package image

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestSelectorRegistry(t *testing.T) {
	registry := selectorRegistry{}
	registry.register(
		kargoapi.ImageSelectionStrategySemVer,
		selectorRegistration{
			predicate: func(sub kargoapi.ImageSubscription) bool {
				return sub.ImageSelectionStrategy ==
					kargoapi.ImageSelectionStrategySemVer
			},
			factory: func(
				kargoapi.ImageSubscription,
				*Credentials,
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
				kargoapi.ImageSelectionStrategySemVer,
				selectorRegistration{},
			)
		})
	})

	t.Run("registration not found", func(t *testing.T) {
		_, err := registry.getSelectorFactory(kargoapi.ImageSubscription{})
		require.Error(t, err)
	})

	t.Run("registration found", func(t *testing.T) {
		factory, err := registry.getSelectorFactory(
			kargoapi.ImageSubscription{
				ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
			},
		)
		require.NoError(t, err)
		require.NotNil(t, factory)
	})
}
