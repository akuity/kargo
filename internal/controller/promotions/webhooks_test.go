package promotions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() (*api.Promotion, *api.Promotion)
		assertions func(error)
	}{
		{
			name: "attempt to mutate",
			setup: func() (*api.Promotion, *api.Promotion) {
				old := &api.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				}
				new := old.DeepCopy()
				new.Spec.State = "another-fake-state"
				return old, new
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "\"fake-name\" is invalid")
				require.Contains(t, err.Error(), "spec is immutable")
			},
		},

		{
			name: "update without mutation",
			setup: func() (*api.Promotion, *api.Promotion) {
				old := &api.Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &api.PromotionSpec{
						Environment: "fake-environment",
						State:       "fake-state",
					},
				}
				new := old.DeepCopy()
				return old, new
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	w := &webhook{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			old, new := testCase.setup()
			testCase.assertions(w.ValidateUpdate(context.Background(), old, new))
		})
	}
}
