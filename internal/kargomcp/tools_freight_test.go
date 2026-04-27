package kargomcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFreightToSummary(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  freightJSON
		assert func(*testing.T, freightSummary)
	}{
		{
			name: "basic freight with image",
			input: func() freightJSON {
				var f freightJSON
				f.Alias = "worn-panther"
				f.Metadata.Name = "abc123"
				f.Metadata.CreationTimestamp = "2026-01-01T00:00:00Z"
				f.Origin.Name = "my-warehouse"
				f.Images = []struct {
					RepoURL string `json:"repoURL"`
					Tag     string `json:"tag"`
				}{{RepoURL: "nginx", Tag: "1.29.6"}}
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
			name: "freight currently in a stage and verified",
			input: func() freightJSON {
				var f freightJSON
				f.Metadata.Name = "def456"
				f.Status.CurrentlyIn = map[string]json.RawMessage{"prod": nil}
				f.Status.VerifiedIn = map[string]json.RawMessage{"staging": nil}
				return f
			}(),
			assert: func(t *testing.T, s freightSummary) {
				require.Equal(t, []string{"prod"}, s.Stages)
				require.Equal(t, []string{"staging"}, s.VerifiedIn)
			},
		},
		{
			name: "freight with commit",
			input: func() freightJSON {
				var f freightJSON
				f.Metadata.Name = "ghi789"
				f.Commits = []struct {
					RepoURL string `json:"repoURL"`
					ID      string `json:"id"`
					Tag     string `json:"tag"`
					Message string `json:"message"`
				}{{RepoURL: "https://github.com/org/repo", ID: "abc1234", Message: "fix: something"}}
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
