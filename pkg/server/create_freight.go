package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/request"
)

// createFreightRequest is the request body for creating a piece of Freight.
type createFreightRequest struct {
	// Freight is the Freight resource to create. Its namespace is overridden with
	// the Project from the request path.
	Freight *kargoapi.Freight `json:"freight"`
	// Metadata is optional status metadata to record on the Freight. Each key is
	// set only if absent (fill-if-missing), so re-creating an already-existing
	// piece of Freight never overwrites metadata already recorded for it.
	Metadata map[string]apiextensionsv1.JSON `json:"metadata,omitempty"`
} // @name CreateFreightRequest

// @id CreateFreight
// @Summary Create a piece of Freight
// @Description Create a Freight resource, optionally recording status metadata on
// @Description it. The operation is idempotent: creating a piece of Freight that
// @Description already exists is not an error, and any metadata is applied
// @Description fill-if-missing so existing metadata is preserved.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body createFreightRequest true "Freight to create"
// @Success 201 {object} kargoapi.Freight "Freight was created"
// @Success 200 {object} kargoapi.Freight "Freight already existed"
// @Router /v1beta1/projects/{project}/freight [post]
func (s *server) createFreight(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	var req createFreightRequest
	if !request.BindJSON(c, &req) {
		return
	}
	if req.Freight == nil {
		_ = c.Error(libhttp.ErrorStr("freight is required", http.StatusBadRequest))
		return
	}
	if req.Freight.Name == "" {
		_ = c.Error(libhttp.ErrorStr("freight name is required", http.StatusBadRequest))
		return
	}

	freight := req.Freight.DeepCopy()
	// The Project from the path is authoritative; a Freight can only be created in
	// the Project the request is scoped to.
	freight.Namespace = project

	// Create the Freight. An already-existing piece of Freight is benign -- Freight
	// is content-addressed, so a re-create carries the same contents -- which keeps
	// a producer's reconcile idempotent. The response status distinguishes a
	// genuinely new piece of Freight (201) from a pre-existing one (200) so callers
	// can tell them apart.
	created := true
	if err := s.client.Create(ctx, freight); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			_ = c.Error(fmt.Errorf("create freight: %w", err))
			return
		}
		created = false
	}

	// Read back the created (or pre-existing) Freight so metadata is applied to its
	// current status and the response reflects the stored resource.
	current := &kargoapi.Freight{}
	key := client.ObjectKey{Namespace: project, Name: freight.Name}
	if err := s.client.Get(ctx, key, current); err != nil {
		_ = c.Error(fmt.Errorf("get freight: %w", err))
		return
	}

	if len(req.Metadata) > 0 {
		newStatus := *current.Status.DeepCopy()
		changed := false
		for k, v := range req.Metadata {
			// Fill-if-missing: never overwrite metadata already recorded.
			if _, exists := newStatus.Metadata[k]; exists {
				continue
			}
			if newStatus.Metadata == nil {
				newStatus.Metadata = map[string]apiextensionsv1.JSON{}
			}
			newStatus.Metadata[k] = v
			changed = true
		}
		if changed {
			// The authorizing client runs a SubjectAccessReview for patch on
			// freights/status, which is the authorization decision for this write.
			if err := s.patchFreightStatusFn(ctx, s.client, current, newStatus); err != nil {
				_ = c.Error(fmt.Errorf("patch freight status: %w", err))
				return
			}
			current.Status = newStatus
		}
	}

	if created {
		c.JSON(http.StatusCreated, current)
		return
	}
	c.JSON(http.StatusOK, current)
}
