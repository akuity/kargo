package verification

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestAnalysisRunnerRollouts_ClearAnalysisRunsForStage(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, rolloutsapi.AddToScheme(scheme))

	tests := []struct {
		name        string
		stage       *kargoapi.Stage
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, client.Client, error)
	}{
		{
			name: "deletes only AnalysisRuns for the Stage",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stage-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
				},
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-stage-analysis",
						Namespace: "fake-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "other-stage",
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				list := &rolloutsapi.AnalysisRunList{}
				require.NoError(t, c.List(context.Background(), list))
				require.Len(t, list.Items, 1)
				assert.Equal(t, "other-stage-analysis", list.Items[0].Name)
			},
		},
		{
			name: "does not delete AnalysisRuns in other namespaces",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			objects: []client.Object{
				&rolloutsapi.AnalysisRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-namespace-analysis",
						Namespace: "other-project",
						Labels: map[string]string{
							kargoapi.LabelKeyStage: "test-stage",
						},
					},
				},
			},
			assertions: func(t *testing.T, c client.Client, err error) {
				require.NoError(t, err)

				list := &rolloutsapi.AnalysisRunList{}
				require.NoError(t, c.List(context.Background(), list))
				require.Len(t, list.Items, 1)
			},
		},
		{
			name: "handles delete error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-project",
					Name:      "test-stage",
				},
			},
			interceptor: interceptor.Funcs{
				DeleteAllOf: func(
					context.Context,
					client.WithWatch,
					client.Object,
					...client.DeleteAllOfOption,
				) error {
					return fmt.Errorf("delete error")
				},
			},
			assertions: func(t *testing.T, _ client.Client, err error) {
				require.ErrorContains(t, err, "error deleting AnalysisRuns for Stage")
				require.ErrorContains(t, err, "delete error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...)

			if tt.interceptor.DeleteAllOf != nil {
				builder = builder.WithInterceptorFuncs(tt.interceptor)
			}

			c := builder.Build()

			runner := &analysisRunnerRollouts{
				client:                     c,
				rolloutsIntegrationEnabled: true,
			}

			err := runner.ClearAnalysisRunsForStage(t.Context(), tt.stage)
			tt.assertions(t, c, err)
		})
	}
}
