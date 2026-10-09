package targets

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id ListTargets
// @Summary List Targets
// @Description List a project's Targets, optionally narrowed to those a
// @Description particular Stage governs or those matching a label selector.
// @Description Returns a TargetList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only Targets governed by this Stage, i.e. selected by its targets.selectors"
// @Param labelSelector query string false "Kubernetes label selector to filter Targets by, e.g. region=us,tier!=canary"
// @Produce json
// @Success 200 {object} kargoapi.TargetList "TargetList resource"
// @Router /v1beta1/projects/{project}/targets [get]
func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	// Parse the label selector up front so that a malformed one is a 400 rather
	// than a failed list.
	selector, err := parseLabelSelectorQuery(c.Query("labelSelector"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	// A Stage filter narrows the list to the Targets the Stage governs. Its
	// selectors are read once here; a classic Stage governs none.
	var stageSelectors []labels.Selector
	if stageName := c.Query("stage"); stageName != "" {
		if stageSelectors, err = h.stageSelectors(c, project, stageName); err != nil {
			_ = c.Error(err)
			return
		}
	}

	rows, err := h.store.List(ctx, project)
	if err != nil {
		_ = c.Error(err)
		return
	}
	targets := make([]kargoapi.Target, len(rows))
	for i, row := range rows {
		target, convErr := targetFromRow(row, project)
		if convErr != nil {
			_ = c.Error(convErr)
			return
		}
		targets[i] = *target
	}

	c.JSON(http.StatusOK, &kargoapi.TargetList{
		Items: filterTargets(targets, selector, stageSelectors),
	})
}

// parseLabelSelectorQuery parses the labelSelector query parameter. An empty
// value yields a nil selector, meaning no label filtering at all.
func parseLabelSelectorQuery(raw string) (labels.Selector, error) {
	if raw == "" {
		return nil, nil
	}
	selector, err := labels.Parse(raw)
	if err != nil {
		return nil, libhttp.Error(
			fmt.Errorf("invalid labelSelector %q: %w", raw, err),
			http.StatusBadRequest,
		)
	}
	return selector, nil
}

// stageSelectors loads the named Stage and parses its target selectors. A
// classic Stage yields an empty, non-nil slice so that callers can distinguish
// "filter by a Stage that governs nothing" from "no Stage filter".
func (h *Handler) stageSelectors(
	c *gin.Context,
	project string,
	stageName string,
) ([]labels.Selector, error) {
	stage := &kargoapi.Stage{}
	if err := h.kube.Get(
		c.Request.Context(),
		client.ObjectKey{Namespace: project, Name: stageName},
		stage,
	); err != nil {
		return nil, err
	}
	selectors, err := api.TargetSelectorsForStage(stage)
	if err != nil {
		return nil, libhttp.Error(err, http.StatusBadRequest)
	}
	if selectors == nil {
		selectors = []labels.Selector{}
	}
	return selectors, nil
}

// filterTargets returns the Targets matching the label selector, if any, and
// any of the Stage selectors, if there are any. A nil selector applies no
// label filter; nil Stage selectors apply no Stage filter, whereas an empty
// list governs nothing. The Stage filter is applied in-process because a Stage
// may have several selectors whose union no single label selector can
// express. The result is never nil.
func filterTargets(
	targets []kargoapi.Target,
	selector labels.Selector,
	stageSelectors []labels.Selector,
) []kargoapi.Target {
	filtered := make([]kargoapi.Target, 0, len(targets))
	for _, target := range targets {
		if selector != nil && !selector.Matches(labels.Set(target.Labels)) {
			continue
		}
		if stageSelectors != nil && !api.AnySelectorMatches(stageSelectors, target.Labels) {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}
