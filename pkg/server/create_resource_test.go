package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	fakeevent "github.com/akuity/kargo/pkg/kubernetes/event/fake"
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
			{
				name: "denies Promotion creation without promote permission",
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						_ context.Context,
						_ string,
						gvr schema.GroupVersionResource,
						_ string,
						key client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							gvr.GroupResource(),
							key.Name,
							errors.New("not permitted to promote"),
						)
					}
				},
				body: mustJSONBody(&kargoapi.Promotion{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "Promotion",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promotion",
						Namespace: testProject.Name,
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
					// The Promotion must not have been created.
					err := c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: "fake-promotion"},
						&kargoapi.Promotion{},
					)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
			{
				name: "creates Promotion when promote is permitted",
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						context.Context,
						string,
						schema.GroupVersionResource,
						string,
						client.ObjectKey,
					) error {
						return nil
					}
				},
				body: mustJSONBody(&kargoapi.Promotion{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "Promotion",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promotion",
						Namespace: testProject.Name,
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)
					// The Promotion must have been created.
					require.NoError(t, c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: "fake-promotion"},
						&kargoapi.Promotion{},
					))
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
