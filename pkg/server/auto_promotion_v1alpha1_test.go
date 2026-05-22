package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getStageAutoPromotionCandidates(t *testing.T) {
	now := time.Now()
	const stageName = "fake-stage"
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "fake-project"}}
	projectConfig := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project.Name,
			Namespace: project.Name,
		},
		Spec: kargoapi.ProjectConfigSpec{
			PromotionPolicies: []kargoapi.PromotionPolicy{{
				StageSelector:        &kargoapi.PromotionPolicySelector{Name: stageName},
				AutoPromotionEnabled: true,
			}},
		},
	}
	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	warehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Name:      origin.Name,
			Namespace: project.Name,
		},
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageName,
			Namespace: project.Name,
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: origin,
				Sources: kargoapi.FreightSources{
					Direct: true,
				},
			}},
		},
		Status: kargoapi.StageStatus{
			AutoPromotionEnabled: true,
		},
	}
	oldFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "old-freight",
			Namespace:         project.Name,
			CreationTimestamp: metav1.Time{Time: now.Add(-time.Hour)},
		},
		Origin: origin,
	}
	newFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "new-freight",
			Namespace:         project.Name,
			CreationTimestamp: metav1.Time{Time: now},
		},
		Origin: origin,
	}
	matchUpstreamStage := stage.DeepCopy()
	matchUpstreamStage.Name = "match-upstream"
	matchUpstreamStage.Spec.RequestedFreight[0].Sources = kargoapi.FreightSources{
		Stages: []string{"upstream"},
		AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
			SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
		},
	}
	matchUpstreamProjectConfig := projectConfig.DeepCopy()
	matchUpstreamProjectConfig.Spec.PromotionPolicies[0].StageSelector.Name = matchUpstreamStage.Name
	currentUpstreamFreight := oldFreight.DeepCopy()
	currentUpstreamFreight.Status = kargoapi.FreightStatus{
		CurrentlyIn: map[string]kargoapi.CurrentStage{"upstream": {}},
		VerifiedIn:  map[string]kargoapi.VerifiedStage{"upstream": {}},
	}
	otherVerifiedFreight := newFreight.DeepCopy()
	otherVerifiedFreight.Status = kargoapi.FreightStatus{
		VerifiedIn: map[string]kargoapi.VerifiedStage{"upstream": {}},
	}

	testRESTEndpoint(
		t,
		&config.ServerConfig{},
		http.MethodGet,
		"/v1beta1/projects/"+project.Name+"/stages/"+stage.Name+"/auto-promotion/candidates",
		[]restTestCase{{
			name: "returns newest available candidate per origin",
			clientBuilder: fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(project, projectConfig, warehouse, stage, oldFreight, newFreight).
				WithIndex(&kargoapi.Freight{}, indexer.FreightByWarehouseField, indexer.FreightByWarehouse),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				var resp autoPromotionCandidatesResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.Len(t, resp.Candidates, 1)
				require.True(t, resp.Candidates[0].Origin.Equals(&origin))
				require.Equal(t, newFreight.Name, resp.Candidates[0].Freight.Name)
			},
		}, {
			name: "honors MatchUpstream currently-in filtering",
			url: "/v1beta1/projects/" + project.Name + "/stages/" +
				matchUpstreamStage.Name + "/auto-promotion/candidates",
			clientBuilder: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					project,
					matchUpstreamProjectConfig,
					warehouse,
					matchUpstreamStage,
					currentUpstreamFreight,
					otherVerifiedFreight,
				).
				WithIndex(&kargoapi.Freight{}, indexer.FreightByWarehouseField, indexer.FreightByWarehouse).
				WithIndex(&kargoapi.Freight{}, indexer.FreightApprovedForStagesField, indexer.FreightApprovedForStages).
				WithIndex(&kargoapi.Freight{}, indexer.FreightByCurrentStagesField, indexer.FreightByCurrentStages).
				WithIndex(&kargoapi.Freight{}, indexer.FreightByVerifiedStagesField, indexer.FreightByVerifiedStages),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				var resp autoPromotionCandidatesResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.Len(t, resp.Candidates, 1)
				require.True(t, resp.Candidates[0].Origin.Equals(&origin))
				require.Equal(t, currentUpstreamFreight.Name, resp.Candidates[0].Freight.Name)
			},
		}, {
			name: "returns no candidates when auto-promotion is disabled",
			clientBuilder: fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(
					project,
					func() *kargoapi.ProjectConfig {
						cfg := projectConfig.DeepCopy()
						cfg.Spec.PromotionPolicies[0].AutoPromotionEnabled = false
						return cfg
					}(),
					func() *kargoapi.Stage {
						disabledStage := stage.DeepCopy()
						return disabledStage
					}(),
					warehouse,
					oldFreight,
					newFreight,
				).
				WithIndex(&kargoapi.Freight{}, indexer.FreightByWarehouseField, indexer.FreightByWarehouse),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				var resp autoPromotionCandidatesResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.Empty(t, resp.Candidates)
			},
		}},
	)
}

func Test_server_resumeStageAutoPromotion(t *testing.T) {
	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "fake-project"}}
	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	otherOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "other-warehouse",
	}
	stageWithHold := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: project.Name,
		},
		Status: kargoapi.StageStatus{
			AutoPromotionHolds: map[string]kargoapi.AutoPromotionHold{
				origin.String(): {
					Freight: kargoapi.FreightReference{
						Name:   "old-freight",
						Origin: origin,
					},
					State: kargoapi.AutoPromotionHoldStateActive,
				},
			},
		},
	}
	stageWithPendingHold := stageWithHold.DeepCopy()
	pendingHold := stageWithPendingHold.Status.AutoPromotionHolds[origin.String()]
	pendingHold.State = kargoapi.AutoPromotionHoldStatePending
	pendingHold.PromotionName = "pending-promotion"
	stageWithPendingHold.Status.AutoPromotionHolds[origin.String()] = pendingHold
	stageWithTwoHolds := stageWithHold.DeepCopy()
	stageWithTwoHolds.Status.AutoPromotionHolds[otherOrigin.String()] =
		kargoapi.AutoPromotionHold{
			Freight: kargoapi.FreightReference{
				Name:   "other-freight",
				Origin: otherOrigin,
			},
			State: kargoapi.AutoPromotionHoldStateActive,
		}

	testRESTEndpoint(
		t,
		&config.ServerConfig{},
		http.MethodPost,
		"/v1beta1/projects/"+project.Name+"/stages/"+stageWithHold.Name+"/auto-promotion/resume",
		[]restTestCase{
			{
				name: "clears active holds",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(project, stageWithHold).
					WithStatusSubresource(stageWithHold),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(resumeStageAutoPromotionRequest{
					Origin: &origin,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)
					stage := &kargoapi.Stage{}
					require.NoError(t, c.Get(
						t.Context(),
						client.ObjectKey{Namespace: project.Name, Name: stageWithHold.Name},
						stage,
					))
					require.Empty(t, stage.Status.AutoPromotionHolds)
				},
			},
			{
				name: "clears only matching hold",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(project, stageWithTwoHolds).
					WithStatusSubresource(stageWithTwoHolds),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(resumeStageAutoPromotionRequest{
					Origin: &origin,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)
					stage := &kargoapi.Stage{}
					require.NoError(t, c.Get(
						t.Context(),
						client.ObjectKey{Namespace: project.Name, Name: stageWithTwoHolds.Name},
						stage,
					))
					require.Len(t, stage.Status.AutoPromotionHolds, 1)
					require.Contains(t, stage.Status.AutoPromotionHolds, otherOrigin.String())
				},
			},
			{
				name: "origin is required",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(project, stageWithHold).
					WithStatusSubresource(stageWithHold),
				serverSetup: authorizeAllStagesPromote,
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "origin must be canonical",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(project, stageWithHold).
					WithStatusSubresource(stageWithHold),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(resumeStageAutoPromotionRequest{
					Origin: &kargoapi.FreightOrigin{
						Kind: "NotAWarehouse",
						Name: "fake-warehouse",
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "pending hold blocks resume",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(project, stageWithPendingHold).
					WithStatusSubresource(stageWithPendingHold),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(resumeStageAutoPromotionRequest{
					Origin: &origin,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
					require.Contains(t, w.Body.String(), "pending-promotion")
				},
			},
			{
				name:          "not authorized is checked before Stage lookup",
				clientBuilder: fake.NewClientBuilder().WithObjects(project),
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							schema.GroupResource{
								Group:    kargoapi.GroupVersion.Group,
								Resource: "stages",
							},
							stageWithHold.Name,
							errors.New("not authorized"),
						)
					}
				},
				body: mustJSONBody(resumeStageAutoPromotionRequest{
					Origin: &origin,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "refresh failure is logged after clearing hold",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(project, stageWithHold).
					WithStatusSubresource(stageWithHold).
					WithInterceptorFuncs(interceptor.Funcs{
						Patch: func() func(
							context.Context,
							client.WithWatch,
							client.Object,
							client.Patch,
							...client.PatchOption,
						) error {
							var stagePatchCount int
							return func(
								ctx context.Context,
								c client.WithWatch,
								obj client.Object,
								patch client.Patch,
								opts ...client.PatchOption,
							) error {
								if _, ok := obj.(*kargoapi.Stage); ok {
									stagePatchCount++
									if stagePatchCount == 2 {
										return errors.New("refresh failed")
									}
								}
								return c.Patch(ctx, obj, patch, opts...)
							}
						}(),
					}),
				serverSetup: authorizeAllStagesPromote,
				body: mustJSONBody(resumeStageAutoPromotionRequest{
					Origin: &origin,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)
					stage := &kargoapi.Stage{}
					require.NoError(t, c.Get(
						t.Context(),
						client.ObjectKey{Namespace: project.Name, Name: stageWithHold.Name},
						stage,
					))
					require.Empty(t, stage.Status.AutoPromotionHolds)
				},
			},
		},
	)
}

func authorizeAllStagesPromote(t *testing.T, s *server) {
	s.authorizeFn = func(
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
