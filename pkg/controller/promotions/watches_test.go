package promotions

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
)

func TestUpdatedArgoCDAppHandler_Update(t *testing.T) {
	tests := []struct {
		name            string
		applications    []client.Object
		indexer         client.IndexerFunc
		selectorIndexer client.IndexerFunc
		interceptor     interceptor.Funcs
		e               event.TypedUpdateEvent[*argocd.Application]
		assertions      func(*testing.T, workqueue.TypedRateLimitingInterface[reconcile.Request])
	}{
		{
			name: "Event without new object",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Event without old object",
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectNew: &argocd.Application{},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
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
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
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
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
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
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
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
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "fake-application-namespace",
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Application matched by label selector enqueues the Promotion",
			applications: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-project",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "selector-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "fake-stage",
						Steps: []kargoapi.PromotionStep{{
							Uses: "argocd-update",
							Config: &apiextensionsv1.JSON{
								Raw: []byte(
									`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`,
								),
							},
						}},
					},
					Status: kargoapi.PromotionStatus{
						Phase:       kargoapi.PromotionPhaseRunning,
						CurrentStep: 0,
					},
				},
			},
			// Name-based index finds nothing for a selector-based step.
			indexer: func(client.Object) []string { return nil },
			selectorIndexer: func(obj client.Object) []string {
				if obj.GetName() == "selector-promotion" {
					return []string{indexer.RunningPromotionsByArgoCDSelectorsValue}
				}
				return nil
			},
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "argocd",
						Labels:    map[string]string{"app": "foo"},
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				require.Equal(t, 1, wq.Len())
				item, _ := wq.Get()
				require.Equal(t, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "fake-project",
						Name:      "selector-promotion",
					},
				}, item)
			},
		},
		{
			name: "Application not matching label selector is not enqueued",
			applications: []client.Object{
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-stage",
						Namespace: "fake-project",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "selector-promotion",
						Namespace: "fake-project",
					},
					Spec: kargoapi.PromotionSpec{
						Stage: "fake-stage",
						Steps: []kargoapi.PromotionStep{{
							Uses: "argocd-update",
							Config: &apiextensionsv1.JSON{
								Raw: []byte(
									`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`,
								),
							},
						}},
					},
					Status: kargoapi.PromotionStatus{
						Phase:       kargoapi.PromotionPhaseRunning,
						CurrentStep: 0,
					},
				},
			},
			indexer: func(client.Object) []string { return nil },
			selectorIndexer: func(obj client.Object) []string {
				if obj.GetName() == "selector-promotion" {
					return []string{indexer.RunningPromotionsByArgoCDSelectorsValue}
				}
				return nil
			},
			e: event.TypedUpdateEvent[*argocd.Application]{
				ObjectOld: &argocd.Application{},
				ObjectNew: &argocd.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-application-name",
						Namespace: "argocd",
						Labels:    map[string]string{"app": "bar"},
					},
				},
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name:         "Application relabeled out of selector still enqueues (old labels matched)",
			applications: selectorTestObjects(),
			indexer:      func(client.Object) []string { return nil },
			selectorIndexer: func(obj client.Object) []string {
				if obj.GetName() == "selector-promotion" {
					return []string{indexer.RunningPromotionsByArgoCDSelectorsValue}
				}
				return nil
			},
			e: event.TypedUpdateEvent[*argocd.Application]{
				// Old labels match the selector; new labels do not. The Promotion
				// must still be woken so it re-evaluates the shrunken match set.
				ObjectOld: matchingSelectorApp(),
				ObjectNew: nonMatchingSelectorApp(),
			},
			assertions: func(t *testing.T, wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				require.Equal(t, 1, wq.Len())
				item, _ := wq.Get()
				require.Equal(t, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "fake-project",
						Name:      "selector-promotion",
					},
				}, item)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			selectorIndexer := tt.selectorIndexer
			if selectorIndexer == nil {
				selectorIndexer = func(client.Object) []string { return nil }
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.applications...).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.RunningPromotionsByArgoCDApplicationsField,
					tt.indexer,
				).
				WithIndex(
					&kargoapi.Promotion{},
					indexer.RunningPromotionsByArgoCDSelectorsField,
					selectorIndexer,
				).
				WithInterceptorFuncs(tt.interceptor)

			u := &UpdatedArgoCDAppHandler[*argocd.Application]{
				kargoClient: c.Build(),
			}

			wq := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			u.Update(t.Context(), tt.e, wq)

			tt.assertions(t, wq)
		})
	}
}

func TestPromotionAcknowledgedByStageHandler_Update(t *testing.T) {
	testCases := []struct {
		name           string
		shardPredicate controller.ResponsibleFor[kargoapi.Stage]
		e              event.TypedUpdateEvent[*kargoapi.Stage]
		assertions     func(
			*testing.T,
			workqueue.TypedRateLimitingInterface[reconcile.Request],
		)
	}{
		{
			name: "Event without new object",
			e: event.TypedUpdateEvent[*kargoapi.Stage]{
				ObjectOld: &kargoapi.Stage{},
			},
			assertions: func(
				t *testing.T,
				wq workqueue.TypedRateLimitingInterface[reconcile.Request],
			) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Event without old object",
			e: event.TypedUpdateEvent[*kargoapi.Stage]{
				ObjectNew: &kargoapi.Stage{},
			},
			assertions: func(
				t *testing.T,
				wq workqueue.TypedRateLimitingInterface[reconcile.Request],
			) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Stage does not belong to shard",
			shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
				IsDefaultController: false,
				ShardName:           "fake-shard",
			},
			// This event would result in a Promotion being added to the work queue if
			// it were not for the shard mismatch.
			e: event.TypedUpdateEvent[*kargoapi.Stage]{
				ObjectOld: &kargoapi.Stage{
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "one-promotion",
						},
					},
				},
				ObjectNew: &kargoapi.Stage{
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "another-promotion",
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				wq workqueue.TypedRateLimitingInterface[reconcile.Request],
			) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "no current promotion",
			shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
				IsDefaultController: true,
				ShardName:           "fake-shard",
			},
			e: event.TypedUpdateEvent[*kargoapi.Stage]{
				ObjectOld: &kargoapi.Stage{},
				ObjectNew: &kargoapi.Stage{},
			},
			assertions: func(
				t *testing.T,
				wq workqueue.TypedRateLimitingInterface[reconcile.Request],
			) {
				require.Equal(t, 0, wq.Len())
			},
		},
		{
			name: "Promotion is enqueued for reconciliation",
			shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
				IsDefaultController: true,
				ShardName:           "fake-shard",
			},
			e: event.TypedUpdateEvent[*kargoapi.Stage]{
				ObjectOld: &kargoapi.Stage{
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "one-promotion",
						},
					},
				},
				ObjectNew: &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "fake-project",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "another-promotion",
						},
					},
				},
			},
			assertions: func(
				t *testing.T,
				wq workqueue.TypedRateLimitingInterface[reconcile.Request],
			) {
				require.Equal(t, 1, wq.Len())
				item, _ := wq.Get()
				require.Equal(t, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "fake-project",
						Name:      "another-promotion",
					},
				}, item)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := &PromotionAcknowledgedByStageHandler[*kargoapi.Stage]{
				shardPredicate: testCase.shardPredicate,
			}

			wq := workqueue.NewTypedRateLimitingQueue(
				workqueue.DefaultTypedControllerRateLimiter[reconcile.Request](),
			)
			p.Update(t.Context(), testCase.e, wq)

			testCase.assertions(t, wq)
		})
	}
}

// selectorTestObjects returns a Stage and a running, selector-based Promotion
// (argocd-update with matchLabels app=foo, namespace argocd) for exercising the
// selector-driven enqueue paths.
func selectorTestObjects() []client.Object {
	return []client.Object{
		&kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{Name: "fake-stage", Namespace: "fake-project"},
		},
		&kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{Name: "selector-promotion", Namespace: "fake-project"},
			Spec: kargoapi.PromotionSpec{
				Stage: "fake-stage",
				Steps: []kargoapi.PromotionStep{{
					Uses: "argocd-update",
					Config: &apiextensionsv1.JSON{
						Raw: []byte(
							`{"apps":[{"namespace":"argocd","selector":{"matchLabels":{"app":"foo"}}}]}`,
						),
					},
				}},
			},
			Status: kargoapi.PromotionStatus{
				Phase:       kargoapi.PromotionPhaseRunning,
				CurrentStep: 0,
			},
		},
	}
}

func matchingSelectorApp() *argocd.Application {
	return &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-application-name",
			Namespace: "argocd",
			Labels:    map[string]string{"app": "foo"},
		},
	}
}

func nonMatchingSelectorApp() *argocd.Application {
	return &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-application-name",
			Namespace: "argocd",
			Labels:    map[string]string{"app": "bar"},
		},
	}
}

// newSelectorAppHandler builds an UpdatedArgoCDAppHandler whose client indexes
// "selector-promotion" into the coarse selector index and finds nothing in the
// name-based index.
func newSelectorAppHandler(
	t *testing.T,
	objs ...client.Object,
) *UpdatedArgoCDAppHandler[*argocd.Application] {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithIndex(
			&kargoapi.Promotion{},
			indexer.RunningPromotionsByArgoCDApplicationsField,
			func(client.Object) []string { return nil },
		).
		WithIndex(
			&kargoapi.Promotion{},
			indexer.RunningPromotionsByArgoCDSelectorsField,
			func(obj client.Object) []string {
				if obj.GetName() == "selector-promotion" {
					return []string{indexer.RunningPromotionsByArgoCDSelectorsValue}
				}
				return nil
			},
		).
		Build()
	return &UpdatedArgoCDAppHandler[*argocd.Application]{kargoClient: c}
}

func newTestWorkqueue() workqueue.TypedRateLimitingInterface[reconcile.Request] {
	return workqueue.NewTypedRateLimitingQueue(
		workqueue.DefaultTypedControllerRateLimiter[reconcile.Request](),
	)
}

func requireSelectorPromotionEnqueued(
	t *testing.T,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	t.Helper()
	require.Equal(t, 1, wq.Len())
	item, _ := wq.Get()
	require.Equal(t, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "fake-project",
			Name:      "selector-promotion",
		},
	}, item)
}

func TestUpdatedArgoCDAppHandler_Create(t *testing.T) {
	t.Run("matching Application enqueues selector Promotion", func(t *testing.T) {
		u := newSelectorAppHandler(t, selectorTestObjects()...)
		wq := newTestWorkqueue()
		u.Create(
			t.Context(),
			event.TypedCreateEvent[*argocd.Application]{Object: matchingSelectorApp()},
			wq,
		)
		requireSelectorPromotionEnqueued(t, wq)
	})

	t.Run("non-matching Application does not enqueue", func(t *testing.T) {
		u := newSelectorAppHandler(t, selectorTestObjects()...)
		wq := newTestWorkqueue()
		u.Create(
			t.Context(),
			event.TypedCreateEvent[*argocd.Application]{Object: nonMatchingSelectorApp()},
			wq,
		)
		require.Equal(t, 0, wq.Len())
	})

	t.Run("nil object is a no-op", func(t *testing.T) {
		u := newSelectorAppHandler(t, selectorTestObjects()...)
		wq := newTestWorkqueue()
		u.Create(t.Context(), event.TypedCreateEvent[*argocd.Application]{}, wq)
		require.Equal(t, 0, wq.Len())
	})
}

func TestUpdatedArgoCDAppHandler_Delete(t *testing.T) {
	t.Run("matching Application enqueues selector Promotion", func(t *testing.T) {
		u := newSelectorAppHandler(t, selectorTestObjects()...)
		wq := newTestWorkqueue()
		u.Delete(
			t.Context(),
			event.TypedDeleteEvent[*argocd.Application]{Object: matchingSelectorApp()},
			wq,
		)
		requireSelectorPromotionEnqueued(t, wq)
	})

	t.Run("non-matching Application does not enqueue", func(t *testing.T) {
		u := newSelectorAppHandler(t, selectorTestObjects()...)
		wq := newTestWorkqueue()
		u.Delete(
			t.Context(),
			event.TypedDeleteEvent[*argocd.Application]{Object: nonMatchingSelectorApp()},
			wq,
		)
		require.Equal(t, 0, wq.Len())
	})

	t.Run("nil object is a no-op", func(t *testing.T) {
		u := newSelectorAppHandler(t, selectorTestObjects()...)
		wq := newTestWorkqueue()
		u.Delete(t.Context(), event.TypedDeleteEvent[*argocd.Application]{}, wq)
		require.Equal(t, 0, wq.Len())
	})
}
