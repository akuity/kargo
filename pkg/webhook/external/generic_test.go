package external

import (
	"testing"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/urls"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_newListOptionsForIndexSelector(t *testing.T) {
	// This is indirectly tested via Test_buildListOptionsForTarget,
	// so we just need a basic smoke test here.
	testRepoURL := urls.NormalizeGit("https://github.com/example/repo.git")
	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	tests := []struct {
		name      string
		kClient   client.Client
		selector  kargoapi.IndexSelector
		env       map[string]any
		expectErr bool
	}{
		{
			name: "Equal selector satisfied",
			kClient: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			selector: kargoapi.IndexSelector{
				MatchExpressions: []kargoapi.IndexSelectorRequirement{
					{
						Key:      indexer.WarehousesBySubscribedURLsField,
						Operator: kargoapi.IndexSelectorRequirementOperatorEqual,
						Value:    "${{ request.body.repository.clone_url }}",
					},
				},
			},
			env: map[string]any{
				"request": map[string]any{
					"body": map[string]any{
						"repository": map[string]any{
							"clone_url": testRepoURL,
						},
					},
				},
			},
			expectErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listOpts, err := newListOptionsForIndexSelector(tt.selector, tt.env)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			whList := new(kargoapi.WarehouseList)
			err = tt.kClient.List(t.Context(), whList, listOpts...)
			require.NoError(t, err)
			require.Len(t, whList.Items, 1)
		})
	}
}
