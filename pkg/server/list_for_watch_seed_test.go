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

func TestServer_listForWatchSeed(t *testing.T) {
	t.Parallel()

	scheme := mustNewScheme()
	internalClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	kubeClient, err := kubernetes.NewClient(
		t.Context(),
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			Scheme:            scheme,
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

	t.Run("authorizes and lists through direct reader", func(t *testing.T) {
		t.Parallel()

		var (
			authorized       bool
			authorizedVerb   string
			authorizedGVR    schema.GroupVersionResource
			authorizedSub    string
			authorizedKey    client.ObjectKey
			directListCalled bool
		)
		directReader := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(
					ctx context.Context,
					cl client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					directListCalled = true
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
			client:       kubeClient,
			directReader: directReader,
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
		require.True(t, directListCalled)
		require.Equal(t, "list", authorizedVerb)
		require.Empty(t, authorizedSub)
		require.Equal(t, kargoapi.GroupVersion.WithResource("promotions"), authorizedGVR)
		require.Equal(t, client.ObjectKey{Namespace: "fake-project"}, authorizedKey)
		require.Equal(t, "42", promotions.ResourceVersion)
	})

	t.Run("authorization failure short-circuits direct list", func(t *testing.T) {
		t.Parallel()

		authErr := errors.New("not authorized")
		var directListCalled bool
		directReader := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					directListCalled = true
					return nil
				},
			}).
			Build()

		s := &server{
			client:       kubeClient,
			directReader: directReader,
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
		require.False(t, directListCalled)
	})

	t.Run("falls back to cached client when no direct reader is wired", func(t *testing.T) {
		t.Parallel()

		var authorizeCalled bool
		s := &server{
			client: kubeClient,
			authorizeFn: func(
				context.Context,
				string,
				schema.GroupVersionResource,
				string,
				client.ObjectKey,
			) error {
				authorizeCalled = true
				return nil
			},
		}

		// Should not error and should not invoke authorizeFn directly;
		// the cached authorizing client performs its own SAR per call,
		// which is bypassed here via SkipAuthorization on kubeClient.
		err := s.listForWatchSeed(
			t.Context(),
			"promotions",
			&kargoapi.PromotionList{},
			client.InNamespace("fake-project"),
		)
		require.NoError(t, err)
		require.False(t, authorizeCalled)
	})
}
