package external

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/controller/git/commit"
	"github.com/akuity/kargo/pkg/helm/chart"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/image"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

type targetResult struct {
	Kind           kargoapi.GenericWebhookTargetKind `json:"kind"`
	ListError      error                             `json:"listError,omitempty"`
	RefreshResults []refreshResult                   `json:"refreshResults,omitempty"`
}

type refreshResult struct {
	Success string `json:"success,omitempty"`
	Failure string `json:"failure,omitempty"`
}

func handleRefreshAction(
	ctx context.Context,
	c client.Client,
	project string,
	actionEnv map[string]any,
	targets []kargoapi.GenericWebhookTarget,
) []targetResult {
	logger := logging.LoggerFromContext(ctx)
	targetResults := make([]targetResult, len(targets))
	for i, target := range targets {
		targetResults[i] = targetResult{Kind: target.Kind}
		switch target.Kind {
		case kargoapi.GenericWebhookTargetKindWarehouse:
			listOpts, err := buildListOptionsForTarget(project, target, actionEnv)
			if err != nil {
				logger.Error(err, "failed to build list options for warehouse target")
				targetResults[i].ListError = fmt.Errorf("failed to build list options for warehouse target: %w", err)
				continue
			}

			var whList kargoapi.WarehouseList
			if err := c.List(ctx, &whList, listOpts...); err != nil {
				logger.Error(err, "error listing warehouse targets")
				targetResults[i].ListError = fmt.Errorf("error listing warehouse targets: %w", err)
				continue
			}

			logger.Info("found Warehouses to refresh", "count", len(whList.Items))

			targetResults[i].RefreshResults = make([]refreshResult, len(whList.Items))
			for j, wh := range whList.Items {
				whKey := client.ObjectKeyFromObject(&wh)
				whLogger := logger.WithValues(
					"namespace", whKey.Namespace,
					"name", whKey.Name,
				)
				if _, err := api.RefreshWarehouse(ctx, c, whKey); err != nil {
					whLogger.Error(err, "error refreshing")
					targetResults[i].RefreshResults[j].Failure = whKey.String()
				} else {
					whLogger.Debug("successfully refreshed Warehouse")
					targetResults[i].RefreshResults[j].Success = whKey.String()
				}
			}
		default:
			targetResults[i].ListError = fmt.Errorf("skipped listing of unsupported target type: %q", target.Kind)
		}
	}
	return targetResults
}

// refreshWarehouses refreshes all Warehouses in the given namespace that are
// subscribed to any of the given repository URLs. If the namespace is empty,
// all Warehouses in the cluster subscribed to the given repository URLs are
// refreshed. Note: Callers are responsible for normalizing the provided
// repository URLs.
func refreshWarehouses(
	ctx context.Context,
	w http.ResponseWriter,
	c client.Client,
	project string,
	repoURLs []string,
	qualifiers ...string,
) {
	logger := logging.LoggerFromContext(ctx)

	// De-dupe repository URLs
	slices.Sort(repoURLs)
	repoURLs = slices.Compact(repoURLs)
	// If there had been any empty strings in the slice, after sorting and
	// compacting, at most the zero element will be empty. If it is, remove it.
	if len(repoURLs) > 0 && repoURLs[0] == "" {
		repoURLs = repoURLs[1:]
	}

	// De-dupe qualifiers
	slices.Sort(qualifiers)
	qualifiers = slices.Compact(qualifiers)
	// If there had been any empty strings in the slice, after sorting and
	// compacting, at most the zero element will be empty. If it is, remove it.
	if len(qualifiers) > 0 && qualifiers[0] == "" {
		qualifiers = qualifiers[1:]
	}

	// The distinct set of all Warehouses that should be refreshed
	toRefresh := map[client.ObjectKey]*kargoapi.Warehouse{}

	for _, repoURL := range repoURLs {
		repoLogger := logger.WithValues("repositoryURL", repoURL)

		listOpts := make([]client.ListOption, 1, 2)
		listOpts[0] = client.MatchingFields{
			indexer.WarehousesBySubscribedURLsField: repoURL,
		}
		if project != "" {
			listOpts = append(listOpts, client.InNamespace(project))
		}

		ws := kargoapi.WarehouseList{}
		if err := c.List(ctx, &ws, listOpts...); err != nil {
			repoLogger.Error(err, "error listing subscribed Warehouses")
			xhttp.WriteErrorJSON(w, err)
			return
		}

		for _, wh := range ws.Items {
			whKey := client.ObjectKeyFromObject(&wh)
			if _, alreadyRefreshing := toRefresh[whKey]; alreadyRefreshing {
				continue
			}
			if len(qualifiers) > 0 {
				shouldRefresh, err := shouldRefresh(wh, repoURL, qualifiers...)
				if err != nil {
					logger.Error(
						err,
						"failed to evaluate if warehouse needs refresh",
						"warehouse", wh.Name,
					)
					xhttp.WriteErrorJSON(w, err)
					return
				}
				if shouldRefresh {
					toRefresh[whKey] = &wh
				}
			} else {
				toRefresh[whKey] = &wh
			}
		}
	}

	logger.Debug("found Warehouses to refresh", "count", len(toRefresh))

	var failures int
	for whKey := range toRefresh {
		whLogger := logger.WithValues(
			"namespace", whKey.Namespace,
			"name", whKey.Name,
		)
		if _, err := api.RefreshWarehouse(ctx, c, whKey); err != nil {
			whLogger.Error(err, "error refreshing Warehouse")
			failures++
		} else {
			whLogger.Debug("refreshed Warehouse")
		}
	}

	if failures > 0 {
		xhttp.WriteResponseJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": fmt.Sprintf("failed to refresh %d of %d warehouses",
					failures,
					len(toRefresh),
				),
			},
		)
		return
	}
	xhttp.WriteResponseJSON(
		w,
		http.StatusOK,
		map[string]string{
			"msg": fmt.Sprintf("refreshed %d warehouse(s)", len(toRefresh)),
		},
	)
}

func shouldRefresh(wh kargoapi.Warehouse, repoURL string, qualifiers ...string) (bool, error) {
	var shouldRefresh bool
	for _, s := range wh.Spec.Subscriptions {
		switch {
		case s.Git != nil && urls.NormalizeGit(s.Git.RepoURL) == repoURL:
			selector, err := commit.NewSelector(*s.Git, nil)
			if err != nil {
				return false, fmt.Errorf("error creating commit selector for Git subscription %q: %w",
					s.Git.RepoURL, err,
				)
			}
			shouldRefresh = slices.ContainsFunc(qualifiers, selector.MatchesRef)
		case s.Image != nil && urls.NormalizeImage(s.Image.RepoURL) == repoURL:
			selector, err := image.NewSelector(*s.Image, nil)
			if err != nil {
				return false, fmt.Errorf("error creating image selector for Image subscription %q: %w",
					s.Image.RepoURL, err,
				)
			}
			shouldRefresh = slices.ContainsFunc(qualifiers, selector.MatchesTag)
		case s.Chart != nil && urls.NormalizeChart(s.Chart.RepoURL) == repoURL:
			selector, err := chart.NewSelector(*s.Chart, nil)
			if err != nil {
				return false, fmt.Errorf("error creating chart selector for Chart subscription %q: %w",
					s.Chart.RepoURL, err,
				)
			}
			shouldRefresh = slices.ContainsFunc(qualifiers, selector.MatchesVersion)
		}
		if shouldRefresh {
			return true, nil
		}
	}
	return false, nil
}
