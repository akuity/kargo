package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/server/user"
)

const (
	freightRejectVerb = "reject"
	// maxFreightRejectionReasonLength mirrors the MaxLength validation marker on
	// FreightRejection.Reason in api/v1alpha1/freight_types.go.
	maxFreightRejectionReasonLength = 1024
)

type rejectFreightRequest struct {
	Reason string `json:"reason,omitempty"`
} // @name RejectFreightRequest

// @id RejectFreight
// @Summary Reject Freight
// @Description Mark a Freight resource as rejected, preventing future promotion.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param body body rejectFreightRequest true "Reject Freight request"
// @Success 200 "Success"
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/reject [post]
func (s *server) rejectFreight(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	var req rejectFreightRequest
	if !bindJSONOrError(c, &req) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if len(req.Reason) > maxFreightRejectionReasonLength {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"reason cannot be longer than %d characters",
				maxFreightRejectionReasonLength,
			),
			http.StatusBadRequest,
		))
		return
	}

	freight := s.getFreightByNameOrAliasForGin(c, project, nameOrAlias)
	if freight == nil {
		return
	}

	if err := s.authorizeFreightRejection(ctx, freight); err != nil {
		_ = c.Error(err)
		return
	}

	if freight.IsRejected() {
		c.Status(http.StatusOK)
		return
	}

	newStatus := *freight.Status.DeepCopy()
	newStatus.Reject(freightRejectionActor(c), req.Reason, time.Now())
	if err := kubeclient.PatchStatus(
		ctx,
		// The caller was authorized for the custom "reject" verb above. The
		// internal client performs the mechanical status patch so rejection
		// permission does not also require patch access to freights/status.
		s.client.InternalClient(),
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		_ = c.Error(fmt.Errorf("patch Freight status: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

// @id ClearFreightRejection
// @Summary Clear Freight rejection
// @Description Clear a Freight resource's rejected status.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Success 204 "Success"
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/reject [delete]
func (s *server) clearFreightRejection(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	freight := s.getFreightByNameOrAliasForGin(c, project, nameOrAlias)
	if freight == nil {
		return
	}

	if err := s.authorizeFreightRejection(ctx, freight); err != nil {
		_ = c.Error(err)
		return
	}

	if !freight.IsRejected() {
		c.Status(http.StatusNoContent)
		return
	}

	newStatus := *freight.Status.DeepCopy()
	newStatus.ClearRejected()
	if err := kubeclient.PatchStatus(
		ctx,
		// The caller was authorized for the custom "reject" verb above. The
		// internal client performs the mechanical status patch so rejection
		// permission does not also require patch access to freights/status.
		s.client.InternalClient(),
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		_ = c.Error(fmt.Errorf("patch Freight status: %w", err))
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *server) authorizeFreightRejection(
	ctx context.Context,
	freight *kargoapi.Freight,
) error {
	return s.authorizeFn(
		ctx,
		freightRejectVerb,
		kargoapi.GroupVersion.WithResource("freights"),
		"",
		client.ObjectKeyFromObject(freight),
	)
}

func freightRejectionActor(c *gin.Context) string {
	if u, ok := user.InfoFromContext(c.Request.Context()); ok {
		return api.FormatEventUserActor(u)
	}
	return ""
}
