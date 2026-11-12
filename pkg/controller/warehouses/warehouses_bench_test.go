package warehouses

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/subscription"
)

// BenchmarkReconcile covers one scheduled discovery of a Warehouse with Git
// subscriptions: artifacts unchanged since the last poll (Freight already
// exists) and new artifacts (Freight created). patches/op counts status
// patches sent to the API server. project-enqueues/op counts changes to the
// Healthy condition, each of which enqueues the Warehouse's Project for a
// stats recompute.
func BenchmarkReconcile(b *testing.B) {
	scheme := runtime.NewScheme()
	require.NoError(b, kargoapi.AddToScheme(scheme))
	for _, newArtifacts := range []bool{false, true} {
		for _, subs := range []int{1, 5, 20} {
			name := "unchanged"
			if newArtifacts {
				name = "new"
			}
			b.Run(fmt.Sprintf("%s/subscriptions-%d", name, subs), func(b *testing.B) {
				warehouse := &kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:  "fake-project",
						Name:       "fake-warehouse",
						Generation: 1,
					},
					Spec: kargoapi.WarehouseSpec{
						Interval: metav1.Duration{Duration: time.Second},
					},
				}
				for i := range subs {
					warehouse.Spec.InternalSubscriptions = append(
						warehouse.Spec.InternalSubscriptions,
						kargoapi.RepoSubscription{Git: &kargoapi.GitSubscription{
							RepoURL: fmt.Sprintf("https://example.com/repo-%d.git", i),
						}},
					)
				}
				var patches, enqueues int
				var healthy metav1.ConditionStatus // "" while absent
				c := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(warehouse).
					WithStatusSubresource(&kargoapi.Warehouse{}).
					WithInterceptorFuncs(interceptor.Funcs{
						SubResourcePatch: func(
							ctx context.Context,
							c client.Client,
							subResourceName string,
							obj client.Object,
							patch client.Patch,
							opts ...client.SubResourcePatchOption,
						) error {
							patches++
							if err := c.SubResource(subResourceName).Patch(ctx, obj, patch, opts...); err != nil {
								return err
							}
							wh, ok := obj.(*kargoapi.Warehouse)
							if !ok {
								return nil
							}
							var h metav1.ConditionStatus
							if cond := conditions.Get(&wh.Status, kargoapi.ConditionTypeHealthy); cond != nil {
								h = cond.Status
							}
							if h != healthy {
								enqueues++
								healthy = h
							}
							return nil
						},
					}).
					Build()
				r := newReconciler(
					c,
					&credentials.FakeDB{},
					subscription.MustNewSubscriberRegistry(),
					ReconcilerConfig{IsDefaultController: true},
					nil,
				)
				// Each poll is one interval after the last, all in the past, so
				// every Reconcile discovers and records a new discoveredAt.
				discoveredAt := time.Now().Add(-1000 * time.Hour)
				var poll int
				r.discoverArtifactsFn = func(
					context.Context, string,
					[]kargoapi.RepoSubscription,
					*kargoapi.DiscoveredArtifacts,
				) (*kargoapi.DiscoveredArtifacts, error) {
					poll++
					commit := "fake-commit"
					if newArtifacts {
						commit = fmt.Sprintf("%040d", poll)
					}
					artifacts := &kargoapi.DiscoveredArtifacts{
						DiscoveredAt: metav1.NewTime(discoveredAt.Add(time.Duration(poll) * time.Second)),
					}
					for i := range subs {
						artifacts.Git = append(artifacts.Git, kargoapi.GitDiscoveryResult{
							RepoURL: fmt.Sprintf("https://example.com/repo-%d.git", i),
							Commits: []kargoapi.DiscoveredCommit{{ID: commit}},
						})
					}
					return artifacts, nil
				}
				r.createFreightFn = func(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
					if newArtifacts {
						return nil
					}
					return apierrors.NewAlreadyExists(
						schema.GroupResource{Group: kargoapi.GroupVersion.Group, Resource: "freights"},
						obj.GetName(),
					)
				}
				req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(warehouse)}
				// The first poll moves the Warehouse from empty to steady-state status.
				_, err := r.Reconcile(b.Context(), req)
				require.NoError(b, err)
				patches, enqueues = 0, 0
				b.ReportAllocs()
				for b.Loop() {
					if _, err = r.Reconcile(b.Context(), req); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(patches)/float64(b.N), "patches/op")
				b.ReportMetric(float64(enqueues)/float64(b.N), "project-enqueues/op")
			})
		}
	}
}
