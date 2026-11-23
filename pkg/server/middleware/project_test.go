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
		assert  func(*testing.T, []*gin.Error, *kargoapi.Project)
	}{
		{
			name: "Project does not exist",
			assert: func(t *testing.T, errs []*gin.Error, seen *kargoapi.Project) {
				require.Nil(t, seen)
				require.Len(t, errs, 1)
				require.True(t, apierrors.IsNotFound(errs[0].Err))
			},
		},
		{
			name: "Project exists",
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name: "fake-project",
					UID:  "fake-uid",
				}},
			},
			assert: func(t *testing.T, errs []*gin.Error, seen *kargoapi.Project) {
				require.Empty(t, errs)
				// The handler sees the Project the middleware fetched.
				require.NotNil(t, seen)
				require.Equal(t, "fake-project", seen.Name)
				require.Equal(t, "fake-uid", string(seen.UID))
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
			var seen *kargoapi.Project
			router.GET(
				"/projects/:project",
				RequireProject(cl),
				func(c *gin.Context) {
					project, ok := ProjectFromContext(c.Request.Context())
					require.True(t, ok)
					seen = project
					c.Status(http.StatusOK)
				},
			)
			router.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodGet, "/projects/fake-project", nil),
			)
			testCase.assert(t, reported, seen)
		})
	}
}

func TestProjectFromContext(t *testing.T) {
	t.Parallel()
	_, ok := ProjectFromContext(t.Context())
	require.False(t, ok)

	project := &kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "fake-project"}}
	got, ok := ProjectFromContext(contextWithProject(t.Context(), project))
	require.True(t, ok)
	require.Same(t, project, got)
}
