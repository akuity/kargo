package kargomcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetStageStripsFreightHistory verifies that get_stage removes
// status.freightHistory (served by get_stage_freight_history instead).
func TestGetStageStripsFreightHistory(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"metadata": map[string]any{"name": "dev"},
		"status": map[string]any{
			"phase": "Steady",
			"freightHistory": []any{
				map[string]any{"id": "abc"},
			},
		},
	}
	data, _ := json.Marshal(payload)
	var v any
	_ = json.Unmarshal(data, &v)

	sanitized, isSanitizedMap := sanitizeResource(v).(map[string]any)
	require.True(t, isSanitizedMap)
	if status, ok := sanitized["status"].(map[string]any); ok {
		delete(status, "freightHistory")
	}

	status, isStatusMap := sanitized["status"].(map[string]any)
	require.True(t, isStatusMap)
	require.Equal(t, "Steady", status["phase"])
	require.NotContains(t, status, "freightHistory")
}

func TestStageToSummary(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  stageJSON
		assert func(*testing.T, stageSummary)
	}{
		{
			name: "minimal stage",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "dev"
				s.Status.Health.Status = "Healthy"
				s.Status.FreightSummary = "abc123"
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.Equal(t, "dev", s.Name)
				require.Equal(t, "Healthy", s.Health)
				require.Equal(t, "abc123", s.CurrentFreight)
				require.False(t, s.AutoPromotionEnabled)
				require.Nil(t, s.LastPromotion)
				require.Empty(t, s.CurrentPromotion)
			},
		},
		{
			name: "stage with auto-promotion enabled",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "dev"
				s.Status.AutoPromotionEnabled = true
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.True(t, s.AutoPromotionEnabled)
			},
		},
		{
			name: "stage with last promotion",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "prod"
				s.Status.Health.Status = "Unknown"
				s.Status.Health.Issues = []string{"reason"}
				s.Status.LastPromotion.Name = "promo-xyz"
				s.Status.LastPromotion.FinishedAt = "2026-01-01T00:00:00Z"
				s.Status.LastPromotion.Status.Phase = "Errored"
				s.Status.LastPromotion.Status.Message = "merge conflict"
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.Equal(t, "prod", s.Name)
				require.Equal(t, []string{"reason"}, s.HealthIssues)
				require.NotNil(t, s.LastPromotion)
				require.Equal(t, "promo-xyz", s.LastPromotion.Name)
				require.Equal(t, "Errored", s.LastPromotion.Phase)
				require.Equal(t, "merge conflict", s.LastPromotion.Message)
			},
		},
		{
			name: "stage with active promotion",
			input: func() stageJSON {
				var s stageJSON
				s.Metadata.Name = "staging"
				s.Status.CurrentPromotion.Name = "promo-active"
				return s
			}(),
			assert: func(t *testing.T, s stageSummary) {
				require.Equal(t, "promo-active", s.CurrentPromotion)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.assert(t, stageToSummary(tc.input))
		})
	}
}

func TestHandleListStages(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/stages": jsonOK(
			`{"items":[{"metadata":{"name":"dev"},"status":{"health":{"status":"Healthy"}}}]}`,
		),
	})
	result, _, err := s.handleListStages(context.Background(), nil, listStagesArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "dev")
}

func TestHandleGetStage(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/stages/dev": jsonOK(
			`{"metadata":{"name":"dev"},"status":{"health":{"status":"Healthy"}}}`,
		),
	})
	result, _, err := s.handleGetStage(context.Background(), nil, getStageArgs{Stage: "dev"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "dev")
}

func TestHandleGetStageFreightHistory(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/stages/dev": jsonOK(`{"metadata":{"name":"dev"},"status":{"freightHistory":[{"id":"hist-1","items":{"wh":{"name":"abc123"}}}]}}`), //nolint:lll
	})
	result, _, err := s.handleGetStageFreightHistory(context.Background(), nil, getStageFreightHistoryArgs{Stage: "dev"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "hist-1")
}

func TestHandleRefreshStage(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/stages/dev/refresh": jsonOK(`{}`),
	})
	result, _, err := s.handleRefreshStage(context.Background(), nil, refreshStageArgs{Stage: "dev"})
	require.NoError(t, err)
	require.False(t, result.IsError)
}
