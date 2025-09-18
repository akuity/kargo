package function

import (
	"context"
	"fmt"

	semver "github.com/Masterminds/semver/v3"
	"github.com/expr-lang/expr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libsemver "github.com/akuity/kargo/internal/controller/semver"
	"github.com/akuity/kargo/pkg/logging"
)

// WarehouseOperations returns a slice of expr.Option containing functions for
// Warehouse operations.
//
// It provides `warehouse()`, `commitFrom()`, `imageFrom()`, and `chartFrom()`
// functions that can be used within expressions.
func WarehouseOperations(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) []expr.Option {
	return []expr.Option{
		Warehouse(),
		CommitFromWarehouse(ctx, wh, artifacts),
		ImageFromWarehouse(ctx, wh, artifacts),
		ChartFromWarehouse(ctx, wh, artifacts),
	}
}

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
// The commitFrom function finds the latest Git commit based on repository URL.
func CommitFromWarehouse(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommitFromWarehouse(ctx, wh, artifacts),
		new(func(repoURL string) kargoapi.GitCommit),
	)
}

// ImageFromWarehouse returns an expr.Option that provides an `imageFrom()` function for
// use in expressions.
//
// The imageFrom function finds the latest container image based on repository URL.
func ImageFromWarehouse(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) expr.Option {
	return expr.Function(
		"imageFrom",
		getImageFromWarehouse(ctx, wh, artifacts),
		new(func(repoURL string) kargoapi.Image),
	)
}

// ChartFromWarehouse returns an expr.Option that provides a `chartFrom()` function for
// use in expressions.
//
// The chartFrom function finds the latest Helm charts based on repository URL.
func ChartFromWarehouse(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) expr.Option {
	return expr.Function(
		"chartFrom",
		getChartFromWarehouse(ctx, wh, artifacts),
		new(func(repoURL string) kargoapi.Chart),
	)
}

// getCommitFromWarehouse returns a function that finds Git commits based on repository URL.
func getCommitFromWarehouse(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) exprFn {
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
			if s.Git != nil && s.Git.RepoURL == repoURL && len(artifacts.Git) != 0 {
				logger.Debug("number of discovered git artifacts",
					"count", len(artifacts.Git),
				)
				for i, dr := range artifacts.Git {
					if dr.RepoURL != repoURL {
						continue
					}
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
		logger.Debug("found latest commit", "commit", latestCommit.Tag)
		return &kargoapi.GitCommit{
			RepoURL:   repoURL,
			ID:        latestCommit.ID,
			Branch:    latestCommit.Branch,
			Tag:       latestCommit.Tag,
			Message:   latestCommit.Subject,
			Author:    latestCommit.Author,
			Committer: latestCommit.Committer,
		}, nil
	}
}

// getImageFromWarehouse returns a function that finds the latest container image based on repository URL.
func getImageFromWarehouse(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) exprFn {
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

		var latestImgRef *kargoapi.DiscoveredImageReference
		for _, s := range wh.Spec.Subscriptions {
			if s.Image != nil && s.Image.RepoURL == repoURL && len(artifacts.Images) != 0 {
				logger.Debug("number of discovered image artifacts",
					"count", len(artifacts.Images),
				)
				for i, dr := range artifacts.Images {
					if dr.RepoURL != repoURL {
						continue
					}
					logger.Debug("discovered image artifact",
						"index", i,
						"numImageRefs", len(dr.References),
					)
					for _, ref := range dr.References {
						if latestImgRef == nil {
							latestImgRef = &ref
							continue
						}
						if ref.CreatedAt.After(latestImgRef.CreatedAt.Time) {
							latestImgRef = &ref
						}
					}
				}
			}
		}
		if latestImgRef == nil {
			return nil, fmt.Errorf("no images found for repoURL %q", repoURL)
		}
		logger.Debug("found latest image reference", "ref", latestImgRef)
		return &kargoapi.Image{
			RepoURL:     repoURL,
			Tag:         latestImgRef.Tag,
			Digest:      latestImgRef.Digest,
			Annotations: latestImgRef.Annotations,
		}, nil
	}
}

// getChartFromWarehouse returns a function that finds the latest Helm chart based on repository URL.
func getChartFromWarehouse(
	ctx context.Context,
	wh *kargoapi.Warehouse,
	artifacts *kargoapi.DiscoveredArtifacts,
) exprFn {
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
			if s.Chart != nil && s.Chart.RepoURL == repoURL && len(artifacts.Charts) != 0 {
				logger.Debug("number of discovered chart artifacts",
					"count", len(artifacts.Charts),
				)
				for i, dr := range artifacts.Charts {
					if dr.RepoURL != repoURL {
						continue
					}
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
		if latestChartVersion == nil {
			return nil, fmt.Errorf("no charts found for repoURL %q", repoURL)
		}
		return &kargoapi.Chart{Version: latestChartVersion.String()}, nil
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
