package promotionrequests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
)

func TestReconcilerConfigName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		cfg      ReconcilerConfig
		expected string
	}{
		{
			name:     "no shard name",
			cfg:      ReconcilerConfig{},
			expected: "promotion-request-controller",
		},
		{
			name:     "with shard name",
			cfg:      ReconcilerConfig{ShardName: "my-shard"},
			expected: "promotion-request-controller-my-shard",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, testCase.cfg.Name())
		})
	}
}

func TestReconcilerConfigShardPredicate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		cfg         ReconcilerConfig
		shardLabel  string
		responsible bool
	}{
		{
			name:        "default controller claims unlabeled PromotionRequest",
			cfg:         ReconcilerConfig{IsDefaultController: true},
			responsible: true,
		},
		{
			name:        "default controller ignores PromotionRequest of another shard",
			cfg:         ReconcilerConfig{IsDefaultController: true},
			shardLabel:  "my-shard",
			responsible: false,
		},
		{
			name:        "sharded controller claims PromotionRequest of its own shard",
			cfg:         ReconcilerConfig{ShardName: "my-shard"},
			shardLabel:  "my-shard",
			responsible: true,
		},
		{
			name:        "sharded controller ignores PromotionRequest of another shard",
			cfg:         ReconcilerConfig{ShardName: "my-shard"},
			shardLabel:  "other-shard",
			responsible: false,
		},
		{
			name:        "sharded controller ignores unlabeled PromotionRequest",
			cfg:         ReconcilerConfig{ShardName: "my-shard"},
			responsible: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			promotionRequest := &kargoapi.PromotionRequest{}
			if testCase.shardLabel != "" {
				promotionRequest.Labels = map[string]string{
					kargoapi.LabelKeyShard: testCase.shardLabel,
				}
			}

			require.Equal(
				t,
				testCase.responsible,
				testCase.cfg.shardPredicate().IsResponsible(promotionRequest),
			)
		})
	}
}

func TestReconcile(t *testing.T) {
	t.Parallel()

	const (
		namespace = "fake-project"
		name      = "fake-promotion-request"
	)

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	t.Run("existing PromotionRequest", func(t *testing.T) {
		promotionRequest := &kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  namespace,
				Name:       name,
				Generation: 1,
			},
			Spec: kargoapi.PromotionRequestSpec{
				Stage:   "fake-stage",
				Freight: "fake-freight",
				TargetSelectors: []metav1.LabelSelector{{
					MatchLabels: map[string]string{"region": "us"},
				}},
			},
		}
		c := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(promotionRequest).
			WithStatusSubresource(&kargoapi.PromotionRequest{}).
			Build()
		r := &reconciler{
			client:      c,
			reconcileFn: DefaultReconcile,
		}
		req := ctrl.Request{NamespacedName: client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}}

		result, err := r.Reconcile(t.Context(), req)
		require.NoError(t, err)
		require.Empty(t, result)

		actual := &kargoapi.PromotionRequest{}
		require.NoError(t, c.Get(t.Context(), req.NamespacedName, actual))
		require.Equal(t, int64(1), actual.Status.ObservedGeneration)
		require.Equal(t, kargoapi.PromotionRequestPhaseErrored, actual.Status.Phase)
		require.NotNil(t, actual.Status.FinishedAt)
		readyCondition := conditions.Get(&actual.Status, kargoapi.ConditionTypeReady)
		require.NotNil(t, readyCondition)
		require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
		require.Equal(t, enterpriseOnlyReason, readyCondition.Reason)
		require.Equal(t, enterpriseOnlyMessage, readyCondition.Message)
		require.Equal(t, int64(1), readyCondition.ObservedGeneration)
		require.False(t, readyCondition.LastTransitionTime.IsZero())

		firstStatus := actual.Status.DeepCopy()
		result, err = r.Reconcile(t.Context(), req)
		require.NoError(t, err)
		require.Empty(t, result)

		require.NoError(t, c.Get(t.Context(), req.NamespacedName, actual))
		require.Equal(t, firstStatus, &actual.Status)
	})

	t.Run("delegates to configured reconcile function", func(t *testing.T) {
		promotionRequest := &kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		}
		c := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(promotionRequest).
			Build()
		r := &reconciler{
			client: c,
			reconcileFn: func(
				_ context.Context,
				kubeClient client.Client,
				actual *kargoapi.PromotionRequest,
			) (ctrl.Result, error) {
				require.Same(t, c, kubeClient)
				require.Equal(t, promotionRequest.Name, actual.Name)
				return ctrl.Result{Requeue: true}, nil
			},
		}

		result, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, result)
	})

	t.Run("PromotionRequest not found", func(t *testing.T) {
		r := &reconciler{
			client:      fake.NewClientBuilder().WithScheme(scheme).Build(),
			reconcileFn: DefaultReconcile,
		}

		result, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: client.ObjectKey{
				Namespace: namespace,
				Name:      name,
			},
		})

		require.NoError(t, err)
		require.Empty(t, result)
	})
}
