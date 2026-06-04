package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
)

func Test_server_rejectFreight(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-freight",
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Alias: "fake-alias",
	}

	testRESTEndpoint(
		t,
		&config.ServerConfig{},
		http.MethodPost,
		"/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name+"/reject",
		[]restTestCase{
			{
				name: "Project not found",
				body: bytes.NewBufferString(`{}`),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Freight not found",
				body: bytes.NewBufferString(`{}`),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "reason too long",
				body: bytes.NewBufferString(`{"reason":"` + strings.Repeat("x", maxFreightRejectionReasonLength+1) + `"}`),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "not authorized",
				body: bytes.NewBufferString(`{}`),
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = func(
						_ context.Context,
						verb string,
						gvr schema.GroupVersionResource,
						subresource string,
						key client.ObjectKey,
					) error {
						assertFreightRejectAuthorization(
							t,
							testFreight,
							verb,
							gvr,
							subresource,
							key,
						)
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("freights").GroupResource(),
							testFreight.Name,
							errors.New("not authorized"),
						)
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "rejects Freight by name",
				body: bytes.NewBufferString(`{"reason":"  contains regression  "}`),
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithInfo(ctx, user.Info{IsAdmin: true})
				},
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = authorizeFreightRejectFn(t, testFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					err := c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight)
					require.NoError(t, err)
					require.NotNil(t, freight.Status.Rejected)
					require.NotNil(t, freight.Status.Rejected.RejectedAt)
					require.Equal(t, kargoapi.EventActorAdmin, freight.Status.Rejected.Actor)
					require.Equal(t, "contains regression", freight.Status.Rejected.Reason)
				},
			},
			{
				name: "rejects Freight by alias",
				url:  "/v1beta1/projects/" + testProject.Name + "/freight/" + testFreight.Alias + "/reject",
				body: bytes.NewBufferString(`{"reason":"bad build"}`),
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = authorizeFreightRejectFn(t, testFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					err := c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight)
					require.NoError(t, err)
					require.NotNil(t, freight.Status.Rejected)
					require.Equal(t, "bad build", freight.Status.Rejected.Reason)
				},
			},
			{
				name: "already rejected is idempotent",
				body: bytes.NewBufferString(`{"reason":"new reason"}`),
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				clientBuilder: fake.NewClientBuilder().
					WithObjects(
						testProject,
						rejectedTestFreight(testFreight, "original reason"),
					).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = authorizeFreightRejectFn(t, testFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					freight := &kargoapi.Freight{}
					err := c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight)
					require.NoError(t, err)
					require.NotNil(t, freight.Status.Rejected)
					require.Equal(t, "original reason", freight.Status.Rejected.Reason)
				},
			},
		},
	)
}

func Test_server_clearFreightRejection(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-freight",
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Alias: "fake-alias",
	}

	testRESTEndpoint(
		t,
		&config.ServerConfig{},
		http.MethodDelete,
		"/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name+"/reject",
		[]restTestCase{
			{
				name: "Project not found",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Freight not found",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "not authorized",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, rejectedTestFreight(testFreight, "bad build")).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = func(
						_ context.Context,
						verb string,
						gvr schema.GroupVersionResource,
						subresource string,
						key client.ObjectKey,
					) error {
						assertFreightRejectAuthorization(
							t,
							testFreight,
							verb,
							gvr,
							subresource,
							key,
						)
						return apierrors.NewForbidden(
							kargoapi.GroupVersion.WithResource("freights").GroupResource(),
							testFreight.Name,
							errors.New("not authorized"),
						)
					}
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "clears rejection by name",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, rejectedTestFreight(testFreight, "bad build")).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = authorizeFreightRejectFn(t, testFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					freight := &kargoapi.Freight{}
					err := c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight)
					require.NoError(t, err)
					require.Nil(t, freight.Status.Rejected)
					require.Contains(t, freight.Status.ApprovedFor, "fake-stage")
				},
			},
			{
				name: "clears rejection by alias",
				url:  "/v1beta1/projects/" + testProject.Name + "/freight/" + testFreight.Alias + "/reject",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, rejectedTestFreight(testFreight, "bad build")).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = authorizeFreightRejectFn(t, testFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					freight := &kargoapi.Freight{}
					err := c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight)
					require.NoError(t, err)
					require.Nil(t, freight.Status.Rejected)
				},
			},
			{
				name: "not rejected is idempotent",
				clientBuilder: fake.NewClientBuilder().
					WithObjects(testProject, testFreight).
					WithStatusSubresource(testFreight),
				serverSetup: func(t *testing.T, s *server) {
					s.authorizeFn = authorizeFreightRejectFn(t, testFreight)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					freight := &kargoapi.Freight{}
					err := c.Get(t.Context(), client.ObjectKeyFromObject(testFreight), freight)
					require.NoError(t, err)
					require.Nil(t, freight.Status.Rejected)
				},
			},
		},
	)
}

func authorizeFreightRejectFn(
	t *testing.T,
	freight *kargoapi.Freight,
) func(context.Context, string, schema.GroupVersionResource, string, client.ObjectKey) error {
	t.Helper()
	return func(
		_ context.Context,
		verb string,
		gvr schema.GroupVersionResource,
		subresource string,
		key client.ObjectKey,
	) error {
		assertFreightRejectAuthorization(t, freight, verb, gvr, subresource, key)
		return nil
	}
}

func assertFreightRejectAuthorization(
	t *testing.T,
	freight *kargoapi.Freight,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key client.ObjectKey,
) {
	t.Helper()
	require.Equal(t, freightRejectVerb, verb)
	require.Equal(t, kargoapi.GroupVersion.WithResource("freights"), gvr)
	require.Empty(t, subresource)
	require.Equal(t, client.ObjectKeyFromObject(freight), key)
}

func rejectedTestFreight(
	freight *kargoapi.Freight,
	reason string,
) *kargoapi.Freight {
	rejected := freight.DeepCopy()
	rejected.Status = kargoapi.FreightStatus{
		ApprovedFor: map[string]kargoapi.ApprovedStage{
			"fake-stage": {},
		},
		Rejected: &kargoapi.FreightRejection{
			RejectedAt: &metav1.Time{Time: time.Now()},
			Reason:     reason,
		},
	}
	return rejected
}
