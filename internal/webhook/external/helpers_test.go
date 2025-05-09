package external

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
)

func TestRefresh(t *testing.T) {
	for _, test := range []struct {
		name           string
		newKubeClient  func() client.Client
		repoName       string
		expectedResult *refreshResult
		expectedErr    error
	}{
		{
			name: "OK",
			newKubeClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&kargoapi.Warehouse{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fakenamespace",
								Name:      "fakename",
							},
							Spec: kargoapi.WarehouseSpec{
								Subscriptions: []kargoapi.RepoSubscription{
									{
										Git: &kargoapi.GitSubscription{
											RepoURL: "https://github.com/username/repo",
										},
									},
								},
							},
						},
					).
					WithIndex(
						&kargoapi.Warehouse{},
						indexer.WarehousesBySubscribedURLsField,
						indexer.WarehousesBySubscribedURLs,
					).Build()
			},
			repoName: "https://github.com/username/repo",
			expectedResult: &refreshResult{
				successes: 1,
				failures:  0,
			},
			expectedErr: nil,
		},
		{
			name: "failed to list warehouses",
			newKubeClient: func() client.Client {
				// this will fail because kargo api
				// is not registered on runtime scheme
				return fake.NewClientBuilder().
					WithScheme(runtime.NewScheme()).Build()
			},
			expectedResult: nil,
			expectedErr: errors.New(
				"failed to list warehouses: no kind is registered for the type v1alpha1.WarehouseList in scheme",
			),
		},
		{
			name: "partial success",
			newKubeClient: func() client.Client {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&kargoapi.Warehouse{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "fakenamespace",
								Name:      "fakename",
							},
							Spec: kargoapi.WarehouseSpec{
								Subscriptions: []kargoapi.RepoSubscription{
									{
										Git: &kargoapi.GitSubscription{
											RepoURL: "https://github.com/username/repo",
										},
									},
								},
							},
						},
						&kargoapi.Warehouse{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "",
								Name:      "",
							},
							Spec: kargoapi.WarehouseSpec{
								Subscriptions: []kargoapi.RepoSubscription{
									{
										Git: &kargoapi.GitSubscription{
											RepoURL: "https://github.com/username/repo",
										},
									},
								},
							},
						},
					).
					WithIndex(
						&kargoapi.Warehouse{},
						indexer.WarehousesBySubscribedURLsField,
						indexer.WarehousesBySubscribedURLs,
					).Build()
			},
			repoName: "https://github.com/username/repo",
			expectedResult: &refreshResult{
				successes: 1,
				failures:  1,
			},
			expectedErr: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := refresh(
				t.Context(),
				test.newKubeClient(),
				test.repoName,
			)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t,
					err.Error(),
					test.expectedErr.Error(),
				)
				return
			}
			require.NoError(t, err)
			require.Equal(t,
				test.expectedResult.failures,
				result.failures,
			)
			require.Equal(t,
				test.expectedResult.successes,
				result.successes,
			)
		})
	}
}
