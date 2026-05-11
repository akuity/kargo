package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_splitYAML(t *testing.T) {
	// Test resources
	project1 := kargoapi.Project{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
		ObjectMeta: metav1.ObjectMeta{Name: "project-1"},
	}
	project2 := kargoapi.Project{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
		ObjectMeta: metav1.ObjectMeta{Name: "project-2"},
	}
	stage := kargoapi.Stage{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Stage"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-stage", Namespace: "my-project"},
	}
	warehouse := kargoapi.Warehouse{
		TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Warehouse"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-warehouse", Namespace: "my-project"},
	}
	secret := corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: "my-secret", Namespace: "test"},
		Type:       corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name       string
		input      []byte
		assertions func(*testing.T, []unstructured.Unstructured, []unstructured.Unstructured, error)
	}{
		{
			name:  "empty input",
			input: []byte(""),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "whitespace only",
			input: []byte("   \n\t\n   "),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "null document",
			input: []byte("null"),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "single YAML Project",
			input: mustYAML(project1),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
			},
		},
		{
			name:  "single YAML non-Project resource",
			input: mustYAML(stage),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				require.Len(t, other, 1)
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "test-stage", other[0].GetName())
			},
		},
		{
			name:  "multiple YAML documents with ---",
			input: mustYAML(project1, project2),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 2)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
				assert.Equal(t, "project-2", projects[1].GetName())
			},
		},
		{
			name: "mixed YAML documents - Projects and other resources",
			input: mustYAML(
				kargoapi.Project{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
					ObjectMeta: metav1.ObjectMeta{Name: "my-project"},
				},
				stage,
				warehouse,
			),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				require.Len(t, other, 2)
				assert.Equal(t, "my-project", projects[0].GetName())
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "test-stage", other[0].GetName())
				assert.Equal(t, "Warehouse", other[1].GetKind())
				assert.Equal(t, "test-warehouse", other[1].GetName())
			},
		},
		{
			name:  "single JSON object - Project",
			input: mustJSON(project1),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
			},
		},
		{
			name:  "single JSON object - non-Project",
			input: mustJSON(stage),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				require.Len(t, other, 1)
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "test-stage", other[0].GetName())
			},
		},
		{
			name:  "multiple concatenated JSON objects",
			input: mustJSONConcat(project1, project2),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 2)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
				assert.Equal(t, "project-2", projects[1].GetName())
			},
		},
		{
			name:  "JSON array of Projects",
			input: mustJSONArray(project1, project2),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 2)
				assert.Empty(t, other)
				assert.Equal(t, "project-1", projects[0].GetName())
				assert.Equal(t, "project-2", projects[1].GetName())
			},
		},
		{
			name: "JSON array with mixed resource types",
			input: mustJSONArray(
				kargoapi.Project{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Project"},
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-project"},
				},
				kargoapi.Stage{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Stage"},
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-stage", Namespace: "mixed-project"},
				},
				kargoapi.Warehouse{
					TypeMeta:   metav1.TypeMeta{APIVersion: kargoapi.GroupVersion.String(), Kind: "Warehouse"},
					ObjectMeta: metav1.ObjectMeta{Name: "mixed-warehouse", Namespace: "mixed-project"},
				},
			),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				require.Len(t, projects, 1)
				require.Len(t, other, 2)
				assert.Equal(t, "mixed-project", projects[0].GetName())
				assert.Equal(t, "Stage", other[0].GetKind())
				assert.Equal(t, "mixed-stage", other[0].GetName())
				assert.Equal(t, "Warehouse", other[1].GetKind())
				assert.Equal(t, "mixed-warehouse", other[1].GetName())
			},
		},
		{
			name:  "empty JSON array",
			input: []byte("[]"),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				assert.Empty(t, other)
			},
		},
		{
			name:  "invalid YAML",
			input: []byte("invalid: yaml: content:"),
			assertions: func(t *testing.T, _, _ []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "invalid JSON",
			input: []byte(`{"invalid": json}`),
			assertions: func(t *testing.T, _, _ []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "invalid JSON array",
			input: []byte(`[{"valid": "json"}, invalid]`),
			assertions: func(t *testing.T, _, _ []unstructured.Unstructured, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "core Kubernetes resource (Secret)",
			input: mustYAML(secret),
			assertions: func(t *testing.T, projects, other []unstructured.Unstructured, err error) {
				require.NoError(t, err)
				assert.Empty(t, projects)
				require.Len(t, other, 1)
				assert.Equal(t, "Secret", other[0].GetKind())
				assert.Equal(t, "my-secret", other[0].GetName())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, otherResources, err := splitYAML(tt.input)
			tt.assertions(t, projects, otherResources, err)
		})
	}
}

// mustYAML marshals objects to YAML. Multiple objects are separated by "---".
func mustYAML(objs ...any) []byte {
	var docs []string
	for _, obj := range objs {
		b, err := sigyaml.Marshal(obj)
		if err != nil {
			panic(err)
		}
		docs = append(docs, string(b))
	}
	return []byte(strings.Join(docs, "---\n"))
}

// mustJSON marshals a single object to JSON.
func mustJSON(obj any) []byte {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return b
}

// mustJSONConcat marshals multiple objects to newline-separated JSON.
func mustJSONConcat(objs ...any) []byte {
	var lines []string
	for _, obj := range objs {
		b, err := json.Marshal(obj)
		if err != nil {
			panic(err)
		}
		lines = append(lines, string(b))
	}
	return []byte(strings.Join(lines, "\n"))
}

// mustJSONArray marshals multiple objects as a JSON array.
func mustJSONArray(objs ...any) []byte {
	b, err := json.Marshal(objs)
	if err != nil {
		panic(err)
	}
	return b
}

func TestObjectOrRaw(t *testing.T) {
	scheme := runtime.NewScheme()
	err := kargoapi.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	testStageName := "test-stage"
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testStageName,
			Namespace: "test-project",
		},
	}

	testUnstructuredData := map[string]any{
		"apiVersion": kargoapi.GroupVersion.String(),
		"kind":       "Stage",
		"metadata": map[string]any{
			"name":      testStageName,
			"namespace": "test-project",
		},
	}
	testUnstructured := &unstructured.Unstructured{Object: testUnstructuredData}

	testJSON, err := json.Marshal(testUnstructuredData)
	require.NoError(t, err)

	testCases := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "unstructured to JSON",
			test: func(*testing.T) {
				stage, raw, err := objectOrRaw(
					client,
					testUnstructured,
					svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
					&kargoapi.Stage{},
				)
				require.NoError(t, err)
				require.Nil(t, stage)
				require.JSONEq(t, string(testJSON), string(raw))
			},
		},
		{
			name: "unstructured to YAML",
			test: func(*testing.T) {
				stage, raw, err := objectOrRaw(
					client,
					testUnstructured,
					svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
					&kargoapi.Stage{},
				)
				require.NoError(t, err)
				require.Nil(t, stage)
				require.YAMLEq(
					t,
					string(testJSON), // Valid JSON is also valid YAML
					string(raw),
				)
			},
		},
		{
			name: "unstructured to structured",
			test: func(*testing.T) {
				stage, raw, err := objectOrRaw(
					client,
					testUnstructured,
					svcv1alpha1.RawFormat_RAW_FORMAT_UNSPECIFIED,
					&kargoapi.Stage{},
				)
				require.NoError(t, err)
				require.Nil(t, raw)
				require.Equal(t, testStageName, stage.GetName())
			},
		},
		{
			name: "structured to JSON",
			test: func(*testing.T) {
				stage, raw, err := objectOrRaw(
					client,
					testStage,
					svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
					&kargoapi.Stage{},
				)
				require.NoError(t, err)
				require.Nil(t, stage)
				obj := map[string]any{}
				err = json.Unmarshal(raw, &obj)
				require.NoError(t, err)
				// Ensure GVK was not lost
				require.Equal(t, kargoapi.GroupVersion.String(), obj["apiVersion"])
				require.Equal(t, "Stage", obj["kind"])
				// Ensure metadata was not lost
				metadata, ok := obj["metadata"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, testStageName, metadata["name"])
			},
		},
		{
			name: "structured to YAML",
			test: func(*testing.T) {
				stage, raw, err := objectOrRaw(
					client,
					testStage,
					svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
					&kargoapi.Stage{},
				)
				require.NoError(t, err)
				require.Nil(t, stage)
				obj := map[string]any{}
				err = sigyaml.Unmarshal(raw, &obj)
				require.NoError(t, err)

				// Ensure GVK was not lost
				require.Equal(t, kargoapi.GroupVersion.String(), obj["apiVersion"])
				require.Equal(t, "Stage", obj["kind"])
				// Ensure metadata was not lost
				metadata, ok := obj["metadata"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, testStageName, metadata["name"])
			},
		},
		{
			name: "structured to structured",
			test: func(*testing.T) {
				stage, raw, err := objectOrRaw(
					client,
					testStage,
					svcv1alpha1.RawFormat_RAW_FORMAT_UNSPECIFIED,
					&kargoapi.Stage{},
				)
				require.NoError(t, err)
				require.Nil(t, raw)
				require.Same(t, testStage, stage)
			},
		},
		{
			name: "structured to structured with type mismatch",
			test: func(*testing.T) {
				_, _, err := objectOrRaw(
					client,
					testStage,
					svcv1alpha1.RawFormat_RAW_FORMAT_UNSPECIFIED,
					&kargoapi.Project{},
				)
				require.Error(t, err)
				require.Contains(t, err.Error(), "type mismatch")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.test)
	}
}
