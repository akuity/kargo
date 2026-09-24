package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

func authorizeStagesPromoteFn(t *testing.T) func(
	context.Context,
	string,
	schema.GroupVersionResource,
	string,
	client.ObjectKey,
) error {
	return func(
		_ context.Context,
		verb string,
		gvr schema.GroupVersionResource,
		_ string,
		_ client.ObjectKey,
	) error {
		switch verb {
		case "promote":
			require.Equal(t, kargoapi.GroupVersion.WithResource("stages"), gvr)
		case "create":
			require.Equal(t, kargoapi.GroupVersion.WithResource("promotions"), gvr)
		default:
			require.Failf(t, "unexpected authorization", "verb %q", verb)
		}
		return nil
	}
}

// authorizeAllStagesPromote grants the promote/create checks used by successful
// promotion creation paths.
func authorizeAllStagesPromote(t *testing.T, s *server) {
	s.authorizeFn = authorizeStagesPromoteFn(t)
}

func Test_server_promoteToStage(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: testWarehouse.Name,
		},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: testProject.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: kargoapi.FreightOrigin{
						Kind: kargoapi.FreightOriginKindWarehouse,
						Name: testWarehouse.Name,
					},
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
			PromotionTemplate: &kargoapi.PromotionTemplate{
				Spec: kargoapi.PromotionTemplateSpec{
					Steps: []kargoapi.PromotionStep{
						{Uses: "fake-step"},
					},
				},
			},
		},
	}

	// Same name as testStage, so that it can stand in for it at the endpoint
	// under test, but governing Targets rather than promoting to itself.
	testTargetAwareStage := testStage.DeepCopy()
	testTargetAwareStage.Spec.Targets = &kargoapi.StageTargets{
		Selectors: []metav1.LabelSelector{{
			MatchLabels: map[string]string{"region": "us"},
		}},
	}
	// Targets live in the database. eu-west is present in the Project but not
	// matched by the Stage's selector, so it must not appear in the resolved
	// list.
	newTargetStore := func() *fakePromotionStore {
		store := &fakePromotionStore{}
		store.addTarget(testProject.Name, "us-east", map[string]string{"region": "us"})
		store.addTarget(testProject.Name, "eu-west", map[string]string{"region": "eu"})
		return store
	}
	targetStore := newTargetStore()
	repromoteStore := newTargetStore()
	repromoteStore.addRequest(
		testProject.Name, testStage.Name, testStage.Name+".existing.01jexam", testFreight.Name,
		kargoapi.PromotionRequestPhaseSucceeded, "us-east",
	)
	notMirroredStore := &fakePromotionStore{createErr: fmt.Errorf("Freight: %w", database.ErrNotMirrored)}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/promotions",
		[]restTestCase{
			{
				name: "Target-aware Stage yields a PromotionRequest",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTargetAwareStage,
					testFreight,
				),
				serverSetup: serverSetups(authorizeAllStagesPromote, withStore(targetStore)),
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithInfo(ctx, user.Info{IsAdmin: true})
				},
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					require.NoError(t, c.List(t.Context(), promos, client.InNamespace(testProject.Name)))
					require.Empty(t, promos.Items)

					require.Len(t, targetStore.created, 1)
					created := targetStore.created[0]
					require.Equal(t, testProject.Name, created.ProjectName)
					require.Equal(t, testStage.Name, created.Stage)
					require.Equal(t, testFreight.Name, created.Freight)
					// The Stage's selectors are resolved to Targets at creation.
					require.Equal(t, []string{"us-east"}, created.Targets)
					require.Equal(t, api.FormatEventUserActor(user.Info{IsAdmin: true}), created.CreatedBy)

					// The response is the request as stored.
					request := &kargoapi.PromotionRequest{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), request))
					require.Equal(t, created.Name, request.Name)
					require.Equal(t, testStage.Name, request.Spec.Stage)
					require.Equal(t, []kargoapi.PromotionRequestTarget{{Name: "us-east"}}, request.Spec.Targets)
					require.Equal(t, kargoapi.PromotionRequestPhasePending, request.Status.Phase)
					require.Equal(t, created.CreatedBy, request.Annotations[kargoapi.AnnotationKeyCreateActor])
				},
			},
			{
				name: "Target-aware Stage without a database",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTargetAwareStage,
					testFreight,
				),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "Target-aware Stage whose resources have not reached the database yet",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTargetAwareStage,
					testFreight,
				),
				serverSetup: serverSetups(authorizeAllStagesPromote, withStore(notMirroredStore)),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
					require.Contains(t, w.Body.String(), "retry shortly")
				},
			},
			{
				// Taylor's question on #6805: re-promoting the same Freight --
				// rolling back to it, say -- must still produce a request. The
				// stand-down guard exists only in the Stage controller's
				// auto-promotion loop; this path creates unconditionally.
				name: "re-promoting the same Freight yields a second PromotionRequest",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTargetAwareStage,
					testFreight,
				),
				serverSetup: serverSetups(authorizeAllStagesPromote, withStore(repromoteStore)),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					require.Len(t, repromoteStore.created, 1)
					require.Len(t, repromoteStore.requests, 2)
					// Distinct requests: names embed a ULID, so a repeat
					// promotion of the same Stage and Freight cannot collide.
					require.NotEqual(t, repromoteStore.requests[0].Name, repromoteStore.requests[1].Name)
				},
			},
			{
				name: "Promotion by origin is refused for a target-aware Stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTargetAwareStage,
					testFreight,
				),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(promoteToStageRequest{
					Origin: testFreight.Origin.String(),
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					// Nothing resolves an origin for a PromotionRequest, so this
					// must fail loudly rather than fall back to a Promotion
					// that would bypass the Stage's Targets.
					require.Equal(t, http.StatusBadRequest, w.Code)

					promos := &kargoapi.PromotionList{}
					require.NoError(t, c.List(t.Context(), promos, client.InNamespace(testProject.Name)))
					require.Empty(t, promos.Items)
				},
			},
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					Freight: "nonexistent-freight",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found by alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					FreightAlias: "nonexistent-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Neither freight nor freightAlias provided",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body:          mustJSONBody(promoteToStageRequest{}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Both freight and freightAlias provided",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					Freight:      testFreight.Name,
					FreightAlias: "fake-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Both freight and origin provided",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
					Origin:  testFreight.Origin.String(),
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Invalid origin",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteToStageRequest{
					Origin: "Warehouse/",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "promoting not authorized",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("stages").GroupResource(),
							testStage.Name,
							errors.New("not authorized"),
						)
					}
				},
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "Freight not available to Stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *kargoapi.Stage {
						s := testStage.DeepCopy()
						s.Spec.RequestedFreight[0].Sources = kargoapi.FreightSources{
							Stages: []string{"some-other-stage"},
						}
						return s
					}(),
					testFreight,
				),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Successfully promote by freight name",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				serverSetup:   authorizeAllStagesPromote,
				body: mustJSONBody(promoteToStageRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testStage.Name, promos.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, promos.Items[0].Spec.Freight)
					require.NotContains(
						t,
						promos.Items[0].Annotations,
						kargoapi.AnnotationKeyRollback,
					)
				},
			},
			{
				name:          "Successfully promote by freight alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				serverSetup:   authorizeAllStagesPromote,
				body: mustJSONBody(promoteToStageRequest{
					FreightAlias: "fake-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testStage.Name, promos.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, promos.Items[0].Spec.Freight)
				},
			},
			{
				name:          "Successfully promote by origin",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				serverSetup:   authorizeAllStagesPromote,
				body: mustJSONBody(promoteToStageRequest{
					Origin: testFreight.Origin.String(),
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testStage.Name, promos.Items[0].Spec.Stage)
					require.Empty(t, promos.Items[0].Spec.Freight)
					require.Equal(t, testFreight.Origin, *promos.Items[0].Spec.Origin)
				},
			},
			func() restTestCase {
				recorder := fakeevent.NewEventRecorder(1)
				return restTestCase{
					name: "Successfully promote by origin records created event after resolution",
					clientBuilder: fake.NewClientBuilder().WithObjects(
						testProject,
						testStage,
						testFreight,
					),
					serverSetup: func(t *testing.T, s *server) {
						authorizeAllStagesPromote(t, s)
						s.sender = k8sevent.NewEventSender(recorder)
						s.createPromotionFn = func(
							ctx context.Context,
							obj client.Object,
							opts ...client.CreateOption,
						) error {
							promo, ok := obj.(*kargoapi.Promotion)
							require.True(t, ok)
							// Simulate the mutating webhook. REST only supplies origin;
							// admission resolves it before the created object is returned.
							promo.Spec.Freight = testFreight.Name
							promo.Spec.Origin = nil
							return s.client.Create(ctx, obj, opts...)
						}
					},
					body: mustJSONBody(promoteToStageRequest{
						Origin: testFreight.Origin.String(),
					}),
					assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
						require.Equal(t, http.StatusCreated, w.Code)
						require.Len(t, recorder.Events, 1)
						event := <-recorder.Events
						require.Equal(t, corev1.EventTypeNormal, event.EventType)
						require.Equal(t, string(kargoapi.EventTypePromotionCreated), event.Reason)
					},
				}
			}(),
		},
	)
}
