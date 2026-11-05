package server

import (
	"context"
	"encoding/json"
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
	// Targets live in the database.
	downstreamTargets := []kargoapi.Target{{ObjectMeta: metav1.ObjectMeta{
		Namespace: testProject.Name,
		Name:      "us-east",
		Labels:    map[string]string{"region": "us"},
	}}}

	// Two more target-aware downstream Stages, named apart from the one above
	// so that a Stage's downstreams can mix Stages that select Targets with
	// one that does not.
	testTargetAwareDownstreamStage2 := testTargetAwareDownstreamStage.DeepCopy()
	testTargetAwareDownstreamStage2.Name = "fake-downstream-stage-2"
	testTargetAwareDownstreamStage3 := testTargetAwareDownstreamStage.DeepCopy()
	testTargetAwareDownstreamStage3.Name = "fake-downstream-stage-3"

	// A server that creates PromotionRollouts promotes to a target-aware
	// downstream Stage through one, in place of a PromotionRequest.
	rolloutsEnabled := &config.ServerConfig{PromotionRolloutsEnabled: true}
	authorizeEverything := func(_ *testing.T, s *server) {
		s.authorizeFn = func(
			context.Context,
			string,
			schema.GroupVersionResource,
			string,
			client.ObjectKey,
		) error {
			return nil
		}
	}
	promoteWithRollouts := func(creator *fakeRolloutCreator) func(*testing.T, *server) {
		return func(t *testing.T, s *server) {
			authorizeEverything(t, s)
			s.rolloutCreator = creator
		}
	}
	// downstreamResponse is the shape of the endpoint's response.
	type downstreamResponse struct {
		Promotions        []kargoapi.Promotion        `json:"promotions"`
		PromotionRequests []kargoapi.PromotionRequest `json:"promotionRequests"`
		PromotionRollouts []kargoapi.PromotionRollout `json:"promotionRollouts"`
		Errors            string                      `json:"errors"`
	}
	decode := func(t *testing.T, w *httptest.ResponseRecorder) downstreamResponse {
		var res downstreamResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		return res
	}
	stagesOf := func(rollouts []kargoapi.PromotionRollout) []string {
		stages := make([]string, len(rollouts))
		for i, rollout := range rollouts {
			stages[i] = rollout.Spec.Stage
		}
		return stages
	}
	happyCreator := &fakeRolloutCreator{}
	fanOutCreator := &fakeRolloutCreator{}
	failingCreator := &fakeRolloutCreator{
		errs: map[string]error{
			testTargetAwareDownstreamStage3.Name: errors.New("something went wrong"),
		},
	}
	disabledCreator := &fakeRolloutCreator{}

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
				name:          "Freight not found by alias",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testStage),
				body: mustJSONBody(promoteDownstreamRequest{
					FreightAlias: "nonexistent-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Freight alias shared by multiple Freight",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testFreight,
					func() *kargoapi.Freight {
						dup := testFreight.DeepCopy()
						dup.Name = "fake-freight-2"
						return dup
					}(),
				),
				body: mustJSONBody(promoteDownstreamRequest{
					FreightAlias: "fake-alias",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
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
				name: "rollouts enabled: target-aware downstream Stage without a database",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testTargetAwareDownstreamStage,
					testFreight,
				),
				serverConfig: rolloutsEnabled,
				serverSetup:  authorizeEverything,
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					// As without the flag, a Stage that cannot be promoted to
					// is reported with the rest rather than failing the call.
					require.Equal(t, http.StatusMultiStatus, w.Code)
					res := decode(t, w)
					require.Contains(t, res.Errors, "database is not configured")
					require.Empty(t, res.PromotionRollouts)
					reqs := &kargoapi.PromotionRequestList{}
					require.NoError(t, c.List(t.Context(), reqs, client.InNamespace(testProject.Name)))
					require.Empty(t, reqs.Items)
				},
			},
			{
				name: "rollouts enabled: target-aware downstream Stage yields a PromotionRollout",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testTargetAwareDownstreamStage,
					testFreight,
				),
				serverConfig: rolloutsEnabled,
				serverSetup:  promoteWithRollouts(happyCreator),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					res := decode(t, w)
					require.Empty(t, res.Promotions)
					require.Empty(t, res.PromotionRequests)
					require.NotContains(t, w.Body.String(), "promotionRequests")
					require.Equal(t, []string{testTargetAwareDownstreamStage.Name}, stagesOf(res.PromotionRollouts))
					require.Equal(t, testFreight.Name, res.PromotionRollouts[0].Spec.Freight)
					promos := &kargoapi.PromotionList{}
					require.NoError(t, c.List(t.Context(), promos, client.InNamespace(testProject.Name)))
					require.Empty(t, promos.Items)
					reqs := &kargoapi.PromotionRequestList{}
					require.NoError(t, c.List(t.Context(), reqs, client.InNamespace(testProject.Name)))
					require.Empty(t, reqs.Items)
					// Promote downstream promotes to every Target the Stage
					// governs, at the Stage's pace.
					require.Equal(t, []rolloutCreation{{
						stage:   testTargetAwareDownstreamStage.Name,
						freight: testFreight.Name,
					}}, happyCreator.creations)
				},
			},
			{
				name: "rollouts enabled: fans out over downstream Stages with and without Targets",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testDownstreamStage,
					testTargetAwareDownstreamStage2,
					testTargetAwareDownstreamStage3,
					testFreight,
				),
				serverConfig: rolloutsEnabled,
				serverSetup:  promoteWithRollouts(fanOutCreator),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					res := decode(t, w)
					require.Empty(t, res.Errors)
					// The Stage without Targets got a Promotion; each of the
					// two with Targets got a rollout.
					require.Len(t, res.Promotions, 1)
					require.Equal(t, testDownstreamStage.Name, res.Promotions[0].Spec.Stage)
					require.ElementsMatch(
						t,
						[]string{testTargetAwareDownstreamStage2.Name, testTargetAwareDownstreamStage3.Name},
						stagesOf(res.PromotionRollouts),
					)
					require.Empty(t, res.PromotionRequests)
					promos := &kargoapi.PromotionList{}
					require.NoError(t, c.List(t.Context(), promos, client.InNamespace(testProject.Name)))
					require.Len(t, promos.Items, 1)
					require.Equal(t, testDownstreamStage.Name, promos.Items[0].Spec.Stage)
					reqs := &kargoapi.PromotionRequestList{}
					require.NoError(t, c.List(t.Context(), reqs, client.InNamespace(testProject.Name)))
					require.Empty(t, reqs.Items)
					require.ElementsMatch(t, []rolloutCreation{
						{stage: testTargetAwareDownstreamStage2.Name, freight: testFreight.Name},
						{stage: testTargetAwareDownstreamStage3.Name, freight: testFreight.Name},
					}, fanOutCreator.creations)
				},
			},
			{
				name: "rollouts enabled: a rollout that cannot be created is reported with the rest",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testDownstreamStage,
					testTargetAwareDownstreamStage2,
					testTargetAwareDownstreamStage3,
					testFreight,
				),
				serverConfig: rolloutsEnabled,
				serverSetup:  promoteWithRollouts(failingCreator),
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusMultiStatus, w.Code)
					res := decode(t, w)
					require.Contains(t, res.Errors, "something went wrong")
					require.Len(t, res.Promotions, 1)
					require.Equal(t, []string{testTargetAwareDownstreamStage2.Name}, stagesOf(res.PromotionRollouts))
				},
			},
			{
				name: "rollouts disabled: target-aware downstream Stage yields a PromotionRequest even with a creator",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testTargetAwareDownstreamStage,
					testFreight,
				),
				serverSetup: func(t *testing.T, s *server) {
					authorizeEverything(t, s)
					s.rolloutCreator = disabledCreator
					s.listTargetsFn = func(context.Context, string) ([]kargoapi.Target, error) {
						return downstreamTargets, nil
					}
				},
				body: mustJSONBody(promoteDownstreamRequest{
					Freight: testFreight.Name,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					res := decode(t, w)
					require.Len(t, res.PromotionRequests, 1)
					require.Empty(t, res.PromotionRollouts)
					require.NotContains(t, w.Body.String(), "promotionRollouts")
					reqs := &kargoapi.PromotionRequestList{}
					require.NoError(t, c.List(t.Context(), reqs, client.InNamespace(testProject.Name)))
					require.Len(t, reqs.Items, 1)
					require.Empty(t, disabledCreator.creations)
				},
			},
			{
				name: "target-aware downstream Stage yields a PromotionRequest",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testStage,
					testTargetAwareDownstreamStage,
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
					s.listTargetsFn = func(context.Context, string) ([]kargoapi.Target, error) {
						return downstreamTargets, nil
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
