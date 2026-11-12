package subscription

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestDefaultSubscriberRegistry_schemas(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		sub  kargoapi.RepoSubscription
		// cfg is valid for the subscription's type only
		cfg string
	}{
		{
			name: "chart",
			sub:  kargoapi.RepoSubscription{Chart: &kargoapi.ChartSubscription{}},
			cfg:  `{"repoURL":"oci://example.com/chart","semverConstraint":"^1"}`,
		},
		{
			name: "git",
			sub:  kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{}},
			cfg:  `{"repoURL":"https://example.com/repo","branch":"main"}`,
		},
		{
			name: "image",
			sub:  kargoapi.RepoSubscription{Image: &kargoapi.ImageSubscription{}},
			cfg:  `{"repoURL":"example.com/image","constraint":"^1"}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			reg, err := DefaultSubscriberRegistry.Get(t.Context(), testCase.sub)
			require.NoError(t, err)
			require.NotNil(t, reg.Metadata.Schema)
			require.Empty(t, reg.Metadata.Schema.Validate(nil, []byte(testCase.cfg)))
			for _, other := range testCases {
				if other.name != testCase.name {
					require.NotEmpty(t, reg.Metadata.Schema.Validate(nil, []byte(other.cfg)))
				}
			}
		})
	}
}
