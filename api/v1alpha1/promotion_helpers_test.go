package v1alpha1

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAbortPromotionRequest_Equals(t *testing.T) {
	tests := []struct {
		name     string
		r1       *AbortPromotionRequest
		r2       *AbortPromotionRequest
		expected bool
	}{
		{
			name:     "both nil",
			r1:       nil,
			r2:       nil,
			expected: true,
		},
		{
			name:     "one nil",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			r2:       nil,
			expected: false,
		},
		{
			name:     "other nil",
			r1:       nil,
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different actions",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			r2:       &AbortPromotionRequest{Action: "other-action", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "different actors",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "other-actor", ControlPlane: true},
			expected: false,
		},
		{
			name:     "different control plane flags",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: false},
			expected: false,
		},
		{
			name:     "equal",
			r1:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			r2:       &AbortPromotionRequest{Action: "fake-action", Actor: "fake-actor", ControlPlane: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.r1.Equals(tt.r2), tt.expected)
		})
	}
}

func TestAbortPromotionRequest_String(t *testing.T) {
	t.Run("abort request is nil", func(t *testing.T) {
		var r *AbortPromotionRequest
		require.Empty(t, r.String())
	})

	t.Run("abort request is empty", func(t *testing.T) {
		r := &AbortPromotionRequest{}
		require.Empty(t, r.String())
	})

	t.Run("abort request has empty action", func(t *testing.T) {
		r := &AbortPromotionRequest{
			Action: "",
		}
		require.Empty(t, r.String())
	})

	t.Run("abort request has data", func(t *testing.T) {
		r := &AbortPromotionRequest{
			Action:       "foo",
			Actor:        "fake-actor",
			ControlPlane: true,
		}
		require.Equal(t, `{"action":"foo","actor":"fake-actor","controlPlane":true}`, r.String())
	})
}

func TestGetPromotion(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *Promotion, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, promo *Promotion, err error) {
				require.NoError(t, err)
				require.Nil(t, promo)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promotion",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, promo *Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-promotion", promo.Name)
				require.Equal(t, "fake-namespace", promo.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promo, err := GetPromotion(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
			)
			testCase.assertions(t, promo, err)
		})
	}
}

func TestAbortPromotion(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	t.Run("not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := AbortPromotion(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		}, AbortActionTerminate)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("already in a terminal phase", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
				Status: PromotionStatus{
					Phase: PromotionPhaseSucceeded,
				},
			},
		).Build()

		err := AbortPromotion(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		}, AbortActionTerminate)
		require.NoError(t, err)

		promotion, err := GetPromotion(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		})
		require.NoError(t, err)
		_, ok := promotion.Annotations[AnnotationKeyAbort]
		require.False(t, ok)
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
			},
		).Build()

		err := AbortPromotion(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		}, AbortActionTerminate)
		require.NoError(t, err)

		stage, err := GetPromotion(context.TODO(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		})
		require.NoError(t, err)
		require.Equal(t, (&AbortPromotionRequest{
			Action: AbortActionTerminate,
		}).String(), stage.Annotations[AnnotationKeyAbort])
	})
}

func Test_ComparePromotionByPhaseAndCreationTime(t *testing.T) {
	now := time.Date(2024, time.April, 10, 0, 0, 0, 0, time.UTC)
	ulidEarlier := ulid.MustNew(ulid.Timestamp(now.Add(-time.Hour)), nil)
	ulidLater := ulid.MustNew(ulid.Timestamp(now.Add(time.Hour)), nil)

	tests := []struct {
		name     string
		a        Promotion
		b        Promotion
		expected int
	}{
		{
			name: "Running before Terminated",
			a: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhaseRunning,
				},
			},
			b: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhaseSucceeded,
				},
			},
			expected: -1,
		},
		{
			name: "Pending before Terminated",
			a: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhasePending,
				},
			},
			b: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhaseSucceeded,
				},
			},
			expected: -1,
		},
		{
			name: "Pending after Running",
			a: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhasePending,
				},
			},
			b: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhaseRunning,
				},
			},
			expected: 1,
		},
		{
			name: "Terminated after Running",
			a: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhaseFailed,
				},
			},
			b: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhaseRunning,
				},
			},
			expected: 1,
		},
		{
			name: "Earlier ULID first if both Running",
			a: Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidEarlier.String(),
				},
				Status: PromotionStatus{
					Phase: PromotionPhaseRunning,
				},
			},
			b: Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidLater.String(),
				},
				Status: PromotionStatus{
					Phase: PromotionPhaseRunning,
				},
			},
			expected: -1,
		},
		{
			name: "Later ULID first if both Terminated",
			a: Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidLater.String(),
				},
				Status: PromotionStatus{
					Phase: PromotionPhaseErrored,
				},
			},
			b: Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidEarlier.String(),
				},
				Status: PromotionStatus{
					Phase: PromotionPhaseSucceeded,
				},
			},
			expected: -1,
		},
		{
			name: "Equal promotions",
			a: Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "promotion-a",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Status: PromotionStatus{
					Phase: PromotionPhasePending,
				},
			},
			b: Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "promotion-a",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Status: PromotionStatus{
					Phase: PromotionPhasePending,
				},
			},
			expected: 0,
		},
		{
			name: "Nil creation timestamps",
			a: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhasePending,
				},
			},
			b: Promotion{
				Status: PromotionStatus{
					Phase: PromotionPhasePending,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComparePromotionByPhaseAndCreationTime(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestComparePromotionPhase(t *testing.T) {
	tests := []struct {
		name     string
		a        PromotionPhase
		b        PromotionPhase
		expected int
	}{
		{
			name:     "Running before Terminated",
			a:        PromotionPhaseRunning,
			b:        PromotionPhaseSucceeded,
			expected: -1,
		},
		{
			name:     "Terminated after Running",
			a:        PromotionPhaseFailed,
			b:        PromotionPhaseRunning,
			expected: 1,
		},
		{
			name:     "Running before other phase",
			a:        PromotionPhaseRunning,
			b:        PromotionPhasePending,
			expected: -1,
		},
		{
			name:     "Other phase after Running",
			a:        "",
			b:        PromotionPhaseRunning,
			expected: 1,
		},
		{
			name:     "Pending before Terminated",
			a:        PromotionPhasePending,
			b:        PromotionPhaseErrored,
			expected: -1,
		},
		{
			name:     "Pending after Running",
			a:        PromotionPhasePending,
			b:        PromotionPhaseRunning,
			expected: 1,
		},
		{
			name:     "Equal Running phases",
			a:        PromotionPhaseRunning,
			b:        PromotionPhaseRunning,
			expected: 0,
		},
		{
			name: "Equal Terminated phases",
			a:    PromotionPhaseSucceeded,
			b:    PromotionPhaseFailed,
		},
		{
			name:     "Equal other phases",
			a:        PromotionPhasePending,
			b:        PromotionPhasePending,
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, ComparePromotionPhase(tt.a, tt.b))
		})
	}
}
