package garbage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func TestNewCollector(t *testing.T) {
	watchClient := fake.NewClientBuilder().Build()
	testCfg := CollectorConfigFromEnv()
	c, ok := NewCollector(watchClient, testCfg).(*collector)
	require.True(t, ok)
	require.Equal(t, testCfg, c.cfg)
	require.NotNil(t, c.cleanProjectsFn)
	require.NotNil(t, c.cleanProjectFn)
	require.NotNil(t, c.listProjectsFn)
	require.NotNil(t, c.listPromotionsFn)
	require.NotNil(t, c.deletePromotionFn)
}

func TestRun(t *testing.T) {
	testCases := []struct {
		name           string
		listProjectsFn func(
			context.Context,
			client.ObjectList,
			...client.ListOption,
		) error
		cleanProjectsFn func(
			ctx context.Context,
			projectCh <-chan string,
			errCh chan<- struct{},
		)
		assertions func(error)
	}{
		{
			name: "error listing Projects",
			listProjectsFn: func(
				context.Context,
				client.ObjectList,
				...client.ListOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error listing projects; no garbage collection performed",
				)
			},
		},

		{
			// The objective of this test case is to ensure that when a worker sends
			// a signal indicating that it handled an error, that Run() handles
			// that correctly.
			name: "cleanProjectsFn sends an error",
			listProjectsFn: func(
				_ context.Context,
				objList client.ObjectList,
				_ ...client.ListOption,
			) error {
				projects, ok := objList.(*corev1.NamespaceList)
				require.True(t, ok)
				projects.Items = []corev1.Namespace{{}}
				return nil
			},
			cleanProjectsFn: func(
				ctx context.Context,
				projectCh <-chan string,
				errCh chan<- struct{},
			) {
				// All we want to do is receive one Project name and send one error
				select {
				case <-projectCh:
				case <-ctx.Done():
					require.FailNow(t, "timed out waiting for a Project name")
				}
				select {
				case errCh <- struct{}{}:
				case <-ctx.Done():
					require.FailNow(t, "timed out signaling an error")
				}
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Equal(
					t,
					"one or more errors were encountered during garbage collection; "+
						"see logs for details",
					err.Error(),
				)
			},
		},

		{
			// The objective of this test case is to ensure that when a worker sends
			// no signals indicating that it handled any errors, that Run() handles
			// that case correctly.
			name: "success",
			listProjectsFn: func(
				_ context.Context,
				objList client.ObjectList,
				_ ...client.ListOption,
			) error {
				projects, ok := objList.(*corev1.NamespaceList)
				require.True(t, ok)
				projects.Items = []corev1.Namespace{{}}
				return nil
			},
			cleanProjectsFn: func(
				ctx context.Context,
				projectCh <-chan string,
				errCh chan<- struct{},
			) {
				// All we want to do is receive one Project name and return
				select {
				case <-projectCh:
				case <-ctx.Done():
					require.FailNow(t, "timed out waiting for a Project name")
				}
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c := &collector{
				cfg: CollectorConfig{
					NumWorkers: 1,
				},
				listProjectsFn:  testCase.listProjectsFn,
				cleanProjectsFn: testCase.cleanProjectsFn,
			}
			err := c.Run(ctx)
			testCase.assertions(err)
		})
	}
}

func TestCleanProjects(t *testing.T) {
	testCases := []struct {
		name           string
		cleanProjectFn func(ctx context.Context, project string) error
		errHandlerFn   func(ctx context.Context, errCh <-chan struct{})
	}{
		{
			// The objective of this test case is to ensure that errCh is signaled
			// when an error occurs.
			name: "error cleaning individual Project",
			cleanProjectFn: func(context.Context, string) error {
				return errors.New("something went wrong")
			},
			errHandlerFn: func(ctx context.Context, errCh <-chan struct{}) {
				select {
				case _, ok := <-errCh:
					if !ok {
						require.FailNow(
							t,
							"error channel was closed without receiving any signals",
						)
					}
				case <-ctx.Done():
					require.FailNow(
						t,
						"timed out without receiving an error signal",
					)
				}
			},
		},

		{
			// The objective of this test case is to ensure that errCh is NOT signaled
			// when everything goes smoothly.
			name: "success",
			cleanProjectFn: func(context.Context, string) error {
				return nil
			},
			errHandlerFn: func(ctx context.Context, errCh <-chan struct{}) {
				select {
				case _, ok := <-errCh:
					if ok {
						require.FailNow(t, "an unexpected error signal was received")
					}
				case <-ctx.Done():
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			c := &collector{
				cleanProjectFn: testCase.cleanProjectFn,
			}

			projectCh := make(chan string)
			errCh := make(chan struct{})

			go c.cleanProjects(ctx, projectCh, errCh)

			select {
			case projectCh <- "fake-project":
			case <-ctx.Done():
				require.FailNow(t, "timed out sending a Project name")
			}

			testCase.errHandlerFn(ctx, errCh)
		})
	}
}

func TestCleanProject(t *testing.T) {
	ctx := context.Background()
	logger := logging.LoggerFromContext(ctx)
	logger.Logger.Level = log.PanicLevel
	ctx = logging.ContextWithLogger(ctx, logger)

	testCases := []struct {
		name             string
		listPromotionsFn func(
			context.Context,
			client.ObjectList,
			...client.ListOption,
		) error
		deletePromotionFn func(
			context.Context,
			client.Object,
			...client.DeleteOption,
		) error
		assertions func(error)
	}{
		{
			name: "error listing Promotions",
			listPromotionsFn: func(
				context.Context,
				client.ObjectList,
				...client.ListOption,
			) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error listing Promotions for Project")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "fewer Promotions than max found",
			listPromotionsFn: func(
				_ context.Context,
				objList client.ObjectList,
				_ ...client.ListOption,
			) error {
				promos, ok := objList.(*api.PromotionList)
				require.True(t, ok)
				promos.Items = []api.Promotion{}
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},

		{
			name: "error deleting Promotion",
			listPromotionsFn: func(
				_ context.Context,
				objList client.ObjectList,
				_ ...client.ListOption,
			) error {
				promos, ok := objList.(*api.PromotionList)
				require.True(t, ok)
				promos.Items = []api.Promotion{}
				for i := 0; i < 100; i++ {
					promos.Items = append(
						promos.Items,
						api.Promotion{
							Status: api.PromotionStatus{
								Phase: api.PromotionPhaseComplete,
							},
						},
					)
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
			c := &collector{
				cfg: CollectorConfig{
					MaxRetainedPromotions: 20,
				},
				listPromotionsFn:  testCase.listPromotionsFn,
				deletePromotionFn: testCase.deletePromotionFn,
			}
			testCase.assertions(c.cleanProject(ctx, "fake-project"))
		})
	}

	t.Run("success", func(t *testing.T) {
		const numPromos = 100
		const testProject = "fake-project"

		scheme := runtime.NewScheme()
		err := api.AddToScheme(scheme)
		require.NoError(t, err)

		initialPromos := []client.Object{}
		creationTime := time.Now()
		for i := 0; i < numPromos; i++ {
			// We make each Promotion look newer then the last to ensure the sort
			// isn't a no-op and actually gets covered by this test
			creationTime = creationTime.Add(time.Hour)
			initialPromos = append(
				initialPromos,
				&api.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:              fmt.Sprintf("promotion-%d", i),
						Namespace:         testProject,
						CreationTimestamp: metav1.NewTime(creationTime),
					},
					Status: api.PromotionStatus{
						Phase: api.PromotionPhaseComplete,
					},
				},
			)
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

		err = c.cleanProject(ctx, testProject)
		require.NoError(t, err)

		promos := api.PromotionList{}
		err = kubeClient.List(
			ctx,
			&promos,
			client.InNamespace(testProject),
		)
		require.NoError(t, err)
		require.Len(t, promos.Items, c.cfg.MaxRetainedPromotions)
	})
}
