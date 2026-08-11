package promotionsets

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
			expected: "promotion-set-controller",
		},
		{
			name:     "with shard name",
			cfg:      ReconcilerConfig{ShardName: "my-shard"},
			expected: "promotion-set-controller-my-shard",
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
			name:        "default controller claims unlabeled PromotionSet",
			cfg:         ReconcilerConfig{IsDefaultController: true},
			responsible: true,
		},
		{
			name:        "default controller ignores PromotionSet of another shard",
			cfg:         ReconcilerConfig{IsDefaultController: true},
			shardLabel:  "my-shard",
			responsible: false,
		},
		{
			name:        "sharded controller claims PromotionSet of its own shard",
			cfg:         ReconcilerConfig{ShardName: "my-shard"},
			shardLabel:  "my-shard",
			responsible: true,
		},
		{
			name:        "sharded controller ignores PromotionSet of another shard",
			cfg:         ReconcilerConfig{ShardName: "my-shard"},
			shardLabel:  "other-shard",
			responsible: false,
		},
		{
			name:        "sharded controller ignores unlabeled PromotionSet",
			cfg:         ReconcilerConfig{ShardName: "my-shard"},
			responsible: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			promotionSet := &kargoapi.PromotionSet{}
			if testCase.shardLabel != "" {
				promotionSet.Labels = map[string]string{
					kargoapi.LabelKeyShard: testCase.shardLabel,
				}
			}

			require.Equal(
				t,
				testCase.responsible,
				testCase.cfg.shardPredicate().IsResponsible(promotionSet),
			)
		})
	}
}

func TestReconcile(t *testing.T) {
	t.Parallel()

	const (
		namespace = "fake-project"
		name      = "fake-promotion-set"
	)

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	t.Run("existing PromotionSet", func(t *testing.T) {
		promotionSet := &kargoapi.PromotionSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:  namespace,
				Name:       name,
				Generation: 1,
			},
			Spec: kargoapi.PromotionSetSpec{
				Stage:   "fake-stage",
				Freight: "fake-freight",
				Targets: []kargoapi.PromotionSetTarget{{
					Name: "fake-target",
				}},
			},
		}
		c := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(promotionSet).
			WithStatusSubresource(&kargoapi.PromotionSet{}).
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

		actual := &kargoapi.PromotionSet{}
		require.NoError(t, c.Get(t.Context(), req.NamespacedName, actual))
		require.Equal(t, int64(1), actual.Status.ObservedGeneration)
		require.Equal(t, kargoapi.PromotionSetPhaseErrored, actual.Status.Phase)
		require.NotNil(t, actual.Status.FinishedAt)
		readyCondition := conditions.Get(&actual.Status, kargoapi.ConditionTypeReady)
		require.NotNil(t, readyCondition)
		require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
		require.Equal(t, unsupportedReason, readyCondition.Reason)
		require.Equal(t, unsupportedMessage, readyCondition.Message)
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
		promotionSet := &kargoapi.PromotionSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
		}
		c := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(promotionSet).
			Build()
		r := &reconciler{
			client: c,
			reconcileFn: func(
				_ context.Context,
				kubeClient client.Client,
				actual *kargoapi.PromotionSet,
			) (ctrl.Result, error) {
				require.Same(t, c, kubeClient)
				require.Equal(t, promotionSet.Name, actual.Name)
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

	t.Run("PromotionSet not found", func(t *testing.T) {
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
