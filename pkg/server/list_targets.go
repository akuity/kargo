package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
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

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchTargets(c, project, selector, stageSelectors, c.Query("resourceVersion"))
		return
	}

	listOpts := []client.ListOption{client.InNamespace(project)}
	if selector != nil {
		listOpts = append(listOpts, client.MatchingLabelsSelector{Selector: selector})
	}

	list := &kargoapi.TargetList{}
	if err = s.listForWatchSeed(ctx, "targets", list, listOpts...); err != nil {
		_ = c.Error(err)
		return
	}
	if stageSelectors != nil {
		list.Items = filterTargetsBySelectors(list.Items, stageSelectors)
	}

	list.ResourceVersion = normalizeListResourceVersion(list.ResourceVersion)

	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.Target) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

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

// filterTargetsBySelectors returns the Targets matching any of the provided
// selectors. Filtering happens in-process because a Stage may have several
// selectors whose union no single label selector can express.
func filterTargetsBySelectors(
	targets []kargoapi.Target,
	selectors []labels.Selector,
) []kargoapi.Target {
	filtered := make([]kargoapi.Target, 0, len(targets))
	for _, target := range targets {
		if api.AnySelectorMatches(selectors, target.Labels) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

// watchTargets streams Target changes through the REST SSE endpoint. The label
// selector is applied by the API server; the Stage filter, being a union of
// selectors, is applied here to each event.
func (s *server) watchTargets(
	c *gin.Context,
	project string,
	selector labels.Selector,
	stageSelectors []labels.Selector,
	resourceVersion string,
) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	watchOpts := buildWatchListOptions(project, resourceVersion)
	if selector != nil {
		watchOpts = append(watchOpts, client.MatchingLabelsSelector{Selector: selector})
	}

	w, err := s.client.Watch(ctx, &kargoapi.TargetList{}, watchOpts...)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch targets: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	SetSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !WriteSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}
			if watchErr := ErrorFromWatchEvent(e); watchErr != nil {
				SendSSEWatchError(c, watchErr)
				return
			}

			target, ok := ConvertWatchEventObject(c, e, (*kargoapi.Target)(nil))
			if !ok {
				continue
			}

			eventType := e.Type
			if stageSelectors != nil {
				var send bool
				eventType, send = FilteredWatchEventType(
					e.Type,
					api.AnySelectorMatches(stageSelectors, target.Labels),
				)
				if !send {
					continue
				}
			}

			if !SendSSEWatchEvent(c, eventType, target) {
				return
			}
		}
	}
}
