package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/server/kubernetes"
	"github.com/akuity/kargo/internal/server/validation"
)

func TestGetConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req         *svcv1alpha1.GetConfigMapRequest
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *connect.Response[svcv1alpha1.GetConfigMapResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Project: "",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"non-existing ConfigMap": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Name:    "kargo-x",
				Project: "kargo-y",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing ConfigMap": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "cm-1",
						Namespace: "kargo-demo",
					},
					Data: map[string]string{
						"foo": "bar",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetRaw())

				require.NotNil(t, r.Msg.GetConfigMap())

				require.Equal(t, "kargo-demo", r.Msg.GetConfigMap().Namespace)
				require.Equal(t, "cm-1", r.Msg.GetConfigMap().Name)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			objects: []client.Object{
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetConfigMap())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()

				require.NoError(t, corev1.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)

				require.NoError(t, err)

				tObj, ok := obj.(*corev1.ConfigMap)
				require.True(t, ok)
				require.Equal(t, "cm-1", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "bar", tObj.Data["foo"])
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetConfigMapRequest{
				Project: "kargo-demo",
				Name:    "cm-1",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			objects: []client.Object{
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.GetConfigMapResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, r)
				require.Nil(t, r.Msg.GetConfigMap())
				require.NotNil(t, r.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, corev1.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					r.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*corev1.ConfigMap)
				require.True(t, ok)
				require.Equal(t, "cm-1", tObj.Name)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "bar", tObj.Data["foo"])
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
						s *runtime.Scheme,
					) (client.Client, error) {
						c := fake.NewClientBuilder().WithScheme(s).WithInterceptorFuncs(testCase.interceptor)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)

			require.NoError(t, err)

			svr := &server{
				client:                    client,
				externalValidateProjectFn: validation.ValidateProject,
			}

			res, err := (svr).GetConfigMap(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
