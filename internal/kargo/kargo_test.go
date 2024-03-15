package kargo

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
		stage      kargoapi.Stage
		freight    string
		assertions func(*testing.T, kargoapi.Stage, kargoapi.Promotion)
	}{
		{
			name: "Promote stage",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "80b44831-ac8d-4900-9df9-ee95f80c0fae",
					Name:      "test",
					Namespace: "kargo-demo",
				},
			},
			freight: testFreight,
			assertions: func(t *testing.T, stage kargoapi.Stage, promo kargoapi.Promotion) {
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, "test", parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
			},
		},
		{
			name: "Promote stage with shard",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "80b44831-ac8d-4900-9df9-ee95f80c0fae",
					Name:      "test",
					Namespace: "kargo-demo",
					Labels: map[string]string{
						kargoapi.ShardLabelKey: "another-shard",
					},
				},
			},
			freight: testFreight,
			assertions: func(t *testing.T, stage kargoapi.Stage, promo kargoapi.Promotion) {
				parts := strings.Split(promo.Name, ".")
				require.Equal(t, "test", parts[0])
				require.Equal(t, testFreight[0:7], parts[2])
				require.Equal(t, "another-shard", promo.Labels[kargoapi.ShardLabelKey])
			},
		},
		{
			name: "Promote stage with very long name",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					UID:       "80b44831-ac8d-4900-9df9-ee95f80c0fae",
					Name:      veryLongResourceName,
					Namespace: "kargo-demo",
				},
			},
			freight: testFreight,
			assertions: func(t *testing.T, stage kargoapi.Stage, promo kargoapi.Promotion) {
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
			require.Equal(t, tc.freight, promo.Spec.Freight)
			require.Equal(t, tc.stage.Name, promo.Spec.Stage)
			require.Equal(t, tc.freight, promo.Spec.Freight)
			require.LessOrEqual(t, len(promo.Name), 253)
			tc.assertions(t, tc.stage, promo)
		})
	}
}

func TestIgnoreAnnotationRemovalUpdates(t *testing.T) {
	testCases := []struct {
		name     string
		old      client.Object
		new      client.Object
		expected bool
	}{
		{
			name:     "nil",
			old:      nil,
			new:      nil,
			expected: true,
		},
		{
			name: "annotation removed",
			old: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: time.Now().Format(time.RFC3339),
					},
				},
			},
			new: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: false,
		},
		{
			name: "annotation set",
			old: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			new: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kargoapi.AnnotationKeyRefresh: time.Now().Format(time.RFC3339),
					},
				},
			},
			expected: true,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p := IgnoreAnnotationRemoval{
				Annotations: []string{kargoapi.AnnotationKeyRefresh},
			}
			e := event.UpdateEvent{
				ObjectOld: tc.old,
				ObjectNew: tc.new,
			}
			require.Equal(t, tc.expected, p.Update(e))
		})
	}
}
