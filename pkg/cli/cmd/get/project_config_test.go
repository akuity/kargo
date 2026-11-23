package get

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	kargogen "github.com/akuity/kargo/pkg/x/client/generated"
)

func Test_getProjectConfigOptions_print(t *testing.T) {
	// The model as the generated client decodes what the API server serves.
	model := &kargogen.ProjectConfig{}
	require.NoError(t, json.Unmarshal([]byte(`{
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"kind": "ProjectConfig",
		"metadata": {"name": "my-project", "namespace": "my-project"},
		"spec": {"freightLinks": [{"title": "t", "url": "https://example.com"}]}
	}`), model))

	testCases := []struct {
		name   string
		args   []string
		assert func(*testing.T, string, error)
	}{
		{
			name: "yaml carries apiVersion and kind",
			args: []string{"--output=yaml"},
			assert: func(t *testing.T, out string, err error) {
				require.NoError(t, err)
				obj := map[string]any{}
				require.NoError(t, yaml.Unmarshal([]byte(out), &obj))
				require.Equal(t, kargoapi.GroupVersion.String(), obj["apiVersion"])
				require.Equal(t, "ProjectConfig", obj["kind"])
				require.Contains(t, out, "https://example.com")
			},
		},
		{
			name: "table",
			assert: func(t *testing.T, out string, err error) {
				require.NoError(t, err)
				require.Contains(t, out, "NAME")
				require.Contains(t, out, "my-project")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			opts := &getProjectConfigOptions{
				IOStreams:  genericiooptions.IOStreams{Out: out, ErrOut: out},
				getOptions: &getOptions{},
				PrintFlags: genericclioptions.NewPrintFlags("").
					WithTypeSetter(kubernetes.GetScheme()),
			}
			cmd := &cobra.Command{}
			opts.AddFlags(cmd)
			require.NoError(t, cmd.ParseFlags(testCase.args))
			err := opts.print(model)
			testCase.assert(t, out.String(), err)
		})
	}
}
