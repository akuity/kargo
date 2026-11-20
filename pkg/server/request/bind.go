package request

import (
	"net/http"

	"github.com/gin-gonic/gin"

	libhttp "github.com/akuity/kargo/pkg/http"
)

// BindJSON reads the request body as JSON into target. It reports false
// after recording a 400 on the context, in which case the handler must
// return.
func BindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return false
	}
	return true
}
