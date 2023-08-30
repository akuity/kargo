package kargo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/akuity/kargo/api/v1alpha1"
	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
)

func TestNewPromotion(t *testing.T) {
	const (
		testFreight          = "f08b2e72c9b2b7b263da6d55f9536e49b5ce972c"
		veryLongResourceName = "the-kubernetes-maximum-length-of-a-label-value-is-only-sixty-" +
			"three-characters-meanwhile-the-maximum-length-of-a-kubernetes-resource-name-" +
			"is-two-hundred-and-fifty-three-characters-but-this-string-is-two-hundred-" +
			"and-thirty-seven-characters"
	)
	t.Parallel()
	testCases := []struct {
		name       string
		stage      api.Stage
		freight    string
		assertions func(*testing.T, api.Stage, kubev1alpha1.Promotion)
	}{
		{
			name: "Promote stage",
			stage: api.Stage{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "80b44831-ac8d-4900-9df9-ee95f80c0fae",
					Name:      "test",
					Namespace: "kargo-demo",
				},
			},
			freight: testFreight,
			assertions: func(t *testing.T, stage api.Stage, promo kubev1alpha1.Promotion) {
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, "test", parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
			},
		},
		{
			name: "Promote stage with shard",
			stage: api.Stage{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "80b44831-ac8d-4900-9df9-ee95f80c0fae",
					Name:      "test",
					Namespace: "kargo-demo",
					Labels: map[string]string{
						controller.ShardLabelKey: "another-shard",
					},
				},
			},
			freight: testFreight,
			assertions: func(t *testing.T, stage api.Stage, promo kubev1alpha1.Promotion) {
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, "test", parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
				require.Equal(t, "another-shard", promo.Labels[controller.ShardLabelKey])
			},
		},
		{
			name: "Promote stage with very long name",
			stage: api.Stage{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "80b44831-ac8d-4900-9df9-ee95f80c0fae",
					Name:      veryLongResourceName,
					Namespace: "kargo-demo",
				},
			},
			freight: testFreight,
			assertions: func(t *testing.T, stage api.Stage, promo kubev1alpha1.Promotion) {
				require.Len(t, promo.Name, 253)
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, veryLongResourceName[0:maxStageNamePrefixLength], parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			promo := NewPromotion(tc.stage, tc.freight)
			require.True(t, metav1.IsControlledBy(&promo, &tc.stage))
			require.Equal(t, tc.freight, promo.Spec.State)
			require.Equal(t, tc.stage.Name, promo.Spec.Stage)
			require.Equal(t, tc.freight, promo.Spec.State)
			require.LessOrEqual(t, len(promo.Name), 253)
			tc.assertions(t, tc.stage, promo)
		})
	}
}
