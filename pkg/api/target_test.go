package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetTarget(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.Target, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Nil(t, target)
			},
		},
		{
			name: "error getting Target",
			client: fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return errors.New("something went wrong")
					},
				},
			).Build(),
			assertions: func(t *testing.T, target *kargoapi.Target, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.ErrorContains(t, err, "error getting Target")
				require.Nil(t, target)
			},
		},
		{
			name: "success",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Target{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-target",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, target *kargoapi.Target, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-target", target.Name)
				require.Equal(t, "fake-namespace", target.Namespace)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			target, err := GetTarget(
				context.Background(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-target",
				},
			)
			testCase.assertions(t, target, err)
		})
	}
}
