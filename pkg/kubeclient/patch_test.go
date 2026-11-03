package kubeclient

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPatchStatus(t *testing.T) {
	testCases := []struct {
		name string
		// status is the Stage's status before the update.
		status kargoapi.StageStatus
		update func(*kargoapi.StageStatus)
		// expectPatch is the status patch sent, or "" when none is.
		expectPatch string
	}{
		{
			name:   "no change: no patch",
			status: kargoapi.StageStatus{FreightSummary: "ok", LastHandledRefresh: "r1"},
			update: func(*kargoapi.StageStatus) {},
		},
		{
			name:   "same value set again: no patch",
			status: kargoapi.StageStatus{FreightSummary: "ok"},
			update: func(s *kargoapi.StageStatus) { s.FreightSummary = "ok" },
		},
		{
			name:   "nil and empty slice are equal: no patch",
			status: kargoapi.StageStatus{},
			update: func(s *kargoapi.StageStatus) { s.Conditions = []metav1.Condition{} },
		},
		{
			name:        "changed field: patched",
			status:      kargoapi.StageStatus{FreightSummary: "old", LastHandledRefresh: "r1"},
			update:      func(s *kargoapi.StageStatus) { s.FreightSummary = "new" },
			expectPatch: `{"status":{"freightSummary":"new"}}`,
		},
		{
			name:        "cleared field: patched",
			status:      kargoapi.StageStatus{FreightSummary: "old"},
			update:      func(s *kargoapi.StageStatus) { s.FreightSummary = "" },
			expectPatch: `{"status":{"freightSummary":null}}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			stage := &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{Namespace: "fake-project", Name: "fake-stage"},
				Status:     testCase.status,
			}
			var patches []string
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(stage.DeepCopy()).
				WithStatusSubresource(&kargoapi.Stage{}).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(
						ctx context.Context,
						c client.Client,
						sub string,
						obj client.Object,
						patch client.Patch,
						opts ...client.SubResourcePatchOption,
					) error {
						data, err := patch.Data(obj)
						require.NoError(t, err)
						patches = append(patches, string(data))
						return c.SubResource(sub).Patch(ctx, obj, patch, opts...)
					},
				}).
				Build()
			original := stage.Status.DeepCopy()

			require.NoError(t, PatchStatus(t.Context(), c, stage, testCase.update))

			if testCase.expectPatch == "" {
				require.Empty(t, patches)
				// Nothing was sent, so the caller's object is untouched.
				require.Equal(t, original, &stage.Status)
				return
			}
			require.Len(t, patches, 1)
			require.JSONEq(t, testCase.expectPatch, patches[0])
			got := &kargoapi.Stage{}
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(stage), got))
			want := original.DeepCopy()
			testCase.update(want)
			gotJSON, err := json.Marshal(got.Status)
			require.NoError(t, err)
			wantJSON, err := json.Marshal(want)
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), string(gotJSON))
		})
	}
}
