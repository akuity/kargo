package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id ListTargets
// @Summary List Targets
// @Description List Target resources from a project's namespace, optionally
// @Description narrowed to those a particular Stage governs or those matching a
// @Description label selector. Returns a TargetList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only Targets governed by this Stage, i.e. selected by its targets.selectors"
// @Param labelSelector query string false "Kubernetes label selector to filter Targets by, e.g. region=us,tier!=canary"
// @Produce json
// @Success 200 {object} kargoapi.TargetList "TargetList custom resource"
// @Router /v1beta1/projects/{project}/targets [get]
func (s *server) listTargets(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

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
		if stageSelectors, err = s.targetSelectorsForStage(c, project, stageName); err != nil {
			_ = c.Error(err)
			return
		}
	}

	if s.store == nil {
		_ = c.Error(errDatabaseNotConfigured)
		return
	}

	watchMode := c.Query("watch") == trueStr
	verb := "list"
	if watchMode {
		verb = "watch"
	}
	if err = s.authorizeStoreRead(ctx, verb, "targets", project, ""); err != nil {
		_ = c.Error(err)
		return
	}

	snapshot := func(ctx context.Context) ([]kargoapi.Target, error) {
		rows, listErr := s.store.ListTargets(ctx, project)
		if listErr != nil {
			return nil, fmt.Errorf("error listing Targets in Project %q: %w", project, listErr)
		}
		targets, convertErr := database.TargetsFromRows(rows, project)
		if convertErr != nil {
			return nil, convertErr
		}
		return filterTargets(targets, selector, stageSelectors), nil
	}

	if watchMode {
		servePolledWatch(c, s.storePollInterval, c.Query("resourceVersion"), polledWatch[kargoapi.Target]{
			snapshot: snapshot,
			name:     func(target kargoapi.Target) string { return target.Name },
			version:  func(target kargoapi.Target) string { return target.ResourceVersion },
		})
		return
	}

	items, err := snapshot(ctx)
	if err != nil {
		_ = c.Error(err)
		return
	}
	list := &kargoapi.TargetList{Items: items}
	versions := make([]string, len(items))
	for i, item := range items {
		versions[i] = item.ResourceVersion
	}
	list.ResourceVersion = maxResourceVersion(versions...)

	c.JSON(http.StatusOK, list)
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

// targetSelectorsForStage loads the named Stage and parses its target
// selectors. A classic Stage yields an empty, non-nil slice so that callers can
// distinguish "filter by a Stage that governs nothing" from "no Stage filter".
func (s *server) targetSelectorsForStage(
	c *gin.Context,
	project string,
	stageName string,
) ([]labels.Selector, error) {
	stage := &kargoapi.Stage{}
	if err := s.client.Get(
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
