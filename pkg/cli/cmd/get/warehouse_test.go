package get

import (
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_repairInboundWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, *kargoapi.Warehouse, error)
	}{
		{
			name: "clears external and creates internal",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []apiextensionsv1.JSON{
						{Raw: []byte(`{"git":{}}`)},
						{Raw: []byte(`{"image":{}}`)},
						{Raw: []byte(`{"chart":{}}`)},
						{Raw: []byte(`{"generic":{}}`)},
					},
				},
			},
			assertions: func(t *testing.T, w *kargoapi.Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, 0, len(w.Spec.Subscriptions))
				require.Greater(t, len(w.Spec.InternalSubscriptions), 0)
			},
		},
		{
			name: "empty spec no changes",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{},
			},
			assertions: func(t *testing.T, w *kargoapi.Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, 0, len(w.Spec.InternalSubscriptions))
				require.Equal(t, 0, len(w.Spec.Subscriptions))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repairInboundWarehouse(tc.warehouse)
			tc.assertions(t, tc.warehouse, err)
		})
	}
}
