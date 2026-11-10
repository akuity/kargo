package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestBuildPromotionFreightCollection(t *testing.T) {
	origin := func(name string) kargoapi.FreightOrigin {
		return kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: name,
		}
	}
	ref := func(freight, warehouse string) kargoapi.FreightReference {
		return kargoapi.FreightReference{Name: freight, Origin: origin(warehouse)}
	}
	requested := func(warehouses ...string) []kargoapi.FreightRequest {
		reqs := make([]kargoapi.FreightRequest, 0, len(warehouses))
		for _, w := range warehouses {
			reqs = append(reqs, kargoapi.FreightRequest{Origin: origin(w)})
		}
		return reqs
	}
	collection := func(refs ...kargoapi.FreightReference) *kargoapi.FreightCollection {
		c := &kargoapi.FreightCollection{}
		c.UpdateOrPush(refs...)
		return c
	}
	names := func(c *kargoapi.FreightCollection) map[string]string {
		out := make(map[string]string, len(c.Freight))
		for o, f := range c.Freight {
			out[o] = f.Name
		}
		return out
	}

	promoted := ref("new-images", "images")

	testCases := []struct {
		name     string
		stage    *kargoapi.Stage
		expected map[string]string
	}{
		{
			name: "last Promotion without a collection is not inherited from",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{RequestedFreight: requested("images", "config")},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{Status: nil},
				},
			},
			expected: map[string]string{"Warehouse/images": "new-images"},
		},
		{
			name: "other origins are carried over from the last Promotion",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{RequestedFreight: requested("images", "config")},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							FreightCollection: collection(
								ref("old-images", "images"),
								ref("cfg", "config"),
							),
						},
					},
				},
			},
			expected: map[string]string{
				"Warehouse/images": "new-images",
				"Warehouse/config": "cfg",
			},
		},
		{
			name: "origins no longer requested are dropped",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{RequestedFreight: requested("images", "config")},
				Status: kargoapi.StageStatus{
					LastPromotion: &kargoapi.PromotionReference{
						Status: &kargoapi.PromotionStatus{
							FreightCollection: collection(
								ref("old-images", "images"),
								ref("cfg", "config"),
								ref("stale", "retired"),
							),
						},
					},
				},
			},
			expected: map[string]string{
				"Warehouse/images": "new-images",
				"Warehouse/config": "cfg",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := buildPromotionFreightCollection(context.Background(), promoted, testCase.stage)
			require.NotNil(t, result)
			require.Equal(t, testCase.expected, names(result))
			require.NotEmpty(t, result.ID)
		})
	}
}

func TestGetStageFreightRefs(t *testing.T) {
	metadata := &apiextensionsv1.JSON{Raw: []byte(`{"key":"value"}`)}
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "artifacts"}
	other := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "other"}
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{Name: "new"},
		Origin:     origin,
		Artifacts: []kargoapi.ArtifactReference{{
			SubscriptionName: "sub",
			Version:          "v2",
			Metadata:         metadata,
		}},
	}
	inherited := kargoapi.FreightReference{
		Name:   "old",
		Origin: other,
		Artifacts: []kargoapi.ArtifactReference{{
			SubscriptionName: "other-sub",
			Version:          "v1",
			Metadata:         metadata,
		}},
	}
	lastCollection := &kargoapi.FreightCollection{}
	lastCollection.UpdateOrPush(inherited)
	stage := &kargoapi.Stage{
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}, {Origin: other}},
		},
		Status: kargoapi.StageStatus{
			LastPromotion: &kargoapi.PromotionReference{
				Status: &kargoapi.PromotionStatus{FreightCollection: lastCollection},
			},
		},
	}

	ref, col, err := GetStageFreightRefs(context.Background(), freight, stage)
	require.NoError(t, err)

	require.Equal(t, "v2", ref.Artifacts[0].Version)
	require.Nil(t, ref.Artifacts[0].Metadata)
	require.Len(t, col.Freight, 2)
	for _, f := range col.Freight {
		for _, a := range f.Artifacts {
			require.Nil(t, a.Metadata)
		}
	}
	require.Equal(t, metadata, freight.Artifacts[0].Metadata)
	require.Equal(t, metadata, lastCollection.Freight[other.String()].Artifacts[0].Metadata)
}

func TestGetPromotionRequestFreightRefs(t *testing.T) {
	const namespace = "fake-namespace"
	metadata := &apiextensionsv1.JSON{Raw: []byte(`{"key":"value"}`)}
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "artifacts"}
	other := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "other"}
	targetRef := kargoapi.FreightReference{
		Name:   "new",
		Origin: origin,
		Artifacts: []kargoapi.ArtifactReference{{
			SubscriptionName: "sub",
			Version:          "v2",
			Metadata:         metadata,
		}},
	}
	collection := &kargoapi.FreightCollection{}
	collection.UpdateOrPush(targetRef, kargoapi.FreightReference{
		Name:   "old",
		Origin: other,
		Artifacts: []kargoapi.ArtifactReference{{
			SubscriptionName: "other-sub",
			Version:          "v1",
			Metadata:         metadata,
		}},
	})
	request := &kargoapi.PromotionRequest{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "fake-request"},
		Status: kargoapi.PromotionRequestStatus{
			Freight:           &targetRef,
			FreightCollection: collection,
		},
	}
	promo := kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "fake-promotion",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: kargoapi.GroupVersion.String(),
				Kind:       promotionRequestKind,
				Name:       request.Name,
				Controller: ptr.To(true),
			}},
		},
	}
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(request).Build()

	ref, col, err := GetPromotionRequestFreightRefs(
		context.Background(),
		c,
		promo,
		&kargoapi.Freight{ObjectMeta: metav1.ObjectMeta{Name: "new"}},
	)
	require.NoError(t, err)

	require.Equal(t, "v2", ref.Artifacts[0].Version)
	require.Nil(t, ref.Artifacts[0].Metadata)
	require.Equal(t, collection.ID, col.ID)
	require.Len(t, col.Freight, 2)
	for _, f := range col.Freight {
		require.NotEmpty(t, f.Artifacts[0].Version)
		require.Nil(t, f.Artifacts[0].Metadata)
	}
}

// TestGetStageFreightRefsStageStatusSize compares Stage status size with and
// without artifact Metadata.
func TestGetStageFreightRefsStageStatusSize(t *testing.T) {
	const historyLen = 10
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "artifacts"}
	stageStatusBytes := func(t *testing.T, metadataBytes int, keepMetadata bool) int {
		stage := &kargoapi.Stage{
			Spec: kargoapi.StageSpec{
				RequestedFreight: []kargoapi.FreightRequest{{Origin: origin}},
			},
		}
		raw := fmt.Sprintf(`{"data":%q}`, strings.Repeat("x", metadataBytes))
		for i := range historyLen {
			freight := &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("freight-%d", i)},
				Origin:     origin,
				Artifacts: []kargoapi.ArtifactReference{{
					ArtifactType:     "application/json",
					SubscriptionName: "sub",
					Version:          fmt.Sprintf("v%d", i),
					Metadata:         &apiextensionsv1.JSON{Raw: []byte(raw)},
				}},
			}
			ref, col, err := GetStageFreightRefs(context.Background(), freight, stage)
			require.NoError(t, err)
			if keepMetadata {
				ref.Artifacts = freight.Artifacts
				col.UpdateOrPush(*ref)
			}
			stage.Status.LastPromotion = &kargoapi.PromotionReference{
				Name:    fmt.Sprintf("promo-%d", i),
				Freight: ref.DeepCopy(),
				Status: &kargoapi.PromotionStatus{
					Phase:             kargoapi.PromotionPhaseSucceeded,
					Freight:           ref.DeepCopy(),
					FreightCollection: col.DeepCopy(),
				},
			}
			stage.Status.FreightHistory.Record(col.DeepCopy())
		}
		b, err := json.Marshal(stage.Status)
		require.NoError(t, err)
		return len(b)
	}

	for _, metadataBytes := range []int{1 << 10, 31 << 10, 120 << 10} {
		t.Run(fmt.Sprintf("metadata=%dKiB", metadataBytes>>10), func(t *testing.T) {
			before := stageStatusBytes(t, metadataBytes, true)
			after := stageStatusBytes(t, metadataBytes, false)
			t.Logf(
				"Stage status: %.1f KiB with Metadata, %.1f KiB without",
				float64(before)/1024, float64(after)/1024,
			)
			// History plus three status fields each hold a copy.
			require.Greater(t, before, (historyLen+3)*metadataBytes)
			require.Less(t, after, 8<<10)
		})
	}
}
