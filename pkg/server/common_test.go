package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	sigyaml "sigs.k8s.io/yaml"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

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
