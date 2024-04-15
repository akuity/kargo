package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/kubernetes"
	"github.com/akuity/kargo/internal/api/user"
	"github.com/akuity/kargo/internal/api/validation"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func TestGetFreight(t *testing.T) {
	testCases := map[string]struct {
		req                       *svcv1alpha1.GetFreightRequest
		objects                   []client.Object
		getFreightByNameOrAliasFn func(
			ctx context.Context,
			c client.Client,
			project string,
			name string,
			alias string,
		) (*kargoapi.Freight, error)
		assertions func(*testing.T, *connect.Response[svcv1alpha1.GetFreightResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "",
				Name:    "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"empty name and alias": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "",
				Alias:   "",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-empty name and alias": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "name",
				Alias:   "alias",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-x",
				Name:    "test",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"existing Freight by name": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			getFreightByNameOrAliasFn: kargoapi.GetFreightByNameOrAlias,
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetFreight())
				require.Equal(t, "kargo-demo", c.Msg.GetFreight().Namespace)
				require.Equal(t, "test", c.Msg.GetFreight().Name)
			},
		},
		"existing Freight by alias": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Alias:   "test-alias",
			},
			getFreightByNameOrAliasFn: kargoapi.GetFreightByNameOrAlias,
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Freight{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
						Labels: map[string]string{
							kargoapi.AliasLabelKey: "test-alias",
						},
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetRaw())

				require.NotNil(t, c.Msg.GetFreight())
				require.Equal(t, "kargo-demo", c.Msg.GetFreight().Namespace)
				require.Equal(t, "test", c.Msg.GetFreight().Name)
			},
		},
		"non-existing Freight by name": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "non-existing",
			},
			getFreightByNameOrAliasFn: kargoapi.GetFreightByNameOrAlias,
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"non-existing Freight by alias": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Alias:   "non-existing",
			},
			getFreightByNameOrAliasFn: kargoapi.GetFreightByNameOrAlias,
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"error getting Freight": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "test",
			},
			getFreightByNameOrAliasFn: func(
				context.Context,
				client.Client,
				string,
				string,
				string,
			) (*kargoapi.Freight, error) {
				return nil, apierrors.NewServiceUnavailable("test")
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
				require.Nil(t, c)
			},
		},
		"raw format JSON": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_JSON,
			},
			getFreightByNameOrAliasFn: kargoapi.GetFreightByNameOrAlias,
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Freight{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Freight",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetFreight())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.Freight)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
		"raw format YAML": {
			req: &svcv1alpha1.GetFreightRequest{
				Project: "kargo-demo",
				Name:    "test",
				Format:  svcv1alpha1.RawFormat_RAW_FORMAT_YAML,
			},
			getFreightByNameOrAliasFn: kargoapi.GetFreightByNameOrAlias,
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Freight{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Freight",
						APIVersion: kargoapi.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "kargo-demo",
						Name:      "test",
					},
				},
			},
			assertions: func(t *testing.T, c *connect.Response[svcv1alpha1.GetFreightResponse], err error) {
				require.NoError(t, err)

				require.NotNil(t, c)
				require.Nil(t, c.Msg.GetFreight())
				require.NotNil(t, c.Msg.GetRaw())

				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))

				obj, _, err := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode(
					c.Msg.GetRaw(),
					nil,
					nil,
				)
				require.NoError(t, err)
				tObj, ok := obj.(*kargoapi.Freight)
				require.True(t, ok)
				require.Equal(t, "kargo-demo", tObj.Namespace)
				require.Equal(t, "test", tObj.Name)
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Simulate an admin user to prevent any authz issues with the authorizing
			// client.
			ctx := user.ContextWithInfo(
				context.Background(),
				user.Info{
					IsAdmin: true,
				},
			)

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.Client, error) {
						c := fake.NewClientBuilder().WithScheme(scheme)
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
				getFreightByNameOrAliasFn: testCase.getFreightByNameOrAliasFn,
			}
			res, err := (svr).GetFreight(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}
