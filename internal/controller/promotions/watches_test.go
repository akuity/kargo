package promotions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
)

func TestUpdatedArgoCDAppHandler_Update(t *testing.T) {
	tests := []struct {
		name          string
		applications  []client.Object
		indexer       client.IndexerFunc
		interceptor   interceptor.Funcs
		shardSelector labels.Selector
		e             event.UpdateEvent
		assertions    func(*testing.T, workqueue.RateLimitingInterface)
	}{
		{
			name: "Event without new object",
			e: event.UpdateEvent{
				ObjectOld: &argocd.Application{},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Event without old object",
			e: event.UpdateEvent{
				ObjectNew: &argocd.Application{},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Event object has indexed Promotion",
			applications: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-promotion",
						Namespace: "fake-namespace",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "matching-promotion",
						Namespace: "fake-namespace",
					},
				},
			},
			indexer: func(obj client.Object) []string {
				if obj.GetName() == "matching-promotion" {
					return []string{"fake-application-namespace:fake-application-name"}
				}
				return nil
			},
			e: event.UpdateEvent{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 1, wq.Len())

				item, _ := wq.Get()
				require.Equal(t, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "fake-namespace",
						Name:      "matching-promotion",
					},
				}, item)
			},
		},
		{
			name: "Event object has multiple indexed Promotions",
			applications: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-promotion",
						Namespace: "fake-namespace",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "matching-promotion-1",
						Namespace: "fake-namespace",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "matching-promotion-2",
						Namespace: "other-namespace",
					},
				},
			},
			indexer: func(obj client.Object) []string {
				if strings.HasPrefix(obj.GetName(), "matching-promotion") {
					return []string{"fake-application-namespace:fake-application-name"}
				}
				return nil
			},
			e: event.UpdateEvent{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 2, wq.Len())

				var items []any
				for i := 0; i <= wq.Len(); i++ {
					item, _ := wq.Get()
					items = append(items, item)
				}

				require.ElementsMatch(
					t,
					items,
					[]reconcile.Request{
						{
							NamespacedName: types.NamespacedName{
								Namespace: "fake-namespace",
								Name:      "matching-promotion-1",
							},
						},
						{
							NamespacedName: types.NamespacedName{
								Namespace: "other-namespace",
								Name:      "matching-promotion-2",
							},
						},
					},
				)
			},
		},
		{
			name: "Event object with indexed Promotion and shard selector",
			applications: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "matching-promotion",
						Namespace: "fake-namespace",
						Labels: map[string]string{
							"shard-label": "shard-value",
						},
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "matching-promotion-without-shard-label",
						Namespace: "fake-namespace",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-promotion-with-shard-label",
						Namespace: "fake-namespace",
						Labels: map[string]string{
							"shard-label": "other-shard-value",
						},
					},
				},
			},
			indexer: func(obj client.Object) []string {
				if strings.HasPrefix(obj.GetName(), "matching-promotion") {
					return []string{"fake-application-namespace:fake-application-name"}
				}
				return nil
			},
			shardSelector: labels.SelectorFromSet(labels.Set{
				"shard-label": "shard-value",
			}),
			e: event.UpdateEvent{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 1, wq.Len())

				item, _ := wq.Get()
				require.Equal(t, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "fake-namespace",
						Name:      "matching-promotion",
					},
				}, item)
			},
		},
		{
			name: "Event object has no indexed Promotion",
			applications: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-promotion",
						Namespace: "fake-namespace",
					},
				},
			},
			indexer: func(client.Object) []string {
				return nil
			},
			e: event.UpdateEvent{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Promotions list error",
			applications: []client.Object{
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "matching-promotion",
						Namespace: "fake-namespace",
					},
				},
			},
			indexer: func(client.Object) []string {
				return []string{"fake-application-namespace:fake-application-name"}
			},
			interceptor: interceptor.Funcs{
				List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
					return apierrors.NewInternalError(errors.New("something went wrong"))
				},
			},
			e: event.UpdateEvent{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.RateLimitingInterface) {
				require.Equal(t, 0, wq.Len())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.applications...).
				WithIndex(
					&kargoapi.Promotion{},
					kubeclient.RunningPromotionsByArgoCDApplicationsIndexField,
					tt.indexer,
				).
				WithInterceptorFuncs(tt.interceptor)

			u := &UpdatedArgoCDAppHandler{
				kargoClient:   c.Build(),
				shardSelector: tt.shardSelector,
			}

			wq := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			u.Update(context.TODO(), tt.e, wq)

			tt.assertions(t, wq)
		})
	}
}
