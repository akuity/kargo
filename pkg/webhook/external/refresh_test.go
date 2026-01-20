package external

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/urls"
)

func TestRefreshWarehouses(t *testing.T) {
	// Callers are responsible for normalizing the repository URL.
	testRepoURL := urls.NormalizeGit("https://github.com/example/repo.git")
	testRepoURLs := []string{testRepoURL}

	const testProject = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		client     client.Client
		project    string
		assertions func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "error listing Warehouses",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(
				interceptor.Funcs{
					List: func(
						context.Context,
						client.WithWatch,
						client.ObjectList,
						...client.ListOption,
					) error {
						return errors.New("something went wrong")
					},
				},
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, "{}", rr.Body.String())
			},
		},
		{
			name: "partial success -- Project not specified",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-other-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ client.Patch,
					_ ...client.PatchOption,
				) error {
					if obj.GetNamespace() == "some-namespace" {
						return nil
					}
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, `{"error":"failed to refresh 1 of 2 warehouses"}`, rr.Body.String())
			},
		},
		{
			name: "complete success -- Project not specified",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-namespace",
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			name:    "partial success -- Project specified",
			project: testProject,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-other-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					_ context.Context,
					_ client.WithWatch,
					obj client.Object,
					_ client.Patch,
					_ ...client.PatchOption,
				) error {
					if obj.GetName() == "some-warehouse" {
						return nil
					}
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(t, `{"error":"failed to refresh 1 of 2 warehouses"}`, rr.Body.String())
			},
		},
		{
			name:    "complete success -- Project specified",
			project: testProject,
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "some-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{RepoURL: testRepoURL},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			refreshWarehouses(
				t.Context(),
				w,
				tt.client,
				tt.project,
				testRepoURLs,
			)
			tt.assertions(t, w)
		})
	}
}

func TestShouldRefresh(t *testing.T) {
	testCases := []struct {
		name       string
		wh         kargoapi.Warehouse
		qualifiers []string
		repoURL    string
		expect     bool
	}{
		{
			name: "Git subscription with matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
						},
					}},
				},
			},
			repoURL:    "https://github.com/username/repo",
			qualifiers: []string{"refs/heads/main"},
			expect:     true,
		},
		{
			name: "Git subscription with non-matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo.git",
							Branch:                  "main",
						},
					}},
				},
			},
			repoURL:    "https://github.com/username/repo",
			qualifiers: []string{"release"},
			expect:     false,
		},
		{
			name: "Image subscription with matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Image: &kargoapi.ImageSubscription{
							RepoURL:                "docker.io/example/repo",
							ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
							Constraint:             "^1.0.0",
							StrictSemvers:          ptr.To(true),
						},
					}},
				},
			},
			repoURL:    "example/repo",
			qualifiers: []string{"v1.0.0"},
			expect:     true,
		},
		{
			name: "Image subscription with non-matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Image: &kargoapi.ImageSubscription{
							RepoURL:                "docker.io/example/repo",
							ImageSelectionStrategy: kargoapi.ImageSelectionStrategySemVer,
							Constraint:             "^1.0.0",
							StrictSemvers:          ptr.To(true),
						},
					}},
				},
			},
			repoURL:    "docker.io/example/repo",
			qualifiers: []string{"invalid-tag"},
			expect:     false,
		},
		{
			name: "Chart subscription with matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Chart: &kargoapi.ChartSubscription{
							RepoURL:          "oci://example.com/charts",
							SemverConstraint: "^1.0.0",
						},
					}},
				},
			},
			repoURL:    "example.com/charts",
			qualifiers: []string{"v1.0.0"},
			expect:     true,
		},
		{
			name: "Chart subscription with non-matching qualifier",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Chart: &kargoapi.ChartSubscription{
							RepoURL:          "oci://example.com/charts",
							SemverConstraint: "^2.0.0",
						},
					}},
				},
			},
			repoURL:    "example.com/charts",
			qualifiers: []string{"1.0.0"},
			expect:     false,
		},
		{
			name:       "No subscriptions",
			wh:         kargoapi.Warehouse{},
			qualifiers: []string{"main"},
			expect:     false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := shouldRefresh(
				t.Context(),
				tc.wh,
				tc.repoURL,
				tc.qualifiers...,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expect, result)
		})
	}
}
