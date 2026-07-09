package kubeclient

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPatchStatus(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	testCases := []struct {
		name        string
		stage       *kargoapi.Stage
		update      func(*kargoapi.StageStatus)
		interceptor interceptor.Funcs
		assert      func(*testing.T, error)
	}{
		{
			name: "no patch issued for no change",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "default",
					Name:            "test-stage",
					ResourceVersion: "1",
				},
			},
			update: func(*kargoapi.StageStatus) {},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					_ context.Context,
					_ client.Client,
					_ string,
					_ client.Object,
					_ client.Patch,
					_ ...client.SubResourcePatchOption,
				) error {
					t.Fatal("unexpected patch call for no-op update")
					return nil
				},
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "patch includes resource version",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "default",
					Name:            "test-stage",
					ResourceVersion: "42",
				},
			},
			update: func(status *kargoapi.StageStatus) {
				status.LastHandledRefresh = "some-token"
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					_ context.Context,
					_ client.Client,
					_ string,
					obj client.Object,
					patch client.Patch,
					_ ...client.SubResourcePatchOption,
				) error {
					data, err := patch.Data(obj)
					require.NoError(t, err)
					var body map[string]any
					require.NoError(t, json.Unmarshal(data, &body))
					meta, ok := body["metadata"].(map[string]any)
					require.True(t, ok, "patch body must contain metadata")
					require.Equal(t, "42", meta["resourceVersion"])
					return nil
				},
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "conflict returns error",
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "default",
					Name:            "test-stage",
					ResourceVersion: "1",
				},
			},
			update: func(status *kargoapi.StageStatus) {
				status.LastHandledRefresh = "some-token"
			},
			interceptor: interceptor.Funcs{
				SubResourcePatch: func(
					_ context.Context,
					_ client.Client,
					_ string,
					_ client.Object,
					_ client.Patch,
					_ ...client.SubResourcePatchOption,
				) error {
					return apierrors.NewConflict(
						schema.GroupResource{
							Group:    kargoapi.GroupVersion.Group,
							Resource: "stages",
						},
						"test-stage",
						nil,
					)
				},
			},
			assert: func(t *testing.T, err error) {
				require.True(t, apierrors.IsConflict(err))
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.stage).
				WithStatusSubresource(tc.stage).
				WithInterceptorFuncs(tc.interceptor).
				Build()
			err := PatchStatus(t.Context(), c, tc.stage, tc.update)
			tc.assert(t, err)
		})
	}
}
