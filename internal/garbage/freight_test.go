package garbage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestCleanProjectFreight(t *testing.T) {
	testCases := []struct {
		name       string
		collector  *collector
		assertions func(*testing.T, error)
	}{
		{
			name: "error listing Warehouses",
			collector: &collector{
				listWarehousesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Warehouses in Project")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error cleaning Warehouse Freight",
			collector: &collector{
				listWarehousesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					warehouses, ok := objList.(*kargoapi.WarehouseList)
					require.True(t, ok)
					warehouses.Items = []kargoapi.Warehouse{{}}
					return nil
				},
				cleanWarehouseFreightFn: func(context.Context, string, string) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "error cleaning Freight from one or more Warehouses",
				)
			},
		},
		{
			name: "success",
			collector: &collector{
				listWarehousesFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					warehouses, ok := objList.(*kargoapi.WarehouseList)
					require.True(t, ok)
					warehouses.Items = []kargoapi.Warehouse{{}}
					return nil
				},
				cleanWarehouseFreightFn: func(context.Context, string, string) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.collector.cleanProjectFreight(
					context.Background(),
					"fake-project",
				),
			)
		})
	}
}

func TestCleanWarehouseFreight(t *testing.T) {
	testCases := []struct {
		name       string
		collector  *collector
		assertions func(*testing.T, error)
	}{
		{
			name: "error listing Freight",
			collector: &collector{
				listFreightFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Freight from Warehouse")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "fewer Freight than threshold",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedFreight: 2,
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}}
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error listing Stages",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedFreight: 1,
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					freight.Items = []kargoapi.Freight{{}, {}}
					return nil
				},
				listStagesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "error listing Stages in Project")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "error deleting Freight",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedFreight:    1,
					MinFreightDeletionAge: time.Minute,
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					now := metav1.Now()
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
							},
						},
					}
					return nil
				},
				listStagesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					// This will appear that no Freight are in use
					return nil
				},
				deleteFreightFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "error deleting one or more Freight from Warehouse",
				)
			},
		},
		{
			name: "success",
			collector: &collector{
				cfg: CollectorConfig{
					MaxRetainedFreight:    1,
					MinFreightDeletionAge: time.Minute,
				},
				listFreightFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					freight, ok := objList.(*kargoapi.FreightList)
					require.True(t, ok)
					now := metav1.Now()
					freight.Items = []kargoapi.Freight{
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
							},
						},
					}
					return nil
				},
				listStagesFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					// This will appear that no Freight are in use
					return nil
				},
				deleteFreightFn: func(
					context.Context,
					client.Object,
					...client.DeleteOption,
				) error {
					return nil
				},
			},
			assertions: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.collector.cleanWarehouseFreight(
					context.Background(),
					"fake-project",
					"fake-warehouse",
				),
			)
		})
	}
}
