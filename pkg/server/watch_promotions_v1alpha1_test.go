package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/api/service/v1alpha1/svcv1alpha1connect"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestWatchPromotions_resourceVersion(t *testing.T) {
	t.Parallel()

	const projectName = "fake-project"

	resourceVersionCh := make(chan string, 1)

	fakeClient := fake.NewClientBuilder().
		WithScheme(mustNewScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			Watch: func(
				_ context.Context,
				_ client.WithWatch,
				_ client.ObjectList,
				opts ...client.ListOption,
			) (watch.Interface, error) {
				var listOpts client.ListOptions
				for _, opt := range opts {
					opt.ApplyToList(&listOpts)
				}
				if listOpts.Raw == nil {
					resourceVersionCh <- ""
				} else {
					resourceVersionCh <- listOpts.Raw.ResourceVersion
				}

				w := watch.NewFake()
				go func() {
					time.Sleep(10 * time.Millisecond)
					w.Add(&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "promotion-1",
						},
					})
				}()
				return w, nil
			},
		}).
		Build()

	k8sClient, err := kubernetes.NewClient(
		t.Context(),
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(
				context.Context,
				*rest.Config,
				*runtime.Scheme,
				string,
			) (client.WithWatch, error) {
				return fakeClient, nil
			},
		},
	)
	require.NoError(t, err)

	svr := &server{client: k8sClient}
	svr.externalValidateProjectFn = func(_ context.Context, _ client.Client, project string) error {
		if project != projectName {
			return validation.ErrProjectNotFound
		}
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(svcv1alpha1connect.NewKargoServiceHandler(svr))
	httpSrv := httptest.NewServer(mux)
	t.Cleanup(httpSrv.Close)

	cli := svcv1alpha1connect.NewKargoServiceClient(httpSrv.Client(), httpSrv.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	stream, err := cli.WatchPromotions(ctx, connect.NewRequest(&svcv1alpha1.WatchPromotionsRequest{
		Project:         projectName,
		ResourceVersion: "123",
	}))
	require.NoError(t, err)
	require.True(t, stream.Receive())
	require.Equal(t, "promotion-1", stream.Msg().GetPromotion().GetName())

	select {
	case rv := <-resourceVersionCh:
		require.Equal(t, "123", rv)
	default:
		require.Fail(t, "watch was not called")
	}
}
