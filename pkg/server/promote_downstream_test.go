package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_promoteDownstream(t *testing.T) {
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
		Status: kargoapi.FreightStatus{
			// Freight must be verified in upstream stage to be available to downstream
			VerifiedIn: map[string]kargoapi.VerifiedStage{
				"fake-stage": {},
			},
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
		},
	}
	testDownstreamStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-downstream-stage",
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
						Stages: []string{testStage.Name},
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

	// Same name as testDownstreamStage, so that it is found downstream of the
	// Stage under test, but governing Targets rather than promoting to itself.
	testTargetAwareDownstreamStage := testDownstreamStage.DeepCopy()
	testTargetAwareDownstreamStage.Spec.Targets = &kargoapi.StageTargets{
		Selectors: []metav1.LabelSelector{{
			MatchLabels: map[string]string{"region": "us"},
		}},
	}
	testDownstreamTarget := &kargoapi.Target{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "us-east",
			Namespace: testProject.Name,
			Labels:    map[string]string{"region": "us"},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name+"/promotions/downstream",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: "nonexistent-freight",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "No downstream stages",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage, testFreight),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "not authorized to promote to a downstream stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testDownstreamStage,
					testFreight,
				),
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
							testDownstreamStage.Name,
							errors.New("not authorized"),
						)
					}
				},
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "successfully promotes downstream",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testDownstreamStage,
					testFreight,
				),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Verify a Promotion was created for the downstream stage
					promos := &kargoapi.PromotionList{}
					err := c.List(t.Context(), promos, client.InNamespace(testProject.Name))
					require.NoError(t, err)
					require.Len(t, promos.Items, 1)
					require.Equal(t, testDownstreamStage.Name, promos.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, promos.Items[0].Spec.Freight)
				},
			},
			{
				name: "target-aware downstream Stage yields a PromotionRequest",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testTargetAwareDownstreamStage,
					testFreight,
					testDownstreamTarget,
				),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					require.Contains(t, w.Body.String(), "promotionRequests")

					promos := &kargoapi.PromotionList{}
					require.NoError(t, c.List(t.Context(), promos, client.InNamespace(testProject.Name)))
					require.Empty(t, promos.Items)

					reqs := &kargoapi.PromotionRequestList{}
					require.NoError(t, c.List(t.Context(), reqs, client.InNamespace(testProject.Name)))
					require.Len(t, reqs.Items, 1)
					require.Equal(t, testTargetAwareDownstreamStage.Name, reqs.Items[0].Spec.Stage)
					require.Equal(t, testFreight.Name, reqs.Items[0].Spec.Freight)
					// The downstream Stage's selectors are resolved to Targets
					// at creation.
					require.Equal(
						t,
						[]kargoapi.PromotionRequestTarget{{Name: "us-east"}},
						reqs.Items[0].Spec.Targets,
					)
					require.Len(t, reqs.Items[0].OwnerReferences, 1)
					require.Equal(
						t,
						testTargetAwareDownstreamStage.Name,
						reqs.Items[0].OwnerReferences[0].Name,
					)
				},
			},
		},
	)
}
