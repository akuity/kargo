package get

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
)

func TestGetStagesOptions_Run_NoPhantomObjects(t *testing.T) {
	served := &kargoapi.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "kargo-demo"},
	}
	servedJSON, err := json.Marshal(served)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(servedJSON)
		},
	))
	defer srv.Close()

	streams, _, out, _ := genericiooptions.NewTestIOStreams()
	printFlags := genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme())
	jsonFormat := "json"
	printFlags.OutputFormat = &jsonFormat
	// AddFlags (never called here, since there's no cobra.Command in this
	// test) is what normally wires OutputFlagSpecified to detect that the
	// user passed -o/--output. Set it directly so PrintObjects takes the
	// structured-printer path instead of falling back to the table printer,
	// which is what actually exercises the JSON round-trip this test cares
	// about.
	printFlags.OutputFlagSpecified = func() bool { return true }

	o := &getStagesOptions{
		IOStreams:     streams,
		PrintFlags:    printFlags,
		getOptions:    &getOptions{},
		Config:        config.CLIConfig{APIAddress: srv.URL, BearerToken: "test-token"},
		ClientOptions: client.Options{},
		Project:       "kargo-demo",
		Names:         []string{"test"},
	}

	require.NoError(t, o.run(t.Context()))
	// Sanity-check that the JSON printer path actually ran and produced the
	// object we served, so the NotContains assertions below are proving
	// something rather than passing vacuously against empty output.
	require.Contains(t, out.String(), `"name": "test"`)
	require.NotContains(t, out.String(), `"promotionTemplate": {}`)
	require.NotContains(t, out.String(), `"health": {}`)
}
