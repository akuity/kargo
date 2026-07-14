package server

import (
	"context"
	"errors"
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
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/config"
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

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/promotions",
		[]restTestCase{
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
