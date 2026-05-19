package deeplinks

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestEvaluateLinks(t *testing.T) {
	t.Parallel()

	ctx := map[string]any{
		"freight": map[string]any{
			"metadata": map[string]any{
				"name":      "abc123",
				"namespace": "my-project",
			},
			"images": []any{
				map[string]any{"tag": "v1.2.3", "repoURL": "ghcr.io/my-org/my-app"},
			},
		},
	}

	testCases := []struct {
		name   string
		links  []kargoapi.DeepLink
		assert func(*testing.T, []ResolvedLink, []string)
	}{
		{
			name: "simple URL with no condition",
			links: []kargoapi.DeepLink{{
				Title: "Freight",
				URL:   "https://example.com/{{ .freight.metadata.name }}",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(t, "Freight", resolved[0].Title)
				require.Equal(t, "https://example.com/abc123", resolved[0].URL)
			},
		},
		{
			name: "description is passed through",
			links: []kargoapi.DeepLink{{
				Title:       "Freight",
				URL:         "https://example.com/{{ .freight.metadata.name }}",
				Description: "View freight details",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(t, "View freight details", resolved[0].Description)
			},
		},
		{
			name: "condition true includes link",
			links: []kargoapi.DeepLink{{
				Title: "Release Notes",
				URL:   "https://github.com/releases/{{ .freight.metadata.name }}",
				If:    `freight.metadata.namespace == "my-project"`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
			},
		},
		{
			name: "condition false omits link",
			links: []kargoapi.DeepLink{{
				Title: "Release Notes",
				URL:   "https://github.com/releases/{{ .freight.metadata.name }}",
				If:    `freight.metadata.namespace == "other-project"`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Empty(t, resolved)
			},
		},
		{
			name: "invalid condition expression collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Bad Condition",
				URL:   "https://example.com",
				If:    `!!!invalid!!!`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], `"Bad Condition"`)
			},
		},
		{
			name: "non-boolean condition collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Non-bool",
				URL:   "https://example.com",
				If:    `1 + 1`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "non-boolean value")
			},
		},
		{
			name: "invalid URL template collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Bad Template",
				URL:   "https://example.com/{{ .freight.metadata.name",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "error parsing URL template")
			},
		},
		{
			name: "URL template execution error collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Exec Error",
				URL:   `https://example.com/{{ index "invalid" .freight.metadata.labels }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "error evaluating URL template")
			},
		},
		{
			name: "sprig functions available in URL template",
			links: []kargoapi.DeepLink{{
				Title: "Sprig",
				URL:   "https://example.com/{{ .freight.metadata.name | upper }}",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(t, "https://example.com/ABC123", resolved[0].URL)
			},
		},
		{
			name: "blocked sprig function env is unavailable",
			links: []kargoapi.DeepLink{{
				Title: "Env",
				URL:   `https://example.com/{{ env "HOME" }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "error parsing URL template")
			},
		},
		{
			name: "multiple links: good and bad are handled independently",
			links: []kargoapi.DeepLink{
				{
					Title: "Good",
					URL:   "https://example.com/{{ .freight.metadata.name }}",
				},
				{
					Title: "Bad",
					URL:   "https://example.com/{{ .freight.metadata.name",
				},
			},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Len(t, resolved, 1)
				require.Equal(t, "Good", resolved[0].Title)
				require.Len(t, errs, 1)
			},
		},
		{
			name:  "empty links returns empty results",
			links: []kargoapi.DeepLink{},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Empty(t, errs)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resolved, errs := EvaluateLinks(tc.links, ctx)
			tc.assert(t, resolved, errs)
		})
	}
}

func TestFreightContext(t *testing.T) {
	t.Parallel()

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "abc123",
			Namespace: "my-project",
		},
	}

	ctx, err := FreightContext(freight)
	require.NoError(t, err)
	require.Contains(t, ctx, "freight")

	freightMap, ok := ctx["freight"].(map[string]any)
	require.True(t, ok)

	meta, ok := freightMap["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "abc123", meta["name"])
}

func TestStageContext(t *testing.T) {
	t.Parallel()

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod",
			Namespace: "my-project",
		},
	}

	ctx, err := StageContext(stage)
	require.NoError(t, err)
	require.Contains(t, ctx, "stage")

	stageMap, ok := ctx["stage"].(map[string]any)
	require.True(t, ok)

	meta, ok := stageMap["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "prod", meta["name"])
}
