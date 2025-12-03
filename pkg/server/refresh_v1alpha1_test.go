package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestRefreshResource(t *testing.T) {
	testSets := map[string]struct {
		kClient    client.Client
		req        *svcv1alpha1.RefreshResourceRequest
		assertions func(*connect.Response[svcv1alpha1.RefreshResourceResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "",
				Name:    "test",
				Kind:    "Warehouse",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "project cannot be empty", "")
			},
		},
		"empty name": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "kargo-demo",
				Name:    "",
				Kind:    "Warehouse",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "name cannot be empty", "")
			},
		},
		"empty kind": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "kargo-demo",
				Name:    "test",
				Kind:    "",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "kind cannot be empty", "")
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "kargo-x",
				Name:    "test",
				Kind:    "Warehouse",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "project not found", "")
			},
		},
		"non-existing warehouse": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "non-existing-project",
				Name:    "test",
				Kind:    "Warehouse",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "Warehouse not found", "")
			},
		},
		"warehouse": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "kargo-demo",
				Name:    "test",
				Kind:    "Warehouse",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.NoError(t, err)
				var wh kargoapi.Warehouse
				require.NoError(t, json.Unmarshal(res.Msg.GetResource().Value, &wh))
				annotation := wh.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
				refreshTime, err := time.Parse(time.RFC3339, annotation)
				require.NoError(t, err)
				// Make sure we set timestamp is close to now
				// Assume it doesn't take 3 seconds to run this unit test.
				require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
				require.Equal(t, "kargo-demo", wh.Namespace)
				require.Equal(t, "test", wh.Name)
			},
		},
		"stage": {
			req: &svcv1alpha1.RefreshResourceRequest{
				Project: "kargo-demo",
				Name:    "test",
				Kind:    "Stage",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.NoError(t, err)
				var st kargoapi.Stage
				require.NoError(t, json.Unmarshal(res.Msg.GetResource().Value, &st))
				annotation := st.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
				refreshTime, err := time.Parse(time.RFC3339, annotation)
				require.NoError(t, err)
				// Make sure we set timestamp is close to now
				// Assume it doesn't take 3 seconds to run this unit test.
				require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
				require.Equal(t, "kargo-demo", st.Namespace)
				require.Equal(t, "test", st.Name)
			},
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						return fake.NewClientBuilder().
							WithScheme(scheme).
							WithObjects(
								mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
								mustNewObject[kargoapi.Stage]("testdata/stage.yaml"),
							).
							Build(), nil
					},
				},
			)
			require.NoError(t, err)
			svr := &server{client: client}
			svr.externalValidateProjectFn = validation.ValidateProject
			res, err := svr.RefreshResource(ctx, connect.NewRequest(ts.req))
			ts.assertions(res, err)
		})
	}
}
