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
				URL:   "https://example.com/${{ freight.metadata.name }}",
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
				URL:         "https://example.com/${{ freight.metadata.name }}",
				Description: "View freight details",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(t, "View freight details", resolved[0].Description)
			},
		},
		{
			name: "static URL with no expressions is returned verbatim",
			links: []kargoapi.DeepLink{{
				Title: "Static",
				URL:   "https://example.com/runbook?a=1&b=2",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(t, "https://example.com/runbook?a=1&b=2", resolved[0].URL)
			},
		},
		{
			name: "multiple expressions mixed with text",
			links: []kargoapi.DeepLink{{
				Title: "Mixed",
				URL: "https://example.com/${{ freight.metadata.namespace }}/" +
					"${{ freight.metadata.name }}?tag=${{ freight.images[0].tag }}",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(
					t,
					"https://example.com/my-project/abc123?tag=v1.2.3",
					resolved[0].URL,
				)
			},
		},
		{
			name: "condition true includes link",
			links: []kargoapi.DeepLink{{
				Title: "Release Notes",
				URL:   "https://github.com/releases/${{ freight.metadata.name }}",
				If:    `${{ freight.metadata.namespace == "my-project" }}`,
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
				URL:   "https://github.com/releases/${{ freight.metadata.name }}",
				If:    `${{ freight.metadata.namespace == "other-project" }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Empty(t, resolved)
			},
		},
		{
			name: "condition with surrounding whitespace includes link",
			links: []kargoapi.DeepLink{{
				Title: "Padded",
				URL:   "https://example.com",
				If:    " ${{ freight.metadata.namespace == \"my-project\" }} \n",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
			},
		},
		{
			name: "bare condition without delimiters collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Legacy Condition",
				URL:   "https://example.com",
				If:    `freight.metadata.namespace == "my-project"`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], `"Legacy Condition"`)
				require.Contains(t, errs[0], "${{ }}")
			},
		},
		{
			name: "non-boolean condition collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Non-bool",
				URL:   "https://example.com",
				If:    `${{ 1 + 1 }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "must evaluate to a boolean")
			},
		},
		{
			name: "nil condition collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Nil",
				URL:   "https://example.com",
				If:    `${{ freight.metadata.labels?.env }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "got <nil>")
			},
		},
		{
			name: "invalid condition expression collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Bad Condition",
				URL:   "https://example.com",
				If:    `${{ !!!invalid!!! }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], `"Bad Condition"`)
			},
		},
		{
			name: "legacy Go template URL collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Legacy URL",
				URL:   "https://example.com/{{ .freight.metadata.name }}",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], `"Legacy URL"`)
				require.Contains(t, errs[0], "Go template syntax")
			},
		},
		{
			name: "legacy Go template URL behind a false condition still errors",
			links: []kargoapi.DeepLink{{
				Title: "Half Migrated",
				URL:   "https://example.com/{{ .freight.metadata.name }}",
				If:    `${{ false }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "Go template syntax")
			},
		},
		{
			name: "unclosed expression in URL collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Unclosed",
				URL:   "https://example.com/${{ freight.metadata.name",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "unclosed expression")
			},
		},
		{
			name: "URL expression runtime error collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Out of Range",
				URL:   "https://example.com/${{ freight.images[5].tag }}",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], `"Out of Range"`)
			},
		},
		{
			name: "URL evaluating to a non-string collects error and skips link",
			links: []kargoapi.DeepLink{{
				Title: "Number",
				URL:   "${{ len(freight.images) }}",
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
				require.Contains(t, errs[0], "not a string")
			},
		},
		{
			name: "missing optional field can be defaulted",
			links: []kargoapi.DeepLink{{
				Title: "Alias",
				URL:   `https://example.com/${{ freight.alias ?? "" }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, errs)
				require.Len(t, resolved, 1)
				require.Equal(t, "https://example.com/", resolved[0].URL)
			},
		},
		{
			name: "secret function is unavailable",
			links: []kargoapi.DeepLink{{
				Title: "Secret",
				URL:   `https://example.com/${{ secret("x") }}`,
			}},
			assert: func(t *testing.T, resolved []ResolvedLink, errs []string) {
				require.Empty(t, resolved)
				require.Len(t, errs, 1)
			},
		},
		{
			name: "multiple links: good and bad are handled independently",
			links: []kargoapi.DeepLink{
				{
					Title: "Good",
					URL:   "https://example.com/${{ freight.metadata.name }}",
				},
				{
					Title: "Bad",
					URL:   "https://example.com/${{ freight.metadata.name",
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
