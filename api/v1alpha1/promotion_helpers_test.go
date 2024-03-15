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
