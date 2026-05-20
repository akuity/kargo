package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestListPromotions(t *testing.T) {
	// We need some promotion names with ULIDs to test the sorting.
	oldestPromotionName := fmt.Sprintf("some-stage.%s.oldest", ulid.Make())
	olderPromotionName := fmt.Sprintf("some-stage.%s.older", ulid.Make())
	oldPromotionName := fmt.Sprintf("some-stage.%s.old", ulid.Make())
	newPromotionName := fmt.Sprintf("some-stage.%s.new", ulid.Make())

	testCases := map[string]struct {
		req              *svcv1alpha1.ListPromotionsRequest
		objects          []client.Object
		interceptorFuncs interceptor.Funcs
		assertions       func(*testing.T, *connect.Response[svcv1alpha1.ListPromotionsResponse], error)
	}{
		"empty project": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"existing project": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-promotion",
						Namespace: "kargo-demo",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotions(), 1)
			},
		},
		"uses list-level resourceVersion": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(
					ctx context.Context,
					cl client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					if err := cl.List(ctx, list, opts...); err != nil {
						return err
					}
					if pl, ok := list.(*kargoapi.PromotionList); ok {
						pl.ResourceVersion = "42"
					}
					return nil
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.Equal(t, "42", r.Msg.GetResourceVersion())
			},
		},
		"falls back to max item resourceVersion": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "promotion-1",
						Namespace: "kargo-demo",
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "promotion-2",
						Namespace: "kargo-demo",
					},
				},
			},
			interceptorFuncs: interceptor.Funcs{
				List: func(
					ctx context.Context,
					cl client.WithWatch,
					list client.ObjectList,
					opts ...client.ListOption,
				) error {
					if err := cl.List(ctx, list, opts...); err != nil {
						return err
					}
					if pl, ok := list.(*kargoapi.PromotionList); ok {
						pl.ResourceVersion = "0"
						for i := range pl.Items {
							pl.Items[i].ResourceVersion = "100"
						}
					}
					return nil
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.Equal(t, "100", r.Msg.GetResourceVersion())
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "non-existing-project",
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"orders by ULID and phase": {
			req: &svcv1alpha1.ListPromotionsRequest{
				Project: "kargo-demo",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      oldestPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseRunning,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseSucceeded,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      olderPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhaseFailed,
					},
				},
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      oldPromotionName,
						Namespace: "kargo-demo",
					},
					Status: kargoapi.PromotionStatus{
						Phase: kargoapi.PromotionPhasePending,
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListPromotionsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetPromotions(), 4)

				// Check that the analysis templates are ordered by ULID and phase.
				require.Equal(t, oldestPromotionName, r.Msg.GetPromotions()[0].GetName())
				require.Equal(t, oldPromotionName, r.Msg.GetPromotions()[1].GetName())
				require.Equal(t, newPromotionName, r.Msg.GetPromotions()[2].GetName())
				require.Equal(t, olderPromotionName, r.Msg.GetPromotions()[3].GetName())
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
						_ string,
					) (client.WithWatch, error) {
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						if testCase.interceptorFuncs.List != nil {
							c.WithInterceptorFuncs(testCase.interceptorFuncs)
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
			res, err := (svr).ListPromotions(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_listPromotions(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/promotions",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Promotions exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					promos := &kargoapi.PromotionList{}
					err := json.Unmarshal(w.Body.Bytes(), promos)
					require.NoError(t, err)
					require.Empty(t, promos.Items)
				},
			},
			{
				name: "lists Promotions",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "promotion-1",
						},
					},
					&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "promotion-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Promotions in the response
					promos := &kargoapi.PromotionList{}
					err := json.Unmarshal(w.Body.Bytes(), promos)
					require.NoError(t, err)
					require.Len(t, promos.Items, 2)
				},
			},
			{
				name: "sets effective resourceVersion",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(
						testProject,
						&kargoapi.Promotion{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: testProject.Name,
								Name:      "promotion-1",
							},
						},
					).
					WithInterceptorFuncs(interceptor.Funcs{
						List: func(
							ctx context.Context,
							cl client.WithWatch,
							list client.ObjectList,
							opts ...client.ListOption,
						) error {
							if err := cl.List(ctx, list, opts...); err != nil {
								return err
							}
							if pl, ok := list.(*kargoapi.PromotionList); ok {
								pl.ResourceVersion = "0"
								for i := range pl.Items {
									pl.Items[i].ResourceVersion = "100"
								}
							}
							return nil
						},
					}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					promos := &kargoapi.PromotionList{}
					err := json.Unmarshal(w.Body.Bytes(), promos)
					require.NoError(t, err)
					require.Equal(t, "100", promos.ResourceVersion)
				},
			},
		},
	)
}

func Test_server_listPromotions_watch(t *testing.T) {
	const projectName = "fake-project"

	resourceVersionTestCase := func() restWatchTestCase {
		resourceVersionCh := make(chan string, 1)
		return restWatchTestCase{
			name: "passes resourceVersion to watch",
			url:  "/v1beta1/projects/" + projectName + "/promotions?watch=true&resourceVersion=123",
			clientBuilder: fake.NewClientBuilder().
				WithObjects(&kargoapi.Project{
					ObjectMeta: metav1.ObjectMeta{Name: projectName},
				}).
				WithInterceptorFuncs(interceptor.Funcs{
					Watch: func(
						ctx context.Context,
						cl client.WithWatch,
						obj client.ObjectList,
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
						return cl.Watch(ctx, obj, opts...)
					},
				}),
			assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
				require.Equal(t, http.StatusOK, w.Code)
				select {
				case rv := <-resourceVersionCh:
					require.Equal(t, "123", rv)
				case <-time.After(time.Second):
					require.Fail(t, "watch was not called")
				}
			},
		}
	}()

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/promotions?watch=true",
		[]restWatchTestCase{
			{
				name:          "project not found",
				url:           "/v1beta1/projects/non-existent/promotions?watch=true",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches all promotions successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "promotion-1",
						},
					},
					&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "promotion-2",
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Create a new promotion to trigger a watch event
					newPromo := &kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      "promotion-3",
						},
					}
					_ = c.Create(ctx, newPromo)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain SSE events from the create operation
					body := w.Body.String()
					require.Contains(t, body, "data:")
				},
			},
			resourceVersionTestCase,
			{
				name: "watches empty promotion list",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers are set
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
				},
			},
		},
	)
}
