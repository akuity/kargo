package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestRefreshResource(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))
	require.NoError(t, kargoapi.AddToScheme(testScheme))

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kargo-demo",
			Labels: map[string]string{
				kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
			},
		},
	}

	testSets := map[string]struct {
		kClient    client.WithWatch
		req        *svcv1alpha1.RefreshResourceRequest
		assertions func(*connect.Response[svcv1alpha1.RefreshResourceResponse], error)
	}{
		"empty project": {
			kClient: fake.NewClientBuilder().WithScheme(testScheme).Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "",
				Name:         "test",
				ResourceType: RefreshResourceTypeWarehouse.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Nil(t, res)
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "project should not be empty", "")
			},
		},
		"empty name": {
			kClient: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(ns).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "",
				ResourceType: RefreshResourceTypeWarehouse.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Nil(t, res)
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "name should not be empty", "")
			},
		},
		"empty resource type": {
			kClient: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(ns).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "test",
				ResourceType: "",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Nil(t, res)
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContains(t, err, "resource type is unset")
			},
		},
		"inavalid resource type": {
			kClient: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(ns).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "test",
				ResourceType: "invalid",
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Nil(t, res)
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContains(t, err, "invalid_argument: \"invalid\" is unsupported as a refresh resource type")
			},
		},
		"non-existing project": {
			kClient: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(ns).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "not-existing-project",
				Name:         "test",
				ResourceType: RefreshResourceTypeWarehouse.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Nil(t, res)
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "project not found", "")
			},
		},
		"resource not found": {
			kClient: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(ns).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "test",
				ResourceType: RefreshResourceTypeWarehouse.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.Nil(t, res)
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.ErrorContainsf(t, err, "Warehouse not found", "")
			},
		},
		"warehouse": {
			kClient: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(ns,
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "kargo-demo",
							Name:      "test",
						},
						Spec: kargoapi.WarehouseSpec{},
					},
				).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "test",
				ResourceType: RefreshResourceTypeWarehouse.String(),
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
		"stage without current promo": {
			kClient: fake.NewClientBuilder().
				WithObjects(ns,
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "kargo-demo",
							Name:      "test",
						},
					},
				).WithScheme(testScheme).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "test",
				ResourceType: RefreshResourceTypeStage.String(),
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
		"stage with current promo": {
			kClient: fake.NewClientBuilder().WithObjects(ns,
				&kargoapi.Stage{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
					Status: kargoapi.StageStatus{
						CurrentPromotion: &kargoapi.PromotionReference{
							Name: "promo-1",
						},
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "promo-1",
					},
				},
			).WithScheme(testScheme).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				Name:         "test",
				ResourceType: RefreshResourceTypeStage.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.NoError(t, err)
				var p kargoapi.Promotion
				require.NoError(t, json.Unmarshal(res.Msg.GetResource().Value, &p))
				annotation := p.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
				refreshTime, err := time.Parse(time.RFC3339, annotation)
				require.NoError(t, err)
				// Make sure we set timestamp is close to now
				// Assume it doesn't take 3 seconds to run this unit test.
				require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
				require.Equal(t, "kargo-demo", p.Namespace)
				require.Equal(t, "test", p.Name)
			},
		},
		"cluster config": {
			kClient: fake.NewClientBuilder().WithScheme(testScheme).
				WithObjects(
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name: api.ClusterConfigName,
						},
					},
				).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "",
				Name:         api.ClusterConfigName,
				ResourceType: RefreshResourceTypeClusterConfig.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.NoError(t, err)
				var cc kargoapi.ClusterConfig
				require.NoError(t, json.Unmarshal(res.Msg.GetResource().Value, &cc))
				annotation := cc.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
				refreshTime, err := time.Parse(time.RFC3339, annotation)
				require.NoError(t, err)
				// Make sure we set timestamp is close to now
				// Assume it doesn't take 3 seconds to run this unit test.
				require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
				require.Equal(t, api.ClusterConfigName, cc.Name)
			},
		},
		"project config": {
			kClient: fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(ns,
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "kargo-demo",
							Name:      "kargo-demo",
						},
					},
				).
				Build(),
			req: &svcv1alpha1.RefreshResourceRequest{
				Project:      "kargo-demo",
				ResourceType: RefreshResourceTypeProjectConfig.String(),
			},
			assertions: func(res *connect.Response[svcv1alpha1.RefreshResourceResponse], err error) {
				require.NoError(t, err)
				var pc kargoapi.ProjectConfig
				require.NoError(t, json.Unmarshal(res.Msg.GetResource().Value, &pc))
				annotation := pc.GetAnnotations()[kargoapi.AnnotationKeyRefresh]
				refreshTime, err := time.Parse(time.RFC3339, annotation)
				require.NoError(t, err)
				// Make sure we set timestamp is close to now
				// Assume it doesn't take 3 seconds to run this unit test.
				require.WithinDuration(t, time.Now(), refreshTime, 3*time.Second)
				require.Equal(t, "kargo-demo", pc.Namespace)
			},
		},
	}
	for name, ts := range testSets {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						_ *runtime.Scheme,
					) (client.WithWatch, error) {
						return ts.kClient, nil
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
