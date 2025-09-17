package function

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libsemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/expr-lang/expr"

	semver "github.com/Masterminds/semver/v3"
)

// Warehouse returns an expr.Option that provides a `warehouse()` function
// for use in expressions.
//
// The warehouse function creates a v1alpha1.FreightOrigin of kind
// v1alpha1.Warehouse with the specified name.
func Warehouse() expr.Option {
	return expr.Function("warehouse", warehouse, new(func(name string) kargoapi.FreightOrigin))
}

// CommitFromWarehouse returns an expr.Option that provides a `commitFrom()` function
// for use in expressions.
//
// The commitFrom function finds Git commits based on repository URL and
// optional origin, using the provided warehouse within
// the project context.
func CommitFromWarehouse(ctx context.Context, wh *kargoapi.Warehouse) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommitFromWarehouse(ctx, wh),
		new(func(repoURL string) kargoapi.GitCommit),
	)
}

// ImageFromWarehouse returns an expr.Option that provides an `imageFrom()` function for
// use in expressions.
//
// The imageFrom function finds container images based on repository URL and
// optional origin, using the provided freight requests and references within
// the project context.
func ImageFromWarehouse(ctx context.Context, wh *kargoapi.Warehouse) expr.Option {
	return expr.Function(
		"imageFrom",
		getImageFromWarehouse(ctx, wh),
		new(func(repoURL string) kargoapi.Image),
	)
}

// ChartFromWarehouse returns an expr.Option that provides a `chartFrom()` function for
// use in expressions.
//
// The chartFrom function finds Helm charts based on repository URL, optional
// chart name, and optional origin, using the provided freight requests and
// references within the project context.
func ChartFromWarehouse(ctx context.Context, wh *kargoapi.Warehouse) expr.Option {
	return expr.Function(
		"chartFrom",
		getChartFromWarehouse(ctx, wh),
		new(func(repoURL string) kargoapi.Chart),
	)
}

// getCommitFromWarehouse returns a function that finds Git commits based on repository URL
// and optional origin.
//
// The returned function uses warehouse to locate the
// appropriate commit within the project context.
func getCommitFromWarehouse(ctx context.Context, wh *kargoapi.Warehouse) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		logger := logging.LoggerFromContext(ctx).WithValues(
			"repoURL", repoURL,
			"warehouse", wh.Name,
		)

		var latestCommit *kargoapi.DiscoveredCommit
		for _, s := range wh.Spec.Subscriptions {
			if s.Git != nil && s.Git.RepoURL == repoURL && len(wh.Status.DiscoveredArtifacts.Git) != 0 {
				logger.Debug("number of discovered git artifacts",
					"count", len(wh.Status.DiscoveredArtifacts.Git),
				)
				for i, dr := range wh.Status.DiscoveredArtifacts.Git {
					logger.Debug("checking discovered git artifact",
						"index", i,
						"numCommits", len(dr.Commits),
					)
					for _, c := range dr.Commits {
						if latestCommit == nil {
							latestCommit = &c
							continue
						}
						if c.CreatorDate.After(latestCommit.CreatorDate.Time) {
							latestCommit = &c
						}
					}
				}
			}
		}
		if latestCommit == nil {
			return nil, fmt.Errorf("no commits found for repoURL %q", repoURL)
		}
		return latestCommit, nil
	}
}

// getImageFromWarehouse returns a function that finds container images based on repository
// URL and optional origin.
//
// The returned function uses the warehouse and references to locate the
// appropriate image within the project context.
func getImageFromWarehouse(ctx context.Context, wh *kargoapi.Warehouse) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		logger := logging.LoggerFromContext(ctx).WithValues(
			"repoURL", repoURL,
			"warehouse", wh.Name,
		)

		var latestImage *kargoapi.DiscoveredImageReference
		for _, s := range wh.Spec.Subscriptions {
			if s.Image != nil && s.Image.RepoURL == repoURL && len(wh.Status.DiscoveredArtifacts.Images) != 0 {
				logger.Debug("number of discovered image artifacts",
					"count", len(wh.Status.DiscoveredArtifacts.Images),
				)
				for i, dr := range wh.Status.DiscoveredArtifacts.Images {
					logger.Debug("checking discovered image artifact",
						"index", i,
						"numImageRefs", len(dr.References),
					)
					for _, ref := range dr.References {
						if latestImage == nil {
							latestImage = &ref
							continue
						}
						if ref.CreatedAt.After(latestImage.CreatedAt.Time) {
							latestImage = &ref
						}
					}
				}
			}
		}
		if latestImage == nil {
			return nil, fmt.Errorf("no images found for repoURL %q", repoURL)
		}
		return latestImage, nil
	}
}

// getChartFromWarehouse returns a function that finds Helm charts based on repository URL,
// optional chart name, and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate chart within the project context.
func getChartFromWarehouse(ctx context.Context, wh *kargoapi.Warehouse) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		logger := logging.LoggerFromContext(ctx).WithValues(
			"repoURL", repoURL,
			"warehouse", wh.Name,
		)

		var latestChartVersion *semver.Version
		for _, s := range wh.Spec.Subscriptions {
			if s.Chart != nil && s.Chart.RepoURL == repoURL && len(wh.Status.DiscoveredArtifacts.Charts) != 0 {
				logger.Debug("number of discovered chart artifacts",
					"count", len(wh.Status.DiscoveredArtifacts.Charts),
				)
				for i, dr := range wh.Status.DiscoveredArtifacts.Charts {
					logger.Debug("checking discovered chart artifact",
						"index", i,
						"numVersions", len(dr.Versions),
					)
					v := getLatestVersion(dr)
					if latestChartVersion == nil {
						latestChartVersion = v
						continue
					}
					if v.GreaterThan(latestChartVersion) {
						latestChartVersion = v
					}
				}
			}
		}
		return kargoapi.DiscoveredImageReference{Tag: latestChartVersion.String()}, nil
	}
}

func getLatestVersion(cdr kargoapi.ChartDiscoveryResult) *semver.Version {
	var latestVersion *semver.Version
	for _, v := range cdr.Versions {
		sv := libsemver.Parse(v, false)
		if sv == nil {
			continue
		}
		if latestVersion == nil || sv.GreaterThan(latestVersion) {
			latestVersion = sv
		}
	}
	return latestVersion
}

// warehouse creates a FreightOrigin of kind Warehouse with the specified name.
//
// It returns an error if the argument count is incorrect or if the name is not
// a string.
func warehouse(a ...any) (any, error) {
	if len(a) != 1 {
		return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
	}

	name, ok := a[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be string, got %T", a[0])
	}

	if name == "" {
		return nil, fmt.Errorf("name must not be empty")
	}

	return kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: name,
	}, nil
}
