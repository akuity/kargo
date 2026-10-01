package rest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	os.Exit(m.Run())
}

var testGVR = schema.GroupVersionResource{
	Group:    "kargo.akuity.io",
	Version:  "v1alpha1",
	Resource: "widgets",
}

type call struct {
	verb string
	gvr  schema.GroupVersionResource
	key  client.ObjectKey
}

func TestGuard_Require(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		authorizer kubernetes.Authorizer
		url        string
		assert     func(*testing.T, *httptest.ResponseRecorder, []call)
	}{
		{
			name: "no authorizer fails closed",
			url:  "/projects/demo/widgets/x",
			assert: func(t *testing.T, w *httptest.ResponseRecorder, _ []call) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "refusal keeps its status",
			authorizer: Func(func(
				context.Context, string, schema.GroupVersionResource, string, client.ObjectKey,
			) error {
				return apierrors.NewForbidden(testGVR.GroupResource(), "x", errors.New("no"))
			}),
			url: "/projects/demo/widgets/x",
			assert: func(t *testing.T, w *httptest.ResponseRecorder, _ []call) {
				require.Equal(t, http.StatusForbidden, w.Code)
				require.Contains(t, w.Body.String(), "forbidden")
			},
		},
		{
			name: "a failure to decide is an unexpected failure",
			authorizer: Func(func(
				context.Context, string, schema.GroupVersionResource, string, client.ObjectKey,
			) error {
				return errors.New("review failed")
			}),
			url: "/projects/demo/widgets/x",
			assert: func(t *testing.T, w *httptest.ResponseRecorder, _ []call) {
				require.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "an allowed request reaches the handler with the key from the route",
			url:  "/projects/demo/widgets/x",
			assert: func(t *testing.T, w *httptest.ResponseRecorder, calls []call) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, []call{{
					verb: "get",
					gvr:  testGVR,
					key:  client.ObjectKey{Namespace: "demo", Name: "x"},
				}}, calls)
			},
		},
		{
			name: "a collection has no name",
			url:  "/projects/demo/widgets",
			assert: func(t *testing.T, w *httptest.ResponseRecorder, calls []call) {
				require.Equal(t, http.StatusOK, w.Code)
				require.Equal(t, client.ObjectKey{Namespace: "demo"}, calls[0].key)
				require.Equal(t, "list", calls[0].verb)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var calls []call
			authorizer := testCase.authorizer
			if authorizer == nil && testCase.name != "no authorizer fails closed" {
				authorizer = Func(func(
					_ context.Context,
					verb string,
					gvr schema.GroupVersionResource,
					_ string,
					key client.ObjectKey,
				) error {
					calls = append(calls, call{verb, gvr, key})
					return nil
				})
			}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Next()
				if len(c.Errors) > 0 {
					libhttp.WriteErrorJSON(c.Writer, c.Errors.Last().Err)
				}
			})
			ok := func(c *gin.Context) { c.Status(http.StatusOK) }
			guard := NewGuard(authorizer, testGVR).Namespace("project").Name("name")
			group := router.Group("/projects/:project/widgets")
			group.GET("", guard.Require("list"), ok)
			group.GET("/:name", guard.Require("get"), ok)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, testCase.url, nil))
			testCase.assert(t, w, calls)
		})
	}
}
