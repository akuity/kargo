package kargomcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

func TestPromotionToSummary(t *testing.T) {
	t.Parallel()
	stage := "prod"
	freight := "abc123"
	p := &models.Promotion{}
	p.Metadata = &models.V1ObjectMeta{Name: "promo-1"}
	p.Spec.Stage = &stage
	p.Spec.Freight = &freight
	p.Status.Phase = "Succeeded"
	p.Status.Message = "done"
	p.Status.StartedAt = "2026-01-01T00:00:00Z"
	p.Status.FinishedAt = "2026-01-01T00:01:00Z"

	s := promotionToSummary(p)
	require.Equal(t, "promo-1", s.Name)
	require.Equal(t, "prod", s.Stage)
	require.Equal(t, "abc123", s.Freight)
	require.Equal(t, "Succeeded", s.Phase)
	require.Equal(t, "done", s.Message)
	require.Equal(t, "2026-01-01T00:00:00Z", s.StartedAt)
	require.Equal(t, "2026-01-01T00:01:00Z", s.FinishedAt)
}

func TestHandleListPromotions(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/promotions": jsonOK(
			`{"items":[{"metadata":{"name":"promo-1"},"spec":{"stage":"dev","freight":"abc"},"status":{"phase":"Succeeded"}}]}`, //nolint:lll
		),
	})
	result, _, err := s.handleListPromotions(context.Background(), nil, listPromotionsArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "promo-1")
}

func TestHandleGetPromotion(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/promotions/promo-1": jsonOK(`{"metadata":{"name":"promo-1"}}`),
	})
	result, _, err := s.handleGetPromotion(context.Background(), nil, getPromotionArgs{Promotion: "promo-1"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "promo-1")
}

func TestHandlePromoteToStage(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/stages/dev/promotions": jsonCreated(`{"metadata":{"name":"promo-new"}}`),
	})
	result, _, err := s.handlePromoteToStage(
		context.Background(), nil,
		promoteToStageArgs{Stage: "dev", Freight: "abc"},
	)
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestHandlePromoteDownstream(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/stages/dev/promotions/downstream": jsonCreated(`{}`),
	})
	result, _, err := s.handlePromoteDownstream(
		context.Background(), nil,
		promoteDownstreamArgs{Stage: "dev", Freight: "abc"},
	)
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestHandleAbortPromotion(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/promotions/promo-1/abort": jsonOK(`{}`),
	})
	result, _, err := s.handleAbortPromotion(context.Background(), nil, abortPromotionArgs{Promotion: "promo-1"})
	require.NoError(t, err)
	require.False(t, result.IsError)
}
