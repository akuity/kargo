package projectconfigs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestResponseBody(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		cfg    *kargoapi.ProjectConfig
		assert func(*testing.T, *Resource)
	}{
		{
			name: "carries apiVersion and kind beside the ProjectConfig",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProject,
					Namespace: testProject,
				},
				Spec: kargoapi.ProjectConfigSpec{
					FreightLinks: []kargoapi.DeepLink{{Title: "t", URL: "https://example.com"}},
				},
			},
			assert: func(t *testing.T, r *Resource) {
				data, err := json.Marshal(r)
				require.NoError(t, err)
				require.Contains(t, string(data), `"apiVersion":"kargo.akuity.io/v1alpha1"`)
				require.Contains(t, string(data), `"kind":"ProjectConfig"`)
				raw := map[string]any{}
				require.NoError(t, json.Unmarshal(data, &raw))
				require.Equal(t, testProject, raw["metadata"].(map[string]any)["name"]) // nolint: forcetypeassert
				require.Contains(t, raw, "spec")
				// The ProjectConfig decodes from it as from any other
				// manifest.
				cfg := &kargoapi.ProjectConfig{}
				require.NoError(t, json.Unmarshal(data, cfg))
				require.Equal(t, testProject, cfg.Name)
				require.Len(t, cfg.Spec.FreightLinks, 1)
			},
		},
		{
			name: "nil is served as an empty ProjectConfig",
			assert: func(t *testing.T, r *Resource) {
				require.Equal(t, kargoapi.GroupVersion.String(), r.APIVersion)
				require.Equal(t, "ProjectConfig", r.Kind)
				require.Empty(t, r.Name)
			},
		},
		{
			name: "is a runtime.Object of the ProjectConfig kind",
			cfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"a": "b"}},
			},
			assert: func(t *testing.T, r *Resource) {
				var obj runtime.Object = r
				require.Equal(
					t,
					kargoapi.GroupVersion.WithKind("ProjectConfig"),
					obj.GetObjectKind().GroupVersionKind(),
				)
				clone, ok := obj.DeepCopyObject().(*Resource)
				require.True(t, ok)
				require.Equal(t, r, clone)
				clone.Labels["a"] = "c"
				require.Equal(t, "b", r.Labels["a"])
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, ResponseBody(testCase.cfg))
		})
	}
}
