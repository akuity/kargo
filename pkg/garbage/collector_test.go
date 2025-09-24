package garbage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewCollector(t *testing.T) {
	watchClient := fake.NewClientBuilder().Build()
	testCfg := CollectorConfigFromEnv()
	c, ok := NewCollector(watchClient, testCfg).(*collector)
	require.True(t, ok)
	require.Equal(t, testCfg, c.cfg)
	require.NotNil(t, c.cleanProjectsFn)
	require.NotNil(t, c.cleanProjectFn)
	require.NotNil(t, c.cleanProjectPromotionsFn)
	require.NotNil(t, c.cleanStagePromotionsFn)
	require.NotNil(t, c.listProjectsFn)
	require.NotNil(t, c.listPromotionsFn)
	require.NotNil(t, c.deletePromotionFn)
	require.NotNil(t, c.cleanProjectFreightFn)
	require.NotNil(t, c.listWarehousesFn)
	require.NotNil(t, c.cleanWarehouseFreightFn)
	require.NotNil(t, c.listFreightFn)
	require.NotNil(t, c.listStagesFn)
	require.NotNil(t, c.deleteFreightFn)
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
		assertions func(*testing.T, error)
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
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(
					t, err, "error listing projects; no garbage collection performed",
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
			assertions: func(t *testing.T, err error) {
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
				_ chan<- struct{},
			) {
				// All we want to do is receive one Project name and return
				select {
				case <-projectCh:
				case <-ctx.Done():
					require.FailNow(t, "timed out waiting for a Project name")
				}
			},
			assertions: func(t *testing.T, err error) {
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
			testCase.assertions(t, err)
		})
	}
}
