package get

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_sanitizeForExport(t *testing.T) {
	testCases := []struct {
		name   string
		obj    func() *kargoapi.Stage
		assert func(*testing.T, map[string]any, error)
	}{
		{
			name: "strips status and non-applyable metadata fields",
			obj: func() *kargoapi.Stage {
				return &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "my-stage",
						Namespace:         "my-project",
						Labels:            map[string]string{"foo": "bar"},
						Annotations:       map[string]string{"baz": "qux"},
						CreationTimestamp: metav1.Now(),
						ResourceVersion:   "123",
						UID:               "abc-123",
						Generation:        5,
					},
					Spec: kargoapi.StageSpec{
						Shard: "my-shard",
					},
					Status: kargoapi.StageStatus{
						FreightSummary: "some freight",
					},
				}
			},
			assert: func(t *testing.T, m map[string]any, err error) {
				require.NoError(t, err)

				require.Equal(t, "kargo.akuity.io/v1alpha1", m["apiVersion"])
				require.Equal(t, "Stage", m["kind"])
				require.NotContains(t, m, "status")

				metadata, ok := m["metadata"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "my-stage", metadata["name"])
				require.Equal(t, "my-project", metadata["namespace"])
				require.Equal(t, map[string]any{"foo": "bar"}, metadata["labels"])
				require.Equal(t, map[string]any{"baz": "qux"}, metadata["annotations"])
				for _, field := range exportMetadataFields {
					require.NotContains(t, metadata, field)
				}

				spec, ok := m["spec"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "my-shard", spec["shard"])
			},
		},
		{
			name: "object with no status is unaffected",
			obj: func() *kargoapi.Stage {
				return &kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{Name: "my-stage", Namespace: "my-project"},
				}
			},
			assert: func(t *testing.T, m map[string]any, err error) {
				require.NoError(t, err)
				require.NotContains(t, m, "status")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitized, err := sanitizeForExport(tc.obj())
			var m map[string]any
			if sanitized != nil {
				m = sanitized.Object
			}
			tc.assert(t, m, err)
		})
	}
}

func Test_PrintExportableObjects(t *testing.T) {
	newStage := func(name string) *kargoapi.Stage {
		return &kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         "my-project",
				CreationTimestamp: metav1.Now(),
				ResourceVersion:   "123",
			},
		}
	}

	testCases := []struct {
		name    string
		objects []*kargoapi.Stage
		export  bool
		useFile bool
		assert  func(*testing.T, string, string, error)
	}{
		{
			name:    "export false falls back to table output",
			objects: []*kargoapi.Stage{newStage("stage-1")},
			export:  false,
			assert: func(t *testing.T, out string, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, out, "stage-1")
				require.NotContains(t, out, "apiVersion")
			},
		},
		{
			name:    "export true strips cruft and prints yaml",
			objects: []*kargoapi.Stage{newStage("stage-1")},
			export:  true,
			assert: func(t *testing.T, out string, _ string, err error) {
				require.NoError(t, err)
				require.Contains(t, out, "kind: Stage")
				require.NotContains(t, out, "resourceVersion")
				require.NotContains(t, out, "creationTimestamp")
			},
		},
		{
			name:    "export true separates multiple objects with document markers",
			objects: []*kargoapi.Stage{newStage("stage-1"), newStage("stage-2")},
			export:  true,
			assert: func(t *testing.T, out string, _ string, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, strings.Count(out, "---\n"))
				require.Contains(t, out, "name: stage-1")
				require.Contains(t, out, "name: stage-2")
			},
		},
		{
			name:    "export true with output file writes to disk instead of stdout",
			objects: []*kargoapi.Stage{newStage("stage-1")},
			export:  true,
			useFile: true,
			assert: func(t *testing.T, out string, fileContents string, err error) {
				require.NoError(t, err)
				require.Empty(t, out)
				require.Contains(t, fileContents, "kind: Stage")
				require.Contains(t, fileContents, "name: stage-1")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var outBuf bytes.Buffer
			streams := genericiooptions.IOStreams{Out: &outBuf, ErrOut: &outBuf}

			outputFile := ""
			if tc.useFile {
				outputFile = filepath.Join(t.TempDir(), "out.yaml")
			}

			err := PrintExportableObjects(
				tc.objects,
				genericclioptions.NewPrintFlags(""),
				streams,
				false,
				tc.export,
				outputFile,
			)

			var fileContents string
			if tc.useFile {
				data, readErr := os.ReadFile(outputFile)
				require.NoError(t, readErr)
				fileContents = string(data)
			}

			tc.assert(t, outBuf.String(), fileContents, err)
		})
	}
}

func Test_PrintExportableObjects_configMap(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "my-configmap",
			Namespace:         "my-project",
			CreationTimestamp: metav1.Now(),
			ResourceVersion:   "123",
		},
		Data: map[string]string{"key": "value"},
	}

	var outBuf bytes.Buffer
	streams := genericiooptions.IOStreams{Out: &outBuf, ErrOut: &outBuf}

	err := PrintExportableObjects(
		[]*corev1.ConfigMap{cm},
		genericclioptions.NewPrintFlags(""),
		streams,
		false,
		true,
		"",
	)
	require.NoError(t, err)
	require.Contains(t, outBuf.String(), "kind: ConfigMap")
	require.NotContains(t, outBuf.String(), "resourceVersion")
}
