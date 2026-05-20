package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getStageAutoPromotionCandidates(t *testing.T) {
	now := time.Now()
	const stageName = "fake-stage"
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

	testRESTEndpoint(
		t,
		&config.ServerConfig{},
		http.MethodGet,
		"/v1beta1/projects/"+project.Name+"/stages/"+stage.Name+"/auto-promotion/candidates",
		[]restTestCase{{
			name:          "returns newest available candidate per origin",
			clientBuilder: fake.NewClientBuilder().WithObjects(project, projectConfig, stage, oldFreight, newFreight),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				var resp autoPromotionCandidatesResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				require.Len(t, resp.Candidates, 1)
				require.True(t, resp.Candidates[0].Origin.Equals(&origin))
				require.Equal(t, newFreight.Name, resp.Candidates[0].Freight.Name)
			},
		}, {
			name: "returns no candidates when auto-promotion is disabled",
			clientBuilder: fake.NewClientBuilder().WithObjects(
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
				oldFreight,
				newFreight,
			),
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
		require.Equal(t, "promote", verb)
		require.Equal(t, kargoapi.GroupVersion.WithResource("stages"), gvr)
		return nil
	}
}
