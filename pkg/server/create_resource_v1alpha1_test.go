package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/user"
)

func Test_server_createResources(t *testing.T) {
	testProject := &kargoapi.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Warehouse",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodPost, "/v1beta1/resources",
		[]restTestCase{
			{
				name: "empty request body",
				body: bytes.NewBufferString(""),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid YAML in request body",
				body: bytes.NewBufferString("invalid: [unclosed sequence"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "resource already exists",
				body:          mustJSONBody(testProject),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates resources from JSON",
				body: mustJSONArrayBody(testProject, testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the response
					var res createResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Examine the Project in the response
					resProject := res.Results[0].CreatedResourceManifest
					require.Equal(t, testProject.APIVersion, resProject["apiVersion"])
					require.Equal(t, testProject.Kind, resProject["kind"])
					resProjectMeta := resProject["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testProject.Name, resProjectMeta["name"])

					// Examine the Warehouse in the response
					resWarehouse := res.Results[1].CreatedResourceManifest
					require.Equal(t, testWarehouse.APIVersion, resWarehouse["apiVersion"])
					require.Equal(t, testWarehouse.Kind, resWarehouse["kind"])
					resWarehouseMeta := resWarehouse["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testWarehouse.Name, resWarehouseMeta["name"])
					require.Equal(t, testWarehouse.Namespace, resWarehouseMeta["namespace"])

					// Verify the Project was created in the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)

					// Verify the Warehouse was created in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
				},
			},
			{
				name: "creates resources from YAML",
				body: mustYAMLBody(testProject, testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the response
					var res createResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Examine the Project in the response
					resProject := res.Results[0].CreatedResourceManifest
					require.Equal(t, testProject.APIVersion, resProject["apiVersion"])
					require.Equal(t, testProject.Kind, resProject["kind"])
					resProjectMeta := resProject["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testProject.Name, resProjectMeta["name"])

					// Examine the Warehouse in the response
					resWarehouse := res.Results[1].CreatedResourceManifest
					require.Equal(t, testWarehouse.APIVersion, resWarehouse["apiVersion"])
					require.Equal(t, testWarehouse.Kind, resWarehouse["kind"])
					resWarehouseMeta := resWarehouse["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testWarehouse.Name, resWarehouseMeta["name"])
					require.Equal(t, testWarehouse.Namespace, resWarehouseMeta["namespace"])

					// Verify the Project was created in the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)

					// Verify the Warehouse was created in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
				},
			},
			{
				name:          "partial failure",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONArrayBody(
					testProject, // Already exists
					testWarehouse,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the response
					var res createResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)

					// First result (Project) should have error
					require.NotEmpty(t, res.Results[0].Error)
					require.Contains(t, res.Results[0].Error, "already exists")

					// Second result (Warehouse) should succeed
					require.Empty(t, res.Results[1].Error)

					// Verify the Warehouse was created in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
				},
			},
		},
	)
}

type errSender struct{ err error }

func (s *errSender) Send(_ context.Context, _ event.Meta) error { return s.err }
func (s *errSender) Shutdown()                                  {}

func Test_server_createResources_freightEvent(t *testing.T) {
	testFreight := &kargoapi.Freight{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Freight",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-freight",
			Namespace: "fake-project",
		},
	}

	// recorder is reassigned by each serverSetup before assertions reads it.
	// Test cases are run sequentially so this is safe.
	var recorder *fakeevent.EventRecorder

	testRESTEndpoint(
		t, nil,
		http.MethodPost, "/v1beta1/resources",
		[]restTestCase{
			{
				name: "non-Freight resource does not send event",
				serverSetup: func(_ *testing.T, s *server) {
					recorder = fakeevent.NewEventRecorder(1)
					s.sender = k8sevent.NewEventSender(recorder)
				},
				body: mustJSONBody(&kargoapi.Warehouse{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "Warehouse",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-warehouse",
						Namespace: "fake-project",
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					require.Empty(t, recorder.Events)
				},
			},
			{
				name: "Freight without sender succeeds without event",
				body: mustJSONBody(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
				},
			},
			{
				name: "Freight with sender sends FreightCreated event",
				serverSetup: func(_ *testing.T, s *server) {
					recorder = fakeevent.NewEventRecorder(1)
					s.sender = k8sevent.NewEventSender(recorder)
				},
				body: mustJSONBody(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					require.Len(t, recorder.Events, 1)
					evt := <-recorder.Events
					require.Equal(t, corev1.EventTypeNormal, evt.EventType)
					require.Equal(t, string(kargoapi.EventTypeFreightCreated), evt.Reason)
					require.Equal(t, "Freight created", evt.Message)
				},
			},
			{
				name: "Freight with sender error still succeeds",
				serverSetup: func(_ *testing.T, s *server) {
					s.sender = &errSender{err: errors.New("send failed")}
				},
				body: mustJSONBody(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
				},
			},
			{
				name: "Freight with user context includes actor in event message",
				serverSetup: func(_ *testing.T, s *server) {
					recorder = fakeevent.NewEventRecorder(1)
					s.sender = k8sevent.NewEventSender(recorder)
				},
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithInfo(ctx, user.Info{IsAdmin: true})
				},
				body: mustJSONBody(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					require.Len(t, recorder.Events, 1)
					evt := <-recorder.Events
					require.Equal(t, string(kargoapi.EventTypeFreightCreated), evt.Reason)
					require.Contains(t, evt.Message, kargoapi.EventActorAdmin)
				},
			},
		},
	)
}

// authGatedClient is a hand-written test double for kubernetes.Client. This
// branch's real authorizing client submits a SubjectAccessReview through a
// brand-new client built from GetRestConfig, which needs a live cluster and
// can't be faked with a store interceptor. This double instead denies Create
// directly for the kinds named in deniedCreateKinds -- simulating what a real
// SubjectAccessReview would deny for kargo-project-creator -- while
// InternalClient() returns the same backing store unrestricted, matching
// real bypass behavior. Every other method (Get, Update, IsObjectNamespaced,
// etc.) is promoted straight from the embedded store.
type authGatedClient struct {
	client.WithWatch
	deniedCreateKinds map[string]bool
}

func (a *authGatedClient) Create(
	ctx context.Context,
	obj client.Object,
	opts ...client.CreateOption,
) error {
	if a.deniedCreateKinds[obj.GetObjectKind().GroupVersionKind().Kind] {
		gvk := obj.GetObjectKind().GroupVersionKind()
		return apierrors.NewForbidden(
			schema.GroupResource{Group: gvk.Group, Resource: strings.ToLower(gvk.Kind) + "s"},
			obj.GetName(),
			errors.New("create is not permitted"),
		)
	}
	return a.WithWatch.Create(ctx, obj, opts...)
}

func (a *authGatedClient) InternalClient() client.WithWatch { return a.WithWatch }

func (a *authGatedClient) Authorize(
	context.Context, string, schema.GroupVersionResource, string, client.ObjectKey,
) error {
	return nil
}

func (a *authGatedClient) APIReader() client.Reader { return a.WithWatch }

// newProjectCreatorClient builds an authGatedClient that denies Create for
// ClusterConfig/ClusterPromotionTask, mirroring the chart's
// kargo-project-creator ClusterRole (full access to Project, read-only on
// those two cluster-scoped kinds). Also returns the raw store so the caller
// can inspect it afterward.
func newProjectCreatorClient(
	t *testing.T,
	store *fake.ClientBuilder,
) (kubernetes.Client, client.WithWatch) {
	t.Helper()
	scheme := newTestScheme(t)
	internalClient := store.
		WithScheme(scheme).
		WithRESTMapper(testRESTMapper(scheme)).
		Build()
	return &authGatedClient{
		WithWatch: internalClient,
		deniedCreateKinds: map[string]bool{
			"ClusterConfig":        true,
			"ClusterPromotionTask": true,
		},
	}, internalClient
}

// Test_server_createResources_clusterScopedNamespaceSpoofing is a regression
// test: a principal who could only create Projects could spoof a
// cluster-scoped resource's metadata.namespace to a just-created Project's
// name and get it created anyway, since Kubernetes ignores metadata.namespace
// for cluster-scoped kinds.
func Test_server_createResources_clusterScopedNamespaceSpoofing(t *testing.T) {
	const projectName = "ghsa-repro"

	// capturedClient is reassigned by each serverSetup before assertions reads
	// it. Test cases are run sequentially, so this is safe (see the same
	// pattern used for the freight event recorder above).
	var capturedClient client.WithWatch

	testRESTEndpoint(
		t, nil,
		http.MethodPost, "/v1beta1/resources",
		[]restTestCase{
			{
				name: "control: direct create of a cluster-scoped resource is denied",
				serverSetup: func(t *testing.T, s *server) {
					s.client, capturedClient = newProjectCreatorClient(t, fake.NewClientBuilder())
					s.authorizeFn = s.client.Authorize
				},
				body: mustJSONBody(&kargoapi.ClusterPromotionTask{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "ClusterPromotionTask",
					},
					ObjectMeta: metav1.ObjectMeta{Name: "control-task"},
					Spec: kargoapi.PromotionTaskSpec{
						Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "exploit: ClusterPromotionTask namespace spoofed to a just-created Project is still denied",
				serverSetup: func(t *testing.T, s *server) {
					s.client, capturedClient = newProjectCreatorClient(t, fake.NewClientBuilder())
					s.authorizeFn = s.client.Authorize
				},
				body: mustJSONArrayBody(
					&kargoapi.Project{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Project",
						},
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.ClusterPromotionTask{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "ClusterPromotionTask",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "exploit-task",
							// The exploit: meaningless to Kubernetes for a cluster-scoped kind.
							Namespace: projectName,
						},
						Spec: kargoapi.PromotionTaskSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					var res createResourceResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error, "Project creation should succeed")
					require.NotEmpty(
						t, res.Results[1].Error,
						"ClusterPromotionTask creation must be denied, not silently bypassed via a spoofed namespace",
					)
					require.Contains(t, res.Results[1].Error, "forbidden")

					err := capturedClient.Get(
						t.Context(),
						client.ObjectKey{Name: "exploit-task"},
						&kargoapi.ClusterPromotionTask{},
					)
					require.True(t, apierrors.IsNotFound(err), "ClusterPromotionTask must not have been persisted")
				},
			},
		},
	)
}

// mustManifest marshals objs to a multi-document YAML manifest, the shape the
// legacy Connect-RPC resource endpoints accept as raw bytes.
func mustManifest(t *testing.T, objs ...any) []byte {
	t.Helper()
	b, err := io.ReadAll(mustYAMLBody(objs...))
	require.NoError(t, err)
	return b
}

// TestCreateResource_clusterScopedNamespaceSpoofing is the legacy Connect-RPC
// CreateResource counterpart to
// Test_server_createResources_clusterScopedNamespaceSpoofing above; see that
// test's doc comment for the full explanation.
func TestCreateResource_clusterScopedNamespaceSpoofing(t *testing.T) {
	const projectName = "ghsa-repro-rpc"

	t.Run("control: direct create of a cluster-scoped resource is denied", func(t *testing.T) {
		s := &server{}
		s.client, _ = newProjectCreatorClient(t, fake.NewClientBuilder())

		_, err := s.CreateResource(t.Context(), connect.NewRequest(&svcv1alpha1.CreateResourceRequest{
			Manifest: mustManifest(t, &kargoapi.ClusterPromotionTask{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kargoapi.GroupVersion.String(),
					Kind:       "ClusterPromotionTask",
				},
				ObjectMeta: metav1.ObjectMeta{Name: "control-task-rpc"},
				Spec: kargoapi.PromotionTaskSpec{
					Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
				},
			}),
		}))
		require.Error(t, err)
		require.True(t, apierrors.IsForbidden(err))
	})

	t.Run(
		"exploit: ClusterPromotionTask namespace spoofed to a just-created Project is still denied",
		func(t *testing.T) {
			s := &server{}
			var capturedClient client.WithWatch
			s.client, capturedClient = newProjectCreatorClient(t, fake.NewClientBuilder())

			resp, err := s.CreateResource(t.Context(), connect.NewRequest(&svcv1alpha1.CreateResourceRequest{
				Manifest: mustManifest(
					t,
					&kargoapi.Project{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Project",
						},
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.ClusterPromotionTask{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "ClusterPromotionTask",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "exploit-task-rpc",
							Namespace: projectName,
						},
						Spec: kargoapi.PromotionTaskSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
						},
					},
				),
			}))
			require.NoError(t, err)
			require.Len(t, resp.Msg.Results, 2)
			require.Empty(t, resp.Msg.Results[0].GetError(), "Project creation should succeed")
			require.NotEmpty(
				t, resp.Msg.Results[1].GetError(),
				"ClusterPromotionTask creation must be denied, not silently bypassed via a spoofed namespace",
			)
			require.Contains(t, resp.Msg.Results[1].GetError(), "forbidden")

			getErr := capturedClient.Get(
				t.Context(),
				client.ObjectKey{Name: "exploit-task-rpc"},
				&kargoapi.ClusterPromotionTask{},
			)
			require.True(t, apierrors.IsNotFound(getErr), "ClusterPromotionTask must not have been persisted")
		},
	)
}
