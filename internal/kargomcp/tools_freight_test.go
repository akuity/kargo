package kargomcp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

func TestFreightToSummary(t *testing.T) {
	t.Parallel()
	namePtr := func(s string) *string { return &s }
	testCases := []struct {
		name   string
		input  *models.Freight
		assert func(*testing.T, freightSummary)
	}{
		{
			name: "basic freight with image",
			input: func() *models.Freight {
				f := &models.Freight{}
				f.Alias = "worn-panther"
				f.Metadata = &models.V1ObjectMeta{
					Name:              "abc123",
					CreationTimestamp: "2026-01-01T00:00:00Z",
				}
				f.Origin.Name = namePtr("my-warehouse")
				f.Images = []*models.Image{{RepoURL: "nginx", Tag: "1.29.6"}}
				return f
			}(),
			assert: func(t *testing.T, s freightSummary) {
				require.Equal(t, "abc123", s.Name)
				require.Equal(t, "worn-panther", s.Alias)
				require.Equal(t, "my-warehouse", s.Warehouse)
				require.Equal(t, "2026-01-01T00:00:00Z", s.CreatedAt)
				require.Len(t, s.Images, 1)
				require.Equal(t, "nginx", s.Images[0].RepoURL)
				require.Equal(t, "1.29.6", s.Images[0].Tag)
				require.Empty(t, s.Stages)
				require.Empty(t, s.VerifiedIn)
			},
		},
		{
			name: "freight currently in a stage, verified, and approved",
			input: func() *models.Freight {
				f := &models.Freight{}
				f.Metadata = &models.V1ObjectMeta{Name: "def456"}
				f.Status.CurrentlyIn = map[string]models.CurrentStage{"prod": {}}
				f.Status.VerifiedIn = map[string]models.VerifiedStage{"staging": {}}
				f.Status.ApprovedFor = map[string]models.ApprovedStage{"hotfix": {}}
				return f
			}(),
			assert: func(t *testing.T, s freightSummary) {
				require.Equal(t, []string{"prod"}, s.Stages)
				require.Equal(t, []string{"staging"}, s.VerifiedIn)
				require.Equal(t, []string{"hotfix"}, s.ApprovedFor)
			},
		},
		{
			name: "freight with commit",
			input: func() *models.Freight {
				f := &models.Freight{}
				f.Metadata = &models.V1ObjectMeta{Name: "ghi789"}
				f.Commits = []*models.GitCommit{{
					RepoURL: "https://github.com/org/repo",
					ID:      "abc1234",
					Message: "fix: something",
				}}
				return f
			}(),
			assert: func(t *testing.T, s freightSummary) {
				require.Len(t, s.Commits, 1)
				require.Equal(t, "abc1234", s.Commits[0].ID)
				require.Equal(t, "fix: something", s.Commits[0].Message)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.assert(t, freightToSummary(tc.input))
		})
	}
}

func TestFlattenFreightGroups(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  any
		assert func(*testing.T, []json.RawMessage)
	}{
		{
			name:   "nil payload",
			input:  nil,
			assert: func(t *testing.T, got []json.RawMessage) { require.Empty(t, got) },
		},
		{
			name: "single default group",
			input: map[string]any{
				"groups": map[string]any{
					"": map[string]any{
						"items": []any{
							map[string]any{"name": "a"},
							map[string]any{"name": "b"},
						},
					},
				},
			},
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 2)
			},
		},
		{
			name: "multiple groups are merged",
			input: map[string]any{
				"groups": map[string]any{
					"g1": map[string]any{"items": []any{map[string]any{"name": "a"}}},
					"g2": map[string]any{"items": []any{map[string]any{"name": "b"}}},
				},
			},
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 2)
			},
		},
		{
			name: "empty groups yields empty slice",
			input: map[string]any{
				"groups": map[string]any{},
			},
			assert: func(t *testing.T, got []json.RawMessage) { require.Empty(t, got) },
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := flattenFreightGroups(tc.input)
			tc.assert(t, got)
		})
	}
}

func TestHandleListFreight(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/freight": jsonOK(
			`{"groups":{"":{"items":[{"metadata":{"name":"abc"},"alias":"funky-moose","origin":{"name":"my-wh"}}]}}}`,
		),
	})
	result, _, err := s.handleListFreight(context.Background(), nil, listFreightArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "funky-moose")
}

func TestHandleGetFreight(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/freight/abc": jsonOK(`{"metadata":{"name":"abc"},"alias":"funky-moose"}`),
	})
	result, _, err := s.handleGetFreight(context.Background(), nil, getFreightArgs{FreightNameOrAlias: "abc"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "funky-moose")
}

func TestHandleApproveFreight(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/freight/abc/approve": jsonOK(`{}`),
	})
	result, _, err := s.handleApproveFreight(
		context.Background(), nil,
		approveFreightArgs{FreightNameOrAlias: "abc", Stage: "dev"},
	)
	require.NoError(t, err)
	require.False(t, result.IsError)
}
