package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestRequireProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))
	testCases := []struct {
		name    string
		objects []client.Object
		assert  func(*testing.T, []*gin.Error, bool)
	}{
		{
			name: "Project does not exist",
			assert: func(t *testing.T, errs []*gin.Error, reached bool) {
				require.False(t, reached)
				require.Len(t, errs, 1)
				require.True(t, apierrors.IsNotFound(errs[0].Err))
			},
		},
		{
			name: "Project exists",
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "fake-project"}},
			},
			assert: func(t *testing.T, errs []*gin.Error, reached bool) {
				require.True(t, reached)
				require.Empty(t, errs)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testCase.objects...).
				Build()
			router := gin.New()
			var reported []*gin.Error
			router.Use(func(c *gin.Context) {
				c.Next()
				reported = c.Errors
			})
			reached := false
			router.GET(
				"/projects/:project",
				RequireProject(cl),
				func(c *gin.Context) {
					reached = true
					c.Status(http.StatusOK)
				},
			)
			router.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodGet, "/projects/fake-project", nil),
			)
			testCase.assert(t, reported, reached)
		})
	}
}
