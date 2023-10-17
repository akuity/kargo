package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetWarehouse(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*Warehouse, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(warehouse *Warehouse, err error) {
				require.NoError(t, err)
				require.Nil(t, warehouse)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(warehouse *Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-warehouse", warehouse.Name)
				require.Equal(t, "fake-namespace", warehouse.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			warehouse, err := GetWarehouse(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-warehouse",
				},
			)
			testCase.assertions(warehouse, err)
		})
	}
}
