package garbage

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func TestCleanProjectPromotions(t *testing.T) {
	ctx := context.Background()
	logger := logging.LoggerFromContext(ctx)
	// logger.Logger.Level = log.PanicLevel
	ctx = logging.ContextWithLogger(ctx, logger)
	testCases := []struct {
		name       string
		collector  *collector
		assertions func(error)
	}{
		{
			name: "error listing Promotions",
			collector: &collector{
				listPromotionsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error listing Promotions for Project")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "fewer Promotions than max found",
			collector: &collector{
				listPromotionsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = []kargoapi.Promotion{}
					return nil
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "error deleting Promotion",
			collector: &collector{
				listPromotionsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					promos, ok := objList.(*kargoapi.PromotionList)
					require.True(t, ok)
					promos.Items = make([]kargoapi.Promotion, 100)
					for i := 0; i < 100; i++ {
						promos.Items[i] = kargoapi.Promotion{
							Status: kargoapi.PromotionStatus{
								Phase: kargoapi.PromotionPhaseSucceeded,
							},
						}
					}
					return nil
				},
				deletePromotionFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error deleting one or more Promotions from Project",
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.collector.cfg.MaxRetainedPromotions = 20
			testCase.assertions(
				testCase.collector.cleanProjectPromotions(ctx, "fake-project"),
			)
		})
	}

	t.Run("success", func(t *testing.T) {
		const numPromos = 100
		const testProject = "fake-project"

		scheme := runtime.NewScheme()
		err := kargoapi.AddToScheme(scheme)
		require.NoError(t, err)

		initialPromos := make([]client.Object, numPromos)
		creationTime := time.Now()
		for i := 0; i < numPromos; i++ {
			creationTime = creationTime.Add(-1 * time.Hour)
			initialPromos[i] = &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:              fmt.Sprintf("promotion-%d", i),
					Namespace:         testProject,
					CreationTimestamp: metav1.NewTime(creationTime),
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			}
		}

		kubeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(initialPromos...).
			Build()

		c := &collector{
			cfg: CollectorConfig{
				MaxRetainedPromotions: 20,
			},
			listPromotionsFn:  kubeClient.List,
			deletePromotionFn: kubeClient.Delete,
		}

		err = c.cleanProjectPromotions(ctx, testProject)
		require.NoError(t, err)

		promos := kargoapi.PromotionList{}
		err = kubeClient.List(
			ctx,
			&promos,
			client.InNamespace(testProject),
		)
		require.NoError(t, err)

		sort.Sort(promosByCreation(promos.Items))
		require.Len(t, promos.Items, c.cfg.MaxRetainedPromotions)
		require.Equal(
			t,
			fmt.Sprintf("promotion-%d", c.cfg.MaxRetainedPromotions-1),
			promos.Items[len(promos.Items)-1].Name,
		)
	})
}
