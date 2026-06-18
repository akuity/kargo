package server

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

// newSeedKubeClient builds a kubernetes.Client whose internal client (and thus
// its uncached APIReader) is the supplied fake.
func newSeedKubeClient(t *testing.T, internalClient client.WithWatch) kubernetes.Client {
	t.Helper()
	kubeClient, err := kubernetes.NewClient(
		t.Context(),
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			Scheme:            mustNewScheme(),
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
				string,
			) (client.WithWatch, error) {
				return internalClient, nil
			},
		},
	)
	require.NoError(t, err)
	return kubeClient
}

func TestServer_listForWatchSeed(t *testing.T) {
	t.Parallel()

	scheme := mustNewScheme()

	t.Run("authorizes and lists through the API reader", func(t *testing.T) {
		t.Parallel()

		var (
			authorized     bool
			authorizedVerb string
			authorizedGVR  schema.GroupVersionResource
			authorizedSub  string
			authorizedKey  client.ObjectKey
			listCalled     bool
		)
		internalClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(
					ctx context.Context,
					cl client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					listCalled = true
					if err := cl.List(ctx, list, opts...); err != nil {
						return err
					}
					if promotions, ok := list.(*kargoapi.PromotionList); ok {
						promotions.ResourceVersion = "42"
					}
					return nil
				},
			}).
			Build()

		s := &server{
			client: newSeedKubeClient(t, internalClient),
			authorizeFn: func(
				_ context.Context,
				verb string,
				gvr schema.GroupVersionResource,
				subresource string,
				key client.ObjectKey,
			) error {
				authorized = true
				authorizedVerb = verb
				authorizedGVR = gvr
				authorizedSub = subresource
				authorizedKey = key
				return nil
			},
		}

		promotions := &kargoapi.PromotionList{}
		err := s.listForWatchSeed(
			t.Context(),
			"promotions",
			promotions,
			client.InNamespace("fake-project"),
		)
		require.NoError(t, err)
		require.True(t, authorized)
		require.True(t, listCalled)
		require.Equal(t, "list", authorizedVerb)
		require.Empty(t, authorizedSub)
		require.Equal(t, kargoapi.GroupVersion.WithResource("promotions"), authorizedGVR)
		require.Equal(t, client.ObjectKey{Namespace: "fake-project"}, authorizedKey)
		require.Equal(t, "42", promotions.ResourceVersion)
	})

	t.Run("authorization failure short-circuits the list", func(t *testing.T) {
		t.Parallel()

		authErr := errors.New("not authorized")
		var listCalled bool
		internalClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					listCalled = true
					return nil
				},
			}).
			Build()

		s := &server{
			client: newSeedKubeClient(t, internalClient),
			authorizeFn: func(
				context.Context,
				string,
				schema.GroupVersionResource,
				string,
				client.ObjectKey,
			) error {
				return authErr
			},
		}

		err := s.listForWatchSeed(
			t.Context(),
			"promotions",
			&kargoapi.PromotionList{},
			client.InNamespace("fake-project"),
		)
		require.ErrorIs(t, err, authErr)
		require.False(t, listCalled)
	})

	t.Run("errors when authorize function is not configured", func(t *testing.T) {
		t.Parallel()

		s := &server{
			client: newSeedKubeClient(t, fake.NewClientBuilder().WithScheme(scheme).Build()),
		}

		err := s.listForWatchSeed(
			t.Context(),
			"promotions",
			&kargoapi.PromotionList{},
			client.InNamespace("fake-project"),
		)
		require.ErrorContains(t, err, "authorize function is not configured")
	})
}
