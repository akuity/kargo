package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	libhttp "github.com/akuity/kargo/pkg/http"
)

func TestHandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "no error",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "request body too large",
			err:            &http.MaxBytesError{Limit: 1024},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectedBody:   `{"error":"request body too large"}`,
		},
		{
			name:           "error carrying a client-facing status code",
			err:            libhttp.ErrorStr("invalid token", http.StatusUnauthorized),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid token"}`,
		},
		{
			// The body must contain nothing but the obfuscated message. Writing
			// the underlying message as well would disclose internal detail.
			name: "error carrying a 500",
			err: libhttp.Error(
				errors.New("create TokenReview: forbidden"),
				http.StatusInternalServerError,
			),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
		{
			name: "Kubernetes status error",
			err: apierrors.NewNotFound(
				schema.GroupResource{Group: "kargo.akuity.io", Resource: "stages"},
				"fake-stage",
			),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"stages.kargo.akuity.io \"fake-stage\" not found"}`,
		},
		{
			// An unrecognized error must not be mistaken for success.
			name:           "error of no recognized type",
			err:            fmt.Errorf("error signing ID token: %w", errors.New("no key")),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal server error"}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			s := &server{}
			router := gin.New()
			router.Use(s.handleError)
			router.GET("/", func(c *gin.Context) {
				if testCase.err != nil {
					_ = c.Error(testCase.err)
					return
				}
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, testCase.expectedStatus, w.Code)
			require.Equal(t, testCase.expectedBody, w.Body.String())
		})
	}
}
