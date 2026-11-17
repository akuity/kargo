package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/request"
	"github.com/akuity/kargo/pkg/server/user"
)

// markFreightInStageRequest is the request body for recording that Freight is
// currently in use by a Stage.
type markFreightInStageRequest struct {
	// Verified indicates the Freight has been verified in the Stage, so that its
	// status should also record it as verified there.
	Verified bool `json:"verified"`
	// VerifiedAt is the time at which the Freight was verified in the Stage. It
	// is ignored unless Verified is true, and defaults to the time the request is
	// handled.
	VerifiedAt *metav1.Time `json:"verifiedAt,omitempty"`
} // @name MarkFreightInStageRequest

// @id MarkFreightInStage
// @Summary Mark Freight as currently in use by a Stage
// @Description Record that the named Freight is currently in use by the Stage,
// @Description and optionally that it has been verified there, updating the
// @Description Freight's status (currentlyIn / verifiedIn). This is used by the
// @Description Stage controller in place of writing Freight status directly.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param stage path string true "Stage name"
// @Param body body markFreightInStageRequest true "Status"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/status/stage/{stage} [post]
func (s *server) markFreightInStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightNameOrAlias := c.Param("freight-name-or-alias")
	stageName := c.Param("stage")

	var req markFreightInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}

	freight, err := s.getFreightByNameOrAlias(ctx, project, freightNameOrAlias)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// These are evaluated before the patch so each event is emitted only on the
	// first transition, not on every re-mark.
	newlyVerified := req.Verified && !freight.IsVerifiedIn(stageName)
	newlyCurrentlyIn := !freight.IsCurrentlyIn(stageName)

	now := time.Now()
	newStatus := *freight.Status.DeepCopy()
	newStatus.AddCurrentStage(stageName, now)
	if req.Verified {
		verifiedAt := now
		if req.VerifiedAt != nil {
			verifiedAt = req.VerifiedAt.Time
		}
		newStatus.AddVerifiedStage(stageName, verifiedAt)
	}

	// The authorizing client runs a SubjectAccessReview for patch on
	// freights/status, which is the authorization decision for this write.
	if err := s.patchFreightStatusFn(ctx, s.client, freight, newStatus); err != nil {
		_ = c.Error(fmt.Errorf("patch freight status: %w", err))
		return
	}

	if newlyVerified {
		s.emitFreightVerifiedInStage(ctx, freight, stageName)
	}

	if newlyCurrentlyIn {
		// Include verifiedIn information in the CurrentlyIn event
		_, verifiedInStage := newStatus.VerifiedIn[stageName]
		s.emitFreightCurrentlyInStage(ctx, freight, stageName, verifiedInStage)
	}

	c.Status(http.StatusOK)
}

// emitFreightVerifiedInStage sends a FreightVerifiedInStage event for the
// Freight. It is a no-op when no event sender is configured.
func (s *server) emitFreightVerifiedInStage(
	ctx context.Context,
	freight *kargoapi.Freight,
	stageName string,
) {
	if s.sender == nil {
		return
	}

	var actor string
	msg := fmt.Sprintf("Freight verified in Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewFreightVerifiedInStage(msg, actor, stageName, freight)
	if err == nil {
		err = s.sender.Send(ctx, evt)
	}
	if err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight verified in Stage event")
	}
}

// emitFreightCurrentlyInStage sends a FreightCurrentlyInStage event for the
// Freight. It is a no-op when no event sender is configured.
func (s *server) emitFreightCurrentlyInStage(
	ctx context.Context,
	freight *kargoapi.Freight,
	stageName string,
	verifiedInStage bool,
) {
	if s.sender == nil {
		return
	}

	var actor string
	msg := fmt.Sprintf("Freight currently in Stage %q", stageName)
	if u, ok := user.IdentityFromContext(ctx); ok {
		actor = u.Actor()
		msg += fmt.Sprintf(" by %q", actor)
	}

	evt, err := event.NewFreightCurrentlyInStage(msg, actor, stageName, verifiedInStage, freight)
	if err == nil {
		err = s.sender.Send(ctx, evt)
	}
	if err != nil {
		logging.LoggerFromContext(ctx).Error(err,
			"error sending Freight currently in Stage event")
	}
}

// @id UnmarkFreightInStage
// @Summary Mark Freight as no longer currently in use by a Stage
// @Description Record that the named Freight is no longer in use by the Stage,
// @Description updating the Freight's status (removing the Stage from
// @Description currentlyIn and finalizing its soak time). The Stage's entries in
// @Description verifiedIn and approvedFor are left intact. This is used by the
// @Description Stage controller in place of writing Freight status directly.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/status/stage/{stage} [patch]
func (s *server) unmarkFreightInStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightNameOrAlias := c.Param("freight-name-or-alias")
	stageName := c.Param("stage")

	freight, err := s.getFreightByNameOrAlias(ctx, project, freightNameOrAlias)
	if err != nil {
		_ = c.Error(err)
		return
	}

	newStatus := *freight.Status.DeepCopy()
	newStatus.RemoveCurrentStage(stageName)

	// The authorizing client runs a SubjectAccessReview for patch on
	// freights/status, which is the authorization decision for this write.
	if err := s.patchFreightStatusFn(ctx, s.client, freight, newStatus); err != nil {
		_ = c.Error(fmt.Errorf("patch freight status: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

// @id DeleteFreightStageStatus
// @Summary Remove all of a Stage's entries from a Freight's status
// @Description Remove the named Stage from every one of the Freight's status
// @Description maps -- currentlyIn, verifiedIn, and approvedFor. This is used by
// @Description the Stage controller to clean up Freight status when a Stage is
// @Description deleted, in place of writing Freight status directly.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/status/stage/{stage} [delete]
func (s *server) deleteFreightStageStatus(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightNameOrAlias := c.Param("freight-name-or-alias")
	stageName := c.Param("stage")

	freight, err := s.getFreightByNameOrAlias(ctx, project, freightNameOrAlias)
	if err != nil {
		_ = c.Error(err)
		return
	}

	newStatus := *freight.Status.DeepCopy()
	delete(newStatus.CurrentlyIn, stageName)
	delete(newStatus.VerifiedIn, stageName)
	delete(newStatus.ApprovedFor, stageName)

	// The authorizing client runs a SubjectAccessReview for patch on
	// freights/status, which is the authorization decision for this write.
	if err := s.patchFreightStatusFn(ctx, s.client, freight, newStatus); err != nil {
		_ = c.Error(fmt.Errorf("patch freight status: %w", err))
		return
	}

	c.Status(http.StatusOK)
}
