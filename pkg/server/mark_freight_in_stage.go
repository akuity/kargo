package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/freightapi"
	"github.com/akuity/kargo/pkg/event"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/request"
	"github.com/akuity/kargo/pkg/server/user"
)

// markFreightCurrentlyInStageRequest is the request body for recording which
// Freight is currently in use by a Stage.
type markFreightCurrentlyInStageRequest struct {
	// Freight is the name of the Freight now currently in use by the Stage.
	Freight string `json:"freight"`
	// Origin is the origin of the Freight. It must match the Freight's own
	// origin. Any other Freight from this origin that is recorded as currently
	// in the Stage is unmarked.
	Origin kargoapi.FreightOrigin `json:"origin"`
} // @name MarkFreightCurrentlyInStageRequest

// @id MarkFreightCurrentlyInStage
// @Summary Mark Freight as currently in use by a Stage
// @Description Record that the named Freight is currently in use by the Stage,
// @Description adding the Stage to the Freight's currentlyIn status. The
// @Description request's origin must match the Freight's origin. Any other
// @Description Freight from that origin that is recorded as currently in the
// @Description Stage has the Stage removed from its currentlyIn.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body markFreightCurrentlyInStageRequest true "Freight"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/freight [post]
func (s *server) markFreightCurrentlyInStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stageName := c.Param("stage")

	var req markFreightCurrentlyInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}
	if req.Freight == "" {
		_ = c.Error(libhttp.ErrorStr("freight is required", http.StatusBadRequest))
		return
	}
	if req.Origin.Kind == "" || req.Origin.Name == "" {
		_ = c.Error(libhttp.ErrorStr("origin kind and name are required", http.StatusBadRequest))
		return
	}

	freight, err := s.getFreightByName(ctx, project, req.Freight)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if !freight.Origin.Equals(&req.Origin) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"freight %q has origin %q, not %q",
				freight.Name, freight.Origin.String(), req.Origin.String(),
			),
			http.StatusBadRequest,
		))
		return
	}

	// Unmark other Freight from the same origin before marking the new one.
	// Only one freight should be Current at a time.
	current, err := freightapi.ListFreightByCurrentStage(ctx, s.client, project, stageName)
	if err != nil {
		_ = c.Error(err)
		return
	}
	for i := range current {
		other := &current[i]
		if other.Name == freight.Name || !other.Origin.Equals(&req.Origin) {
			continue
		}
		newStatus := *other.Status.DeepCopy()
		newStatus.RemoveCurrentStage(stageName)
		// The authorizing client runs a SubjectAccessReview for patch on
		// freights/status, which is the authorization decision for this write.
		if err = s.patchFreightStatusFn(ctx, s.client, other, newStatus); err != nil {
			_ = c.Error(fmt.Errorf("patch status of freight %q: %w", other.Name, err))
			return
		}
	}

	// Evaluated before the patch so the event is emitted only on the first
	// transition, not on every re-mark.
	newlyCurrentlyIn := !freight.IsCurrentlyIn(stageName)

	newStatus := *freight.Status.DeepCopy()
	newStatus.AddCurrentStage(stageName, time.Now())
	if err = s.patchFreightStatusFn(ctx, s.client, freight, newStatus); err != nil {
		_ = c.Error(fmt.Errorf("patch status of freight %q: %w", freight.Name, err))
		return
	}

	if newlyCurrentlyIn {
		// Include verifiedIn information in the CurrentlyIn event
		s.emitFreightCurrentlyInStage(ctx, freight, stageName, freight.IsVerifiedIn(stageName))
	}

	c.Status(http.StatusOK)
}

// markFreightVerifiedInStageRequest is the request body for recording that
// Freight has been verified in a Stage.
type markFreightVerifiedInStageRequest struct {
	// VerifiedAt is the time at which the Freight was verified in the Stage. It
	// defaults to the time the request is handled.
	VerifiedAt *metav1.Time `json:"verifiedAt,omitempty"`
} // @name MarkFreightVerifiedInStageRequest

// @id MarkFreightVerifiedInStage
// @Summary Mark Freight as verified in a Stage
// @Description Record that the named Freight has been verified in the Stage,
// @Description adding the Stage to the Freight's verifiedIn status.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param freight path string true "Freight name"
// @Param stage path string true "Stage name"
// @Param body body markFreightVerifiedInStageRequest true "Verification"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/freight/{freight}/status/stage/{stage} [post]
func (s *server) markFreightVerifiedInStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightName := c.Param("freight")
	stageName := c.Param("stage")

	var req markFreightVerifiedInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}

	freight, err := s.getFreightByName(ctx, project, freightName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Evaluated before the patch so the event is emitted only on the first
	// transition, not on every re-mark.
	newlyVerified := !freight.IsVerifiedIn(stageName)

	verifiedAt := time.Now()
	if req.VerifiedAt != nil {
		verifiedAt = req.VerifiedAt.Time
	}
	newStatus := *freight.Status.DeepCopy()
	newStatus.AddVerifiedStage(stageName, verifiedAt)

	// The authorizing client runs a SubjectAccessReview for patch on
	// freights/status, which is the authorization decision for this write.
	if err = s.patchFreightStatusFn(ctx, s.client, freight, newStatus); err != nil {
		_ = c.Error(fmt.Errorf("patch freight status: %w", err))
		return
	}

	if newlyVerified {
		s.emitFreightVerifiedInStage(ctx, freight, stageName)
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
// @Param freight path string true "Freight name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/freight/{freight}/status/stage/{stage} [delete]
func (s *server) deleteFreightStageStatus(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	freightName := c.Param("freight")
	stageName := c.Param("stage")

	freight, err := s.getFreightByName(ctx, project, freightName)
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
	if err = s.patchFreightStatusFn(ctx, s.client, freight, newStatus); err != nil {
		_ = c.Error(fmt.Errorf("patch freight status: %w", err))
		return
	}

	c.Status(http.StatusOK)
}
