package promotionrollouts

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestValidatePromotionRollout(t *testing.T) {
	t.Parallel()
	valid := func() *kargoapi.PromotionRollout {
		return &kargoapi.PromotionRollout{
			Spec: kargoapi.PromotionRolloutSpec{Stage: "prod", Freight: "abc123"},
		}
	}
	testCases := []struct {
		name    string
		rollout func() *kargoapi.PromotionRollout
		fields  []string
	}{
		{
			name:    "valid",
			rollout: valid,
		},
		{
			name: "valid with the project as namespace",
			rollout: func() *kargoapi.PromotionRollout {
				r := valid()
				r.Namespace = testProject
				return r
			},
		},
		{
			name: "stage and freight are required",
			rollout: func() *kargoapi.PromotionRollout {
				return &kargoapi.PromotionRollout{}
			},
			fields: []string{"spec.stage", "spec.freight"},
		},
		{
			name: "stage and freight must be DNS subdomains",
			rollout: func() *kargoapi.PromotionRollout {
				r := valid()
				r.Spec.Stage = "Prod"
				r.Spec.Freight = "a_b"
				return r
			},
			fields: []string{"spec.stage", "spec.freight"},
		},
		{
			name: "another namespace",
			rollout: func() *kargoapi.PromotionRollout {
				r := valid()
				r.Namespace = "other"
				return r
			},
			fields: []string{"metadata.namespace"},
		},
		{
			name: "what the server assigns is refused",
			rollout: func() *kargoapi.PromotionRollout {
				r := valid()
				r.Name = "prod.01A.abc"
				r.Number = 3
				r.Spec.Targets = []kargoapi.PromotionRolloutTarget{{Name: "us-east"}}
				r.Spec.UpdateStrategy = kargoapi.TargetUpdateStrategy{MaxConcurrent: 1}
				return r
			},
			fields: []string{"metadata.name", "number", "spec.targets", "spec.updateStrategy"},
		},
		{
			name: "metadata Kubernetes would have honored is refused",
			rollout: func() *kargoapi.PromotionRollout {
				r := valid()
				r.Labels = map[string]string{"a": "b"}
				r.Annotations = map[string]string{"a": "b"}
				r.Finalizers = []string{"x"}
				r.OwnerReferences = []metav1.OwnerReference{{Name: "x"}}
				return r
			},
			fields: []string{
				"metadata.labels", "metadata.annotations", "metadata.finalizers", "metadata.ownerReferences",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			errs := validatePromotionRollout(testCase.rollout(), testProject)
			var fields []string
			for _, err := range errs {
				fields = append(fields, err.Field)
			}
			slices.Sort(fields)
			expected := slices.Clone(testCase.fields)
			slices.Sort(expected)
			require.Equal(t, expected, fields)
		})
	}
}
