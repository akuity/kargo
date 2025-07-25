package external

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/controller/git/commit"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/helm/chart"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

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
	qualifier string,
	repoURLs ...string,
) {
	logger := logging.LoggerFromContext(ctx)

	// De-dupe repository URLs
	slices.Sort(repoURLs)
	repoURLs = slices.Compact(repoURLs)
	// If there had been any empty strings in the slice, after sorting and
	// compacting, at most the zero element will be empty. If it is, remove it.
	if repoURLs[0] == "" {
		repoURLs = repoURLs[1:]
	}

	warehouses := []kargoapi.Warehouse{}

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

		warehouses = append(warehouses, ws.Items...)
	}

	slices.SortFunc(warehouses, func(lhs, rhs kargoapi.Warehouse) int {
		return strings.Compare(lhs.Namespace+lhs.Name, rhs.Namespace+rhs.Name)
	})
	warehouses = slices.CompactFunc(warehouses, func(lhs, rhs kargoapi.Warehouse) bool {
		return lhs.Namespace == rhs.Namespace && lhs.Name == rhs.Name
	})

	if qualifier != "" {
		refreshEligibleWarehouses := make([]kargoapi.Warehouse, 0, len(warehouses))
		for _, wh := range warehouses {
			shouldRefresh, err := shouldRefresh(ctx, wh.Spec.Subscriptions, qualifier, repoURLs...)
			if err != nil {
				// log the error but obscure the details from the response
				logger.Error(err, "failed to evaluate if warehouse needs refresh", "warehouse", wh.Name)
				xhttp.WriteErrorJSON(w, err)
				return
			}
			if *shouldRefresh {
				refreshEligibleWarehouses = append(refreshEligibleWarehouses, wh)
			}
		}
		warehouses = refreshEligibleWarehouses
	}

	logger.Debug("found Warehouses to refresh", "count", len(warehouses))

	var failures int
	for _, wh := range warehouses {
		objKey := client.ObjectKeyFromObject(&wh)
		if _, err := api.RefreshWarehouse(ctx, c, objKey); err != nil {
			logger.Error(err, "error refreshing Warehouse", "objectKey", objKey)
			failures++
		} else {
			logger.Debug("refreshed Warehouse", "objectKey", objKey)
		}
	}

	if failures > 0 {
		xhttp.WriteResponseJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": fmt.Sprintf("failed to refresh %d of %d warehouses",
					failures,
					len(warehouses),
				),
			},
		)
		return
	}
	xhttp.WriteResponseJSON(
		w,
		http.StatusOK,
		map[string]string{
			"msg": fmt.Sprintf("refreshed %d warehouse(s)", len(warehouses)),
		},
	)
}

func shouldRefresh(
	_ context.Context,
	subs []kargoapi.RepoSubscription,
	qualifier string,
	repoURLs ...string,
) (*bool, error) {
	var shouldRefresh bool
	subs = filterSubsByRepoURL(subs, repoURLs...) // only interested in subs that contain any of the repo URLs.
	for _, s := range subs {
		switch {
		case s.Git != nil:
			selector, err := commit.NewSelector(*s.Git, nil)
			if err != nil {
				return nil, fmt.Errorf("error creating commit selector for Git subscription %q: %w",
					s.Git.RepoURL, err,
				)
			}
			shouldRefresh = selector.MatchesRef(qualifier)
		case s.Image != nil:
			selector, err := image.NewSelector(*s.Image, nil)
			if err != nil {
				return nil, fmt.Errorf("error creating image selector for Image subscription %q: %w",
					s.Image.RepoURL, err,
				)
			}
			shouldRefresh = selector.MatchesTag(qualifier)
		case s.Chart != nil:
			selector, err := chart.NewSelector(*s.Chart, nil)
			if err != nil {
				return nil, fmt.Errorf("error creating chart selector for Chart subscription %q: %w",
					s.Chart.RepoURL, err,
				)
			}
			shouldRefresh = selector.MatchesVersion(qualifier)
		}
		if shouldRefresh {
			// exit early if we already found a match
			return &shouldRefresh, nil
		}
	}
	return &shouldRefresh, nil
}

// filterSubsByRepoURL deletes all subscriptions from subs that do not
// match any of the provided repository URLs; omitting them from processing.
func filterSubsByRepoURL(subs []kargoapi.RepoSubscription, repoURLs ...string) []kargoapi.RepoSubscription {
	containsRepoURL := func(sub kargoapi.RepoSubscription) bool {
		return sub.Image != nil && slices.Contains(repoURLs, sub.Image.RepoURL) ||
			sub.Git != nil && slices.Contains(repoURLs, sub.Git.RepoURL) ||
			sub.Chart != nil && slices.Contains(repoURLs,
				helm.NormalizeChartRepositoryURL(sub.Chart.RepoURL),
			)
	}
	return slices.DeleteFunc(subs, func(sub kargoapi.RepoSubscription) bool {
		return !containsRepoURL(sub)
	})
}
