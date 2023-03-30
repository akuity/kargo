package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() (*Promotion, *Promotion)
		assertions func(error)
	}{
		{
			name: "attempt to mutate",
			setup: func() (*Promotion, *Promotion) {
				old := &Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &PromotionSpec{
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
			setup: func() (*Promotion, *Promotion) {
				old := &Promotion{
					ObjectMeta: v1.ObjectMeta{
						Name:      "fake-name",
						Namespace: "fake-namespace",
					},
					Spec: &PromotionSpec{
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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			old, new := testCase.setup()
			testCase.assertions(new.ValidateUpdate(old))
		})
	}
}
