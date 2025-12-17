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

func refreshObjects(
	ctx context.Context,
	c client.Client,
	objList []client.Object,
) ([]selectedTarget, string, string) {
	logger := logging.LoggerFromContext(ctx)
	selectedTargets := make([]selectedTarget, len(objList))
	var successCount, failureCount int
	for i, obj := range objList {
		objKey := client.ObjectKeyFromObject(obj)
		objLogger := logger.WithValues(
			"namespace", objKey.Namespace,
			"name", objKey.Name,
			"kind", obj.GetObjectKind(),
		)
		selectedTargets[i] = selectedTarget{
			Namespace: objKey.Namespace,
			Name:      objKey.Name,
		}
		if err := api.RefreshObject(ctx, c, obj); err != nil {
			objLogger.Error(err, "error refreshing")
			failureCount++
			selectedTargets[i].Success = false
		} else {
			objLogger.Debug("successfully refreshed object")
			successCount++
			selectedTargets[i].Success = true
		}
	}
	result := getResult(len(objList), successCount, failureCount)
	summary := fmt.Sprintf("Refreshed %d of %d selected resources",
		successCount,
		len(objList),
	)
	return selectedTargets, result, summary
}

func getResult(total, successes, failures int) string {
	var result string
	switch {
	case successes == total:
		result = resultSuccess
	case failures == total:
		result = resultFailure
	default:
		result = resultPartialSuccess
	}
	return result
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
				shouldRefresh, err := shouldRefresh(ctx, wh, repoURL, qualifiers...)
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
	for _, wh := range toRefresh {
		whLogger := logger.WithValues(
			"namespace", wh.Namespace,
			"name", wh.Name,
		)
		if err := api.RefreshObject(ctx, c, wh); err != nil {
			failures++
			whLogger.Error(err, "failed to refresh Warehouse")
		} else {
			whLogger.Debug("marked Warehouse for refresh")
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

func shouldRefresh(
	ctx context.Context,
	wh kargoapi.Warehouse,
	repoURL string,
	qualifiers ...string,
) (bool, error) {
	var shouldRefresh bool
	for _, s := range wh.Spec.InternalSubscriptions {
		switch {
		case s.Git != nil && urls.NormalizeGit(s.Git.RepoURL) == repoURL:
			selector, err := commit.NewSelector(ctx, *s.Git, nil)
			if err != nil {
				return false, fmt.Errorf("error creating commit selector for Git subscription %q: %w",
					s.Git.RepoURL, err,
				)
			}
			shouldRefresh = slices.ContainsFunc(qualifiers, selector.MatchesRef)
		case s.Image != nil && urls.NormalizeImage(s.Image.RepoURL) == repoURL:
			selector, err := image.NewSelector(ctx, *s.Image, nil)
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
