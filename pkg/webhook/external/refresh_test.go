package external

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/urls"
)

func TestRefreshWarehouses(t *testing.T) {
	// Callers are responsible for normalizing the repository URL.
	testRepoURL := urls.NormalizeGit("https://github.com/example/repo.git")
	testRepoURLs := []string{testRepoURL}
	testOtherRepoURL := urls.NormalizeGit("https://github.com/example/other-repo.git")

	const testProject = "fake-project"

	testScheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	testCases := []struct {
		name       string
		client     client.Client
		project    string
		repoURLs   []string // Defaults to testRepoURLs
		qualifiers []string
		// skipped is the expected skipped-Warehouse count by Project.
		skipped    map[string]float64
		assertions func(*testing.T, client.Client, *httptest.ResponseRecorder)
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
			assertions: func(t *testing.T, _ client.Client, rr *httptest.ResponseRecorder) {
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
			assertions: func(t *testing.T, _ client.Client, rr *httptest.ResponseRecorder) {
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
			assertions: func(t *testing.T, _ client.Client, rr *httptest.ResponseRecorder) {
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
			assertions: func(t *testing.T, _ client.Client, rr *httptest.ResponseRecorder) {
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
			assertions: func(t *testing.T, _ client.Client, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(t, `{"msg":"refreshed 1 warehouse(s)"}`, rr.Body.String())
			},
		},
		{
			// A Warehouse whose selector can't be built must not prevent other
			// Warehouses subscribed to the same repository from being refreshed.
			name:       "Warehouse with invalid selector is skipped",
			skipped:    map[string]float64{testProject: 1},
			project:    testProject,
			qualifiers: []string{"refs/heads/main"},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						// Sorts first, so it is evaluated first
						Name: "bad-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL:          testRepoURL,
								Branch:           "main",
								ExpressionFilter: "(1 + 2",
							},
						}},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "good-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: testRepoURL,
								Branch:  "main",
							},
						}},
					},
				},
			).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, c client.Client, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				require.JSONEq(
					t,
					`{"msg":"refreshed 1 warehouse(s); skipped 1 with an invalid subscription"}`,
					rr.Body.String(),
				)
				for name, refreshed := range map[string]bool{
					"bad-warehouse":  false,
					"good-warehouse": true,
				} {
					wh := &kargoapi.Warehouse{}
					require.NoError(t, c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject, Name: name},
						wh,
					))
					_, ok := api.RefreshAnnotationValue(wh.GetAnnotations())
					require.Equal(t, refreshed, ok, name)
				}
			},
		},
		{
			// A Warehouse subscribed to more than one of the repositories is
			// skipped, and counted, only once. Skips are also reported when
			// refreshing another Warehouse fails.
			name:       "skipped Warehouse counted once and reported on failure",
			skipped:    map[string]float64{testProject: 1},
			project:    testProject,
			repoURLs:   []string{testRepoURL, testOtherRepoURL},
			qualifiers: []string{"refs/heads/main"},
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "bad-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{
							{
								Git: &kargoapi.GitSubscription{
									RepoURL:          testRepoURL,
									Branch:           "main",
									ExpressionFilter: "(1 + 2",
								},
							},
							{
								Git: &kargoapi.GitSubscription{
									RepoURL:          testOtherRepoURL,
									Branch:           "main",
									ExpressionFilter: "(1 + 2",
								},
							},
						},
					},
				},
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      "good-warehouse",
					},
					Spec: kargoapi.WarehouseSpec{
						InternalSubscriptions: []kargoapi.RepoSubscription{{
							Git: &kargoapi.GitSubscription{
								RepoURL: testRepoURL,
								Branch:  "main",
							},
						}},
					},
				},
			).WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(
					context.Context,
					client.WithWatch,
					client.Object,
					client.Patch,
					...client.PatchOption,
				) error {
					return errors.New("something went wrong")
				},
			}).WithIndex(
				&kargoapi.Warehouse{},
				indexer.WarehousesBySubscribedURLsField,
				indexer.WarehousesBySubscribedURLs,
			).Build(),
			assertions: func(t *testing.T, _ client.Client, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
				require.JSONEq(
					t,
					`{"error":"failed to refresh 1 of 1 warehouses; skipped 1 with an invalid subscription"}`,
					rr.Body.String(),
				)
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			repoURLs := tt.repoURLs
			if repoURLs == nil {
				repoURLs = testRepoURLs
			}
			counter := prometheus.NewCounterVec(
				prometheus.CounterOpts{Name: "test_warehouses_skipped_total"},
				[]string{"project"},
			)
			saved := warehousesSkipped
			warehousesSkipped = counter
			t.Cleanup(func() { warehousesSkipped = saved })
			w := httptest.NewRecorder()
			refreshWarehouses(
				t.Context(),
				w,
				tt.client,
				tt.project,
				repoURLs,
				nil,
				tt.qualifiers...,
			)
			tt.assertions(t, tt.client, w)
			require.Equal(t, len(tt.skipped), testutil.CollectAndCount(counter))
			for project, value := range tt.skipped {
				require.Equal(t, value, testutil.ToFloat64(counter.WithLabelValues(project)))
			}
		})
	}
}

func TestShouldRefresh(t *testing.T) {
	testCases := []struct {
		name         string
		wh           kargoapi.Warehouse
		qualifiers   []string
		repoURL      string
		changedFiles []string
		expect       bool
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
			name: "Git subscription with matching ref and matching includePaths",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
							IncludePaths:            []string{"apps/foo"},
						},
					}},
				},
			},
			repoURL:      "https://github.com/username/repo",
			qualifiers:   []string{"refs/heads/main"},
			changedFiles: []string{"apps/foo/values.yaml"},
			expect:       true,
		},
		{
			name: "Git subscription with matching ref but non-matching includePaths",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
							IncludePaths:            []string{"apps/foo"},
						},
					}},
				},
			},
			repoURL:      "https://github.com/username/repo",
			qualifiers:   []string{"refs/heads/main"},
			changedFiles: []string{"apps/bar/values.yaml"},
			expect:       false,
		},
		{
			name: "Git subscription with matching ref and excluded path",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
							ExcludePaths:            []string{"docs"},
						},
					}},
				},
			},
			repoURL:      "https://github.com/username/repo",
			qualifiers:   []string{"refs/heads/main"},
			changedFiles: []string{"docs/README.md"},
			expect:       false,
		},
		{
			name: "Git subscription with matching ref and no path filters -- always refreshes",
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
			repoURL:      "https://github.com/username/repo",
			qualifiers:   []string{"refs/heads/main"},
			changedFiles: []string{"anything/file.txt"},
			expect:       true,
		},
		{
			name: "Git subscription with matching ref and path filters but no changed files -- filtering skipped, refreshes",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
							IncludePaths:            []string{"apps/foo"},
						},
					}},
				},
			},
			repoURL:    "https://github.com/username/repo",
			qualifiers: []string{"refs/heads/main"},
			expect:     true,
		},
		{
			name: "Git subscription with include and exclude paths -- include wins then exclude filters",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
							IncludePaths:            []string{"apps/"},
							ExcludePaths:            []string{"apps/ignored"},
						},
					}},
				},
			},
			repoURL:      "https://github.com/username/repo",
			qualifiers:   []string{"refs/heads/main"},
			changedFiles: []string{"apps/ignored/config.yaml"},
			expect:       false,
		},
		{
			name: "Git subscription with include and exclude -- non-excluded file matches",
			wh: kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{{
						Git: &kargoapi.GitSubscription{
							CommitSelectionStrategy: kargoapi.CommitSelectionStrategyNewestFromBranch,
							RepoURL:                 "https://github.com/username/repo",
							Branch:                  "main",
							IncludePaths:            []string{"apps/"},
							ExcludePaths:            []string{"apps/ignored"},
						},
					}},
				},
			},
			repoURL:      "https://github.com/username/repo",
			qualifiers:   []string{"refs/heads/main"},
			changedFiles: []string{"apps/important/config.yaml"},
			expect:       true,
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
				tc.changedFiles,
				tc.qualifiers...,
			)
			require.NoError(t, err)
			require.Equal(t, tc.expect, result)
		})
	}
}
