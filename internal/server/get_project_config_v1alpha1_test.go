package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/server/kubernetes"
)

func TestGetProjectConfig(t *testing.T) {
	testCases := map[string]struct {
		req         *svcv1alpha1.GetProjectConfigRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetProjectConfigResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Name: "",
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing ProjectConfig": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Name: "kargo-x",
			},
			objects: []client.Object{},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing ProjectConfig": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Name: "kargo-demo",
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetRaw())

				require.NotNil(t, r.Msg.GetProjectConfig())
				require.Equal(t, "kargo-demo", r.Msg.GetProjectConfig().Name)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Name:   "kargo-demo",
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProjectConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								AutoPromotionEnabled: true,
								StageSelector: &kargoapi.PromotionPolicySelector{
									Name: "foo",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetProjectConfig())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)

				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ProjectConfig)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, 1, len(tObj.Spec.PromotionPolicies))
				require.Equal(t, true, tObj.Spec.PromotionPolicies[0].AutoPromotionEnabled)
				require.Equal(t, "foo", tObj.Spec.PromotionPolicies[0].StageSelector.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetProjectConfigRequest{
				Name:   "kargo-demo",
				Format: svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				&kargoapi.ProjectConfig{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ProjectConfig",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kargo-demo",
						Namespace: "kargo-demo",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								AutoPromotionEnabled: true,
								StageSelector: &kargoapi.PromotionPolicySelector{
									Name: "foo",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetProjectConfigResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetProjectConfig())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.ProjectConfig)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, 1, len(tObj.Spec.PromotionPolicies))
				require.Equal(t, true, tObj.Spec.PromotionPolicies[0].AutoPromotionEnabled)
				require.Equal(t, "foo", tObj.Spec.PromotionPolicies[0].StageSelector.Name)
			},
		},
	}

	for name, testCase := range testCases {
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
						c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(testCase.interceptor)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)

			require.NoError(t, err)

			svr := &server{
				client: client,
			}

			res, err := (svr).GetProjectConfig(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
