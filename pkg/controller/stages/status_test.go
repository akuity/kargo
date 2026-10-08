package stages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
)

func TestPatchStageStatus(t *testing.T) {
	t.Parallel()

	const (
		testProject = "test-project"
		testStage   = "test-stage"
	)

	testCases := []struct {
		name         string
		serverStatus kargoapi.StageStatus
		newStatus    kargoapi.StageStatus
		assert       func(*testing.T, string, *kargoapi.Stage)
	}{
		{
			name: "server value for an existing key survives",
			serverStatus: kargoapi.StageStatus{
				FreightSummary: "0/1 Fulfilled",
				Metadata: map[string]apiextensionsv1.JSON{
					"lastPromotedAt": {Raw: []byte(`"2024-06-01T00:00:00Z"`)},
				},
			},
			newStatus: kargoapi.StageStatus{
				FreightSummary: "1/1 Fulfilled",
				Metadata: map[string]apiextensionsv1.JSON{
					"lastPromotedAt": {Raw: []byte(`"2024-01-01T00:00:00Z"`)},
				},
			},
			assert: func(t *testing.T, patch string, stage *kargoapi.Stage) {
				assert.NotContains(t, patch, "metadata")
				assert.JSONEq(
					t,
					`"2024-06-01T00:00:00Z"`,
					string(stage.Status.Metadata["lastPromotedAt"].Raw),
				)
				assert.Equal(t, "1/1 Fulfilled", stage.Status.FreightSummary)
			},
		},
		{
			name: "key the desired status lacks survives",
			serverStatus: kargoapi.StageStatus{
				FreightSummary: "0/1 Fulfilled",
				Metadata: map[string]apiextensionsv1.JSON{
					"firstPromotedAt": {Raw: []byte(`"2024-01-01T00:00:00Z"`)},
				},
			},
			newStatus: kargoapi.StageStatus{
				FreightSummary: "1/1 Fulfilled",
			},
			assert: func(t *testing.T, patch string, stage *kargoapi.Stage) {
				assert.NotContains(t, patch, "metadata")
				require.Contains(t, stage.Status.Metadata, "firstPromotedAt")
				assert.JSONEq(
					t,
					`"2024-01-01T00:00:00Z"`,
					string(stage.Status.Metadata["firstPromotedAt"].Raw),
				)
				assert.Equal(t, "1/1 Fulfilled", stage.Status.FreightSummary)
			},
		},
		{
			name: "key the server lacks is not added",
			serverStatus: kargoapi.StageStatus{
				FreightSummary: "0/1 Fulfilled",
			},
			newStatus: kargoapi.StageStatus{
				FreightSummary: "1/1 Fulfilled",
				Metadata: map[string]apiextensionsv1.JSON{
					"onlyInDesired": {Raw: []byte(`"v1"`)},
				},
			},
			assert: func(t *testing.T, patch string, stage *kargoapi.Stage) {
				assert.NotContains(t, patch, "metadata")
				assert.NotContains(t, stage.Status.Metadata, "onlyInDesired")
				assert.Equal(t, "1/1 Fulfilled", stage.Status.FreightSummary)
			},
		},
		{
			name: "changes to other fields are still patched",
			serverStatus: kargoapi.StageStatus{
				FreightSummary: "0/1 Fulfilled",
				Health:         &kargoapi.Health{Status: kargoapi.HealthStateUnknown},
				Conditions: []metav1.Condition{{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionUnknown,
					Reason:             "Testing",
					LastTransitionTime: metav1.Now(),
				}},
				Metadata: map[string]apiextensionsv1.JSON{
					"lastPromotedAt": {Raw: []byte(`"2024-06-01T00:00:00Z"`)},
				},
			},
			newStatus: kargoapi.StageStatus{
				FreightSummary: "1/1 Fulfilled",
				Health:         &kargoapi.Health{Status: kargoapi.HealthStateHealthy},
				Conditions: []metav1.Condition{{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					Reason:             "Testing",
					LastTransitionTime: metav1.Now(),
				}},
			},
			assert: func(t *testing.T, patch string, stage *kargoapi.Stage) {
				assert.NotContains(t, patch, "metadata")
				assert.Contains(t, patch, "freightSummary")
				assert.Contains(t, patch, "health")
				assert.Contains(t, patch, "conditions")
				assert.Equal(t, "1/1 Fulfilled", stage.Status.FreightSummary)
				require.NotNil(t, stage.Status.Health)
				assert.Equal(t, kargoapi.HealthStateHealthy, stage.Status.Health.Status)
				readyCond := conditions.Get(&stage.Status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCond)
				assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
				assert.JSONEq(
					t,
					`"2024-06-01T00:00:00Z"`,
					string(stage.Status.Metadata["lastPromotedAt"].Raw),
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			var patches []string

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testProject,
						Name:      testStage,
					},
					Status: testCase.serverStatus,
				}).
				WithStatusSubresource(&kargoapi.Stage{}).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(
						ctx context.Context,
						c client.Client,
						subResourceName string,
						obj client.Object,
						patch client.Patch,
						opts ...client.SubResourcePatchOption,
					) error {
						data, err := patch.Data(obj)
						if err != nil {
							return err
						}
						patches = append(patches, string(data))
						return c.SubResource(subResourceName).Patch(
							ctx, obj, patch, opts...,
						)
					},
				}).
				Build()

			objKey := client.ObjectKey{Namespace: testProject, Name: testStage}

			stage := &kargoapi.Stage{}
			require.NoError(t, c.Get(t.Context(), objKey, stage))

			newStatus := testCase.newStatus.DeepCopy()
			require.NoError(t, patchStageStatus(t.Context(), c, stage, newStatus))

			require.Len(t, patches, 1)

			stage = &kargoapi.Stage{}
			require.NoError(t, c.Get(t.Context(), objKey, stage))
			testCase.assert(t, patches[0], stage)
		})
	}
}
