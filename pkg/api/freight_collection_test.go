package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestAddFreightToCollection(t *testing.T) {
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
		name      string
		requested []kargoapi.FreightRequest
		previous  *kargoapi.FreightCollection
		expected  map[string]string
	}{
		{
			name:      "Stage requesting one origin takes nothing from the previous collection",
			requested: requested("images"),
			previous:  collection(ref("old-images", "images"), ref("cfg", "config")),
			expected:  map[string]string{"Warehouse/images": "new-images"},
		},
		{
			name:      "no previous collection",
			requested: requested("images", "config"),
			expected:  map[string]string{"Warehouse/images": "new-images"},
		},
		{
			name:      "previous collection holding no Freight",
			requested: requested("images", "config"),
			previous:  &kargoapi.FreightCollection{},
			expected:  map[string]string{"Warehouse/images": "new-images"},
		},
		{
			name:      "other origins are carried over from the previous collection",
			requested: requested("images", "config"),
			previous:  collection(ref("old-images", "images"), ref("cfg", "config")),
			expected: map[string]string{
				"Warehouse/images": "new-images",
				"Warehouse/config": "cfg",
			},
		},
		{
			name:      "origins no longer requested are dropped",
			requested: requested("images", "config"),
			previous: collection(
				ref("old-images", "images"),
				ref("cfg", "config"),
				ref("stale", "retired"),
			),
			expected: map[string]string{
				"Warehouse/images": "new-images",
				"Warehouse/config": "cfg",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var previousBefore *kargoapi.FreightCollection
			if testCase.previous != nil {
				previousBefore = testCase.previous.DeepCopy()
			}
			result := addFreightToCollection(
				context.Background(),
				&kargoapi.Stage{
					Spec: kargoapi.StageSpec{RequestedFreight: testCase.requested},
				},
				testCase.previous,
				promoted,
			)
			require.NotNil(t, result)
			require.Equal(t, testCase.expected, names(result))
			require.NotEmpty(t, result.ID)
			// The previous collection is left as it was.
			require.Equal(t, previousBefore, testCase.previous)
		})
	}
}

func TestLastPromotionFreight(t *testing.T) {
	testCollection := &kargoapi.FreightCollection{ID: "fake-id"}
	testCases := []struct {
		name     string
		status   kargoapi.StageStatus
		expected *kargoapi.FreightCollection
	}{
		{
			name: "no last Promotion",
		},
		{
			name: "last Promotion without a status",
			status: kargoapi.StageStatus{
				LastPromotion: &kargoapi.PromotionReference{},
			},
		},
		{
			name: "last Promotion without a collection",
			status: kargoapi.StageStatus{
				LastPromotion: &kargoapi.PromotionReference{
					Status: &kargoapi.PromotionStatus{},
				},
			},
		},
		{
			name: "last Promotion with a collection",
			status: kargoapi.StageStatus{
				LastPromotion: &kargoapi.PromotionReference{
					Status: &kargoapi.PromotionStatus{
						FreightCollection: testCollection,
					},
				},
			},
			expected: testCollection,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Same(
				t,
				testCase.expected,
				lastPromotionFreight(&kargoapi.Stage{Status: testCase.status}),
			)
		})
	}
}

func TestResolveFreightRefs(t *testing.T) {
	const (
		testNamespace = "fake-namespace"
		testRollout   = "0193a6b0-0000-7000-8000-000000000001"
		testRequest   = "fake-request"
	)

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	origin := func(name string) kargoapi.FreightOrigin {
		return kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: name,
		}
	}
	ref := func(freight, warehouse string) kargoapi.FreightReference {
		return kargoapi.FreightReference{Name: freight, Origin: origin(warehouse)}
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
	// newStage returns a Stage that requests Freight directly from each of the
	// named Warehouses.
	newStage := func(warehouses ...string) *kargoapi.Stage {
		stage := &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "fake-stage",
			},
		}
		for _, w := range warehouses {
			stage.Spec.RequestedFreight = append(
				stage.Spec.RequestedFreight,
				kargoapi.FreightRequest{
					Origin:  origin(w),
					Sources: kargoapi.FreightSources{Direct: true},
				},
			)
		}
		return stage
	}
	newTargetAwareStage := func(warehouses ...string) *kargoapi.Stage {
		stage := newStage(warehouses...)
		stage.Spec.Targets = &kargoapi.StageTargets{}
		return stage
	}
	rolloutChild := kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "fake-rollout-child",
		},
		Spec: kargoapi.PromotionSpec{
			Target:           "fake-target",
			PromotionRollout: testRollout,
		},
	}
	requestChild := kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "fake-request-child",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: kargoapi.GroupVersion.String(),
				Kind:       promotionRequestKind,
				Name:       testRequest,
				Controller: ptr.To(true),
			}},
		},
		Spec: kargoapi.PromotionSpec{Target: "fake-target"},
	}

	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "new-images",
		},
		Origin: origin("images"),
		Images: []kargoapi.Image{{RepoURL: "fake-repo", Tag: "v2"}},
	}

	testCases := []struct {
		name    string
		promo   kargoapi.Promotion
		stage   *kargoapi.Stage
		freight *kargoapi.Freight
		objects []client.Object
		assert  func(
			*testing.T,
			*kargoapi.FreightReference,
			*kargoapi.FreightCollection,
			error,
		)
	}{
		{
			name:  "Freight not found",
			stage: newStage("images"),
			assert: func(
				t *testing.T,
				_ *kargoapi.FreightReference,
				_ *kargoapi.FreightCollection,
				err error,
			) {
				require.ErrorContains(t, err, "not found in namespace")
			},
		},
		{
			name:    "Freight not available to the Stage",
			stage:   newStage("config"),
			freight: testFreight,
			assert: func(
				t *testing.T,
				_ *kargoapi.FreightReference,
				_ *kargoapi.FreightCollection,
				err error,
			) {
				require.ErrorContains(t, err, "is not available to Stage")
			},
		},
		{
			name: "references already resolved are returned as they are",
			promo: kargoapi.Promotion{
				Spec: rolloutChild.Spec,
				Status: kargoapi.PromotionStatus{
					Freight:           ptr.To(ref("resolved", "images")),
					FreightCollection: collection(ref("resolved", "images")),
				},
			},
			stage:   newTargetAwareStage("images"),
			freight: testFreight,
			assert: func(
				t *testing.T,
				freightRef *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, "resolved", freightRef.Name)
				require.Equal(
					t,
					map[string]string{"Warehouse/images": "resolved"},
					names(freightCol),
				)
			},
		},
		{
			name:    "rollout child needs no PromotionRequest",
			promo:   rolloutChild,
			stage:   newTargetAwareStage("images"),
			freight: testFreight,
			assert: func(
				t *testing.T,
				freightRef *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, testFreight.Name, freightRef.Name)
				require.Equal(t, testFreight.Origin, freightRef.Origin)
				require.Equal(t, testFreight.Images, freightRef.Images)
				require.Equal(
					t,
					map[string]string{"Warehouse/images": "new-images"},
					names(freightCol),
				)
				require.NotEmpty(t, freightCol.ID)
			},
		},
		{
			name:  "rollout child carries other origins over from the Freight history",
			promo: rolloutChild,
			stage: func() *kargoapi.Stage {
				stage := newTargetAwareStage("images", "config")
				stage.Status.FreightHistory = kargoapi.FreightHistory{
					collection(ref("old-images", "images"), ref("cfg", "config")),
					collection(ref("older-images", "images"), ref("old-cfg", "config")),
				}
				// A target-aware Stage records no last Promotion; were it to,
				// it is not what a rollout's child inherits from.
				stage.Status.LastPromotion = &kargoapi.PromotionReference{
					Status: &kargoapi.PromotionStatus{
						FreightCollection: collection(ref("wrong-cfg", "config")),
					},
				}
				return stage
			}(),
			freight: testFreight,
			assert: func(
				t *testing.T,
				freightRef *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, testFreight.Name, freightRef.Name)
				require.Equal(
					t,
					map[string]string{
						"Warehouse/images": "new-images",
						"Warehouse/config": "cfg",
					},
					names(freightCol),
				)
			},
		},
		{
			name:  "rollout child drops origins no longer requested",
			promo: rolloutChild,
			stage: func() *kargoapi.Stage {
				stage := newTargetAwareStage("images", "config")
				stage.Status.FreightHistory = kargoapi.FreightHistory{collection(
					ref("old-images", "images"),
					ref("cfg", "config"),
					ref("stale", "retired"),
				)}
				return stage
			}(),
			freight: testFreight,
			assert: func(
				t *testing.T,
				_ *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]string{
						"Warehouse/images": "new-images",
						"Warehouse/config": "cfg",
					},
					names(freightCol),
				)
			},
		},
		{
			name:    "rollout child of a Stage with no Freight history",
			promo:   rolloutChild,
			stage:   newTargetAwareStage("images", "config"),
			freight: testFreight,
			assert: func(
				t *testing.T,
				_ *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]string{"Warehouse/images": "new-images"},
					names(freightCol),
				)
			},
		},
		{
			name: "Promotion to a Target with neither rollout nor PromotionRequest",
			promo: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      "fake-orphan",
				},
				Spec: kargoapi.PromotionSpec{Target: "fake-target"},
			},
			stage:   newTargetAwareStage("images"),
			freight: testFreight,
			assert: func(
				t *testing.T,
				_ *kargoapi.FreightReference,
				_ *kargoapi.FreightCollection,
				err error,
			) {
				require.ErrorContains(t, err, "cannot find PromotionRequest")
			},
		},
		{
			name:    "PromotionRequest child takes its PromotionRequest's references",
			promo:   requestChild,
			stage:   newTargetAwareStage("images", "config"),
			freight: testFreight,
			objects: []client.Object{&kargoapi.PromotionRequest{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testRequest,
				},
				Status: kargoapi.PromotionRequestStatus{
					Freight: ptr.To(ref("new-images", "images")),
					FreightCollection: collection(
						ref("new-images", "images"),
						ref("request-cfg", "config"),
					),
				},
			}},
			assert: func(
				t *testing.T,
				freightRef *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, "new-images", freightRef.Name)
				require.Equal(
					t,
					map[string]string{
						"Warehouse/images": "new-images",
						"Warehouse/config": "request-cfg",
					},
					names(freightCol),
				)
			},
		},
		{
			name: "regular Stage's Promotion carries other origins over from the last Promotion",
			stage: func() *kargoapi.Stage {
				stage := newStage("images", "config")
				stage.Status.LastPromotion = &kargoapi.PromotionReference{
					Status: &kargoapi.PromotionStatus{
						FreightCollection: collection(
							ref("old-images", "images"),
							ref("cfg", "config"),
						),
					},
				}
				return stage
			}(),
			freight: testFreight,
			assert: func(
				t *testing.T,
				freightRef *kargoapi.FreightReference,
				freightCol *kargoapi.FreightCollection,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, testFreight.Name, freightRef.Name)
				require.Equal(
					t,
					map[string]string{
						"Warehouse/images": "new-images",
						"Warehouse/config": "cfg",
					},
					names(freightCol),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			freightRef, freightCol, err := ResolveFreightRefs(
				context.Background(),
				fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(testCase.objects...).
					Build(),
				testCase.promo,
				testCase.stage,
				testCase.freight,
			)
			testCase.assert(t, freightRef, freightCol, err)
		})
	}
}
