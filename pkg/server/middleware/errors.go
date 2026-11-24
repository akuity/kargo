package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	libhttp "github.com/akuity/kargo/pkg/http"
)

// ErrorResponse is the body of every error response the REST API sends.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HandleErrors returns Gin middleware that answers the last error a handler
// or another middleware reported. It is the only thing that writes an error
// response, so that every one has the same shape.
//
// An error carrying a client-facing status (libhttp.HTTPError, or a
// Kubernetes status error) is answered with that status and its message. A
// request body over its limit is answered with a 413. Anything else is, by
// definition, an error nobody anticipated, and is answered as an internal
// server error whose body discloses nothing; the request logging middleware
// records the error itself.
func HandleErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err

		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(
				http.StatusRequestEntityTooLarge,
				ErrorResponse{Error: "request body too large"},
			)
			return
		}

		var httpErr *libhttp.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.Code() == http.StatusInternalServerError {
				respondInternalServerError(c)
				return
			}
			c.JSON(httpErr.Code(), ErrorResponse{Error: httpErr.Error()})
			return
		}

		var statusErr *apierrors.StatusError
		if errors.As(err, &statusErr) {
			c.JSON(int(statusErr.Status().Code), ErrorResponse{Error: err.Error()})
			return
		}

		respondInternalServerError(c)
	}
}

// respondInternalServerError responds with a 500 whose body discloses nothing
// about the underlying failure.
func respondInternalServerError(c *gin.Context) {
	c.JSON(
		http.StatusInternalServerError,
		ErrorResponse{Error: "internal server error"},
	)
}
