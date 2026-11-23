// nolint:lll
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/freightapi"
	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/server/request"
	"github.com/akuity/kargo/pkg/server/user"
)

// markFreightsCurrentlyInStageRequest is the request body for recording that
// a set of Freights were set as current for stage.
type markFreightsCurrentlyInStageRequest struct {
	// FreightNames is a list of names of freights to be marked as current in stage
	FreightNames []string
} // @name MarkFreightsCurrentlyInStageRequest

// @id MarkFreightsCurrentlyInStage
// @Summary Mark a set of Freights as currently in use by a Stage
// @Description Record that the named Freights are currently in use by the Stage,
// @Description adding the Stage to the Freight's currentlyIn status. Any other
// @Description Freights will have this Stage removed from their currentlyIn.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body markFreightsCurrentlyInStageRequest true "CurrentlyIn"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/stage-freight/{stage}/current [post]
func (s *server) markFreightsCurrentlyInStage(c *gin.Context) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	project := c.Param("project")
	stageName := c.Param("stage")

	var req markFreightsCurrentlyInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}

	freights := []*kargoapi.Freight{}
	freightNames := map[string]struct{}{}
	for _, freightName := range req.FreightNames {
		freightNames[freightName] = struct{}{}
		freight, err := s.getFreightByName(ctx, project, freightName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Error(err, "freight is current but not found")
				continue
			}
			_ = c.Error(err)
			return
		}
		freights = append(freights, freight)
	}

	// Unmark other Freight before marking the new one.
	// Only one freight should be Current per origin at a time.
	// Caller ensures that FreightNames are one per origin.
	current, err := freightapi.ListFreightByCurrentStage(ctx, s.client, project, stageName)
	if err != nil {
		_ = c.Error(err)
		return
	}
	for i := range current {
		other := &current[i]
		if _, ok := freightNames[other.Name]; ok {
			// Freight will be set as current, skipping clean
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

	for _, freight := range freights {
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
	}

	c.Status(http.StatusOK)
}

// markFreightsVerifiedInStageRequest is the request body for recording that
// Freight has been verified in a Stage.
type markFreightsVerifiedInStageRequest struct {
	// FreightNames is a list of names of freights to be marked as verified in stage
	FreightNames []string
	// VerifiedAt is the time at which the Freight was verified in the Stage. It
	// defaults to the time the request is handled.
	VerifiedAt *metav1.Time `json:"verifiedAt,omitempty"`
} // @name MarkFreightsVerifiedInStageRequest

// @id MarkFreightsVerifiedInStage
// @Summary Mark a set of Freights as verified in a Stage, adding to verifiedIn
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param body body markFreightsVerifiedInStageRequest true "Verification"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/stage-freight/{stage}/verified [post]
func (s *server) markFreightsVerifiedInStage(c *gin.Context) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	project := c.Param("project")
	stageName := c.Param("stage")

	var req markFreightsVerifiedInStageRequest
	if !request.BindJSON(c, &req) {
		return
	}

	for _, freightName := range req.FreightNames {

		freight, err := s.getFreightByName(ctx, project, freightName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Error(err, "freight is verified but not found")
				continue
			}
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

// @id ClearStageFreights
// @Summary Remove all of a Stage's entries from any Freight's status
// @Description Remove the named Stage from all Freights which reference it. This is used by
// @Description the Stage controller to clean up Freight status when a Stage is deleted.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/internal/stage-freight/{stage} [delete]
func (s *server) clearStageFreights(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stageName := c.Param("stage")

	freights, err := freightapi.ListFreightReferencingStage(ctx, s.client, project, stageName)
	if err != nil {
		_ = c.Error(err)
		return
	}

	for _, freight := range freights {
		newStatus := *freight.Status.DeepCopy()
		delete(newStatus.CurrentlyIn, stageName)
		delete(newStatus.VerifiedIn, stageName)
		delete(newStatus.ApprovedFor, stageName)

		// The authorizing client runs a SubjectAccessReview for patch on
		// freights/status, which is the authorization decision for this write.
		if err = s.patchFreightStatusFn(ctx, s.client, &freight, newStatus); err != nil {
			_ = c.Error(fmt.Errorf("patch freight status: %w", err))
			return
		}
	}

	c.Status(http.StatusOK)
}
