package function

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/expr-lang/expr"
	gocache "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/freight"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/urls"
)

type exprFn func(params ...any) (any, error)

// FreightOperations returns a slice of expr.Option containing functions for
// Freight operations.
//
// It provides `warehouse()`, `commitFrom()`, `imageFrom()`, and `chartFrom()`
// functions that can be used within expressions. The functions operate within
// the context of a given project with the provided freight requests and
// references.
func FreightOperations(
	ctx context.Context,
	c client.Client,
	project string,
	freightRequests []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) []expr.Option {
	return []expr.Option{
		Warehouse(),
		CommitFromFreight(ctx, c, project, freightRequests, freightRefs),
		ImageFromFreight(ctx, c, project, freightRequests, freightRefs),
		ChartFromFreight(ctx, c, project, freightRequests, freightRefs),
		ArtifactFromFreight(ctx, c, project, freightRequests, freightRefs),
		FreightMetadata(ctx, c, project),
	}
}

// DiscoveredArtifactsOperations returns a slice of expr.Option containing
// functions for retrieving artifacts from a Warehouse's discovered artifacts.
//
// It provides `commitFrom()`, `imageFrom()`, and `chartFrom()` functions for
// use in the context of expressions defining criteria that permit or block
// automatic Freight creation after artifact discovery. These functions behave
// identically to functions of the same names used within the context of a
// Promotion process, however, they are implemented differently since they
// resolve artifacts from different data.
func DiscoveredArtifactsOperations(artifacts *kargoapi.DiscoveredArtifacts) []expr.Option {
	return []expr.Option{
		CommitFromDiscoveredArtifacts(artifacts),
		ImageFromDiscoveredArtifacts(artifacts),
		ChartFromDiscoveredArtifacts(artifacts),
		ArtifactFromDiscoveredArtifacts(artifacts),
	}
}

// DataOperations returns a slice of expr.Option containing functions for
// data operations, such as accessing ConfigMaps and Secrets.
//
// When the cache parameter is set, the functions will cache the retrieved
// ConfigMaps and Secrets to avoid repeated API calls. This can
// improve performance when the same ConfigMaps and Secrets are accessed
// multiple times within the same expression evaluation.
func DataOperations(ctx context.Context, c client.Client, cache *gocache.Cache, project string) []expr.Option {
	return []expr.Option{
		ConfigMap(ctx, c, cache, project),
		SharedConfigMap(ctx, c, cache),
		Secret(ctx, c, cache, project),
		SharedSecret(ctx, c, cache),
		FreightMetadata(ctx, c, project),
		StageMetadata(ctx, c, project),
	}
}

// UtilityOperations returns a slice of expr.Option containing functions for
// utility operations, such as semantic version comparisons.
func UtilityOperations() []expr.Option {
	return []expr.Option{
		SemverDiff(),
		SemverParse(),
	}
}

// StatusOperations returns a slice of expr.Option containing functions for
// assessing the status of all preceding steps.
func StatusOperations(
	currentStepAlias string,
	stepExecMetas kargoapi.StepExecutionMetadataList,
) []expr.Option {
	return []expr.Option{
		Always(),
		Failure(stepExecMetas),
		Success(stepExecMetas),
		Status(currentStepAlias, stepExecMetas),
	}
}

func SharedSecret(ctx context.Context, c client.Client, cache *gocache.Cache) expr.Option {
	return expr.Function(
		"sharedSecret",
		getSecret(ctx, c, cache, os.Getenv("SHARED_RESOURCES_NAMESPACE"), false),
		new(func(name string) map[string]string),
	)
}

func SharedConfigMap(ctx context.Context, c client.Client, cache *gocache.Cache) expr.Option {
	return expr.Function(
		"sharedConfigMap",
		getConfigMap(ctx, c, cache, os.Getenv("SHARED_RESOURCES_NAMESPACE")),
		new(func(name string) map[string]string),
	)
}

// Warehouse returns an expr.Option that provides a `warehouse()` function
// for use in expressions.
//
// The warehouse function creates a v1alpha1.FreightOrigin of kind
// v1alpha1.Warehouse with the specified name.
func Warehouse() expr.Option {
	return expr.Function("warehouse", warehouse, new(func(name string) kargoapi.FreightOrigin))
}

// CommitFromFreight returns an expr.Option that provides a `commitFrom()` function
// for use in expressions.
//
// The commitFrom function finds Git commits based on repository URL and
// optional origin, using the provided freight requests and references within
// the project context.
func CommitFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommitFromFreight(ctx, c, project, freightReqs, freightRefs),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.GitCommit),
		new(func(repoURL string) kargoapi.GitCommit),
	)
}

// CommitFromDiscoveredArtifacts returns an expr.Option that provides a
// `commitFrom()` function for use, specifically, in expressions that define
// criteria that permit or block automatic Freight creation after artifact
// discovery.
//
// The commitFrom function finds the latest Git commit based on repository URL.
func CommitFromDiscoveredArtifacts(artifacts *kargoapi.DiscoveredArtifacts) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommitFromDiscoveredArtifacts(artifacts),
		new(func(repoURL string) kargoapi.DiscoveredCommit),
	)
}

// ImageFromFreight returns an expr.Option that provides an `imageFrom()` function for
// use in expressions.
//
// The imageFrom function finds container images based on repository URL and
// optional origin, using the provided freight requests and references within
// the project context.
func ImageFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"imageFrom",
		getImageFromFreight(ctx, c, project, freightReqs, freightRefs),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.Image),
		new(func(repoURL string) kargoapi.Image),
	)
}

// ImageFromDiscoveredArtifacts returns an expr.Option that provides an
// `imageFrom()` function for use, specifically, in expressions that define
// criteria that permit or block automatic Freight creation after artifact
// discovery.
//
// The imageFrom function finds the latest container image based on repository URL.
func ImageFromDiscoveredArtifacts(artifacts *kargoapi.DiscoveredArtifacts) expr.Option {
	return expr.Function(
		"imageFrom",
		getImageFromDiscoveredArtifacts(artifacts),
		new(func(repoURL string) kargoapi.DiscoveredImageReference),
	)
}

// ChartFromFreight returns an expr.Option that provides a `chartFrom()` function for
// use in expressions.
//
// The chartFrom function finds Helm charts based on repository URL, optional
// chart name, and optional origin, using the provided freight requests and
// references within the project context.
func ChartFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"chartFrom",
		getChartFromFreight(ctx, c, project, freightReqs, freightRefs),
		new(func(repoURL string, chartName string, origin kargoapi.FreightOrigin) kargoapi.Chart),
		new(func(repoURL string, chartName string) kargoapi.Chart),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.Chart),
		new(func(repoURL string) kargoapi.Chart),
	)
}

// ChartFromDiscoveredArtifacts returns an expr.Option that provides a
// `chartFrom()` function for use, specifically, in expressions that define
// criteria that permit or block automatic Freight creation after artifact
// discovery.
//
// The chartFrom function finds the latest Helm charts based on repository URL.
func ChartFromDiscoveredArtifacts(artifacts *kargoapi.DiscoveredArtifacts) expr.Option {
	return expr.Function(
		"chartFrom",
		getChartFromDiscoveredArtifacts(artifacts),
		new(func(repoURL string, chartName string) kargoapi.Chart),
		new(func(repoURL string) kargoapi.Chart),
	)
}

// ArtifactFromFreight returns an expr.Option that provides an `artifactFrom()`
// function for use in expressions.
//
// The artifactFrom() function finds artifacts based on the provided
// subscription name and optional origin, using the provided freight requests
// and references within the project context.
func ArtifactFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"artifactFrom",
		getArtifactFromFreight(ctx, c, project, freightReqs, freightRefs),
		new(func(name string, origin kargoapi.FreightOrigin) expressionFriendlyArtifactReference),
		new(func(name string) expressionFriendlyArtifactReference),
	)
}

// ArtifactFromDiscoveredArtifacts returns an expr.Option that provides an
// `artifactFrom()` function for use, specifically, in expressions that define
// criteria that permit or block automatic Freight creation after artifact
// discovery.
//
// The artifactFrom() function finds artifacts based on the subscription name.
func ArtifactFromDiscoveredArtifacts(
	artifacts *kargoapi.DiscoveredArtifacts,
) expr.Option {
	return expr.Function(
		"artifactFrom",
		getArtifactFromDiscoveredArtifacts(artifacts),
		new(func(name string) expressionFriendlyArtifactReference),
	)
}

func FreightMetadata(
	ctx context.Context,
	c client.Client,
	project string,
) expr.Option {
	return expr.Function(
		"freightMetadata",
		freightMetadata(ctx, c, project),
		new(func(freightRefName, key string) any), // Deprecated
		new(func(freightRefName string) map[string]any),
	)
}

func freightMetadata(
	ctx context.Context,
	c client.Client,
	project string,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 && len(a) != 2 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		freightRefName, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}
		if freightRefName == "" {
			return nil, fmt.Errorf("freight ref name must not be empty")
		}

		// Retrieve the Freight object as unstructured because it bypasses the
		// client's cache. This is essential for cases where Freight metadata is
		// being accessed very shortly after having been updated.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   kargoapi.GroupVersion.Group,
			Version: kargoapi.GroupVersion.Version,
			Kind:    "Freight",
		})
		if err := c.Get(
			ctx,
			client.ObjectKey{Namespace: project, Name: freightRefName},
			u,
		); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to get freight %s: %w", freightRefName, err)
		}
		freight := &kargoapi.Freight{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			u.Object,
			freight,
		); err != nil {
			return nil, fmt.Errorf(
				"error converting unstructured object to Freight: %w", err,
			)
		}

		// If only one argument, return the whole metadata map
		if len(a) == 1 {
			if freight.Status.Metadata == nil {
				return nil, nil
			}

			decoded := make(map[string]any, len(freight.Status.Metadata))
			for k, v := range freight.Status.Metadata {
				var val any
				if err := json.Unmarshal(v.Raw, &val); err != nil {
					return nil, fmt.Errorf("failed to unmarshal metadata value for key %s: %w", k, err)
				}
				decoded[k] = val
			}
			return decoded, nil
		}

		// Deprecated: If two arguments, return the value for the key
		key, ok := a[1].(string)
		if !ok {
			return nil, fmt.Errorf("second argument must be string, got %T", a[1])
		}
		if key == "" {
			return nil, fmt.Errorf("metadata key must not be empty")
		}

		var data any
		found, err := freight.Status.GetMetadata(key, &data)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata %s from freight %s: %w", key, freightRefName, err)
		}
		if !found {
			return nil, nil
		}
		return data, nil
	}
}

// StageMetadata returns an expr.Option that provides a `stageMetadata()` function
// for use in expressions.
//
// Usage:
//   - `stageMetadata(stageName)` returns the entire metadata map for the Stage.
func StageMetadata(
	ctx context.Context,
	c client.Client,
	project string,
) expr.Option {
	return expr.Function(
		"stageMetadata",
		stageMetadata(ctx, c, project),
		new(func(stageName string) map[string]any),
	)
}

func stageMetadata(
	ctx context.Context,
	c client.Client,
	project string,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		stageName, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
		}
		if stageName == "" {
			return nil, fmt.Errorf("stage name must not be empty")
		}

		// Retrieve the Stage object as unstructured because it bypasses the
		// client's cache. This is essential for cases where Stage metadata is being
		// accessed very shortly after having been updated.
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   kargoapi.GroupVersion.Group,
			Version: kargoapi.GroupVersion.Version,
			Kind:    "Stage",
		})
		if err := c.Get(
			ctx, client.ObjectKey{Namespace: project, Name: stageName},
			u,
		); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to get stage %s: %w", stageName, err)
		}
		stage := &kargoapi.Stage{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
			u.Object,
			stage,
		); err != nil {
			return nil, fmt.Errorf(
				"error converting unstructured object to Stage: %w", err,
			)
		}
		if stage.Status.Metadata == nil {
			return nil, nil
		}
		decoded := make(map[string]any, len(stage.Status.Metadata))
		for k, v := range stage.Status.Metadata {
			var val any
			if err := json.Unmarshal(v.Raw, &val); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata value for key %s: %w", k, err)
			}
			decoded[k] = val
		}
		return decoded, nil
	}
}

// ConfigMap returns an expr.Option that provides a `configMap()` function for
// use in expressions.
func ConfigMap(ctx context.Context, c client.Client, cache *gocache.Cache, project string) expr.Option {
	return expr.Function(
		"configMap",
		getConfigMap(ctx, c, cache, project),
		new(func(name string) map[string]string),
	)
}

// Secret returns an expr.Option that provides a `secret()` function for use in
// expressions.
func Secret(ctx context.Context, c client.Client, cache *gocache.Cache, project string) expr.Option {
	return expr.Function(
		"secret",
		getSecret(ctx, c, cache, project, true),
		new(func(name string) map[string]string),
	)
}

// SemverDiff returns an expr.Option that provides a `semverDiff()` function for
// use in expressions.
//
// The semverDiff function compares two semantic version strings and returns a
// string indicating the magnitude of difference between them -- one of:
// "Major", "Minor", "Patch", "Metadata", "None", or "Incomparable" if either or
// both arguments are not valid semantic versions.
func SemverDiff() expr.Option {
	return expr.Function(
		"semverDiff",
		semverDiff,
		new(func(ver1Str, ver2Str string) string),
	)
}

// SemverParse returns an expr.Option that provides a `semverParse()` function
// for use in expressions.
//
// The semverParse function parses a semantic version string and returns a
// *semver.Version struct. This allows direct access to version component
// methods like Major(), Minor(), Patch(), Prerelease(), and Metadata(), as
// well as utility methods like IncMajor(), IncMinor(), and IncPatch().
func SemverParse() expr.Option {
	return expr.Function(
		"semverParse",
		semverParse,
		new(func(verStr string) *semver.Version),
	)
}

// Always returns an expr.Option that provides an `always()` function
// for use in expressions.
//
// The `always()` function unconditionally returns true.
func Always() expr.Option {
	return expr.Function(
		"always",
		func(...any) (any, error) {
			return true, nil
		},
		new(func() bool),
	)
}

// Failure returns an expr.Option that provides a `failure()` function
// for use in expressions.
//
// The `failure()` function checks the status of all preceding steps and
// returns true if any of them have failed or errored and false otherwise.
func Failure(stepExecMetas kargoapi.StepExecutionMetadataList) expr.Option {
	return expr.Function("failure", hasFailure(stepExecMetas), new(func() bool))
}

// Success returns an expr.Option that provides a `success()` function
// for use in expressions.
//
// The `success()` function checks the status of all preceding steps and
// returns true if none of them have failed or errored and false otherwise.
func Success(stepExecMetas kargoapi.StepExecutionMetadataList) expr.Option {
	return expr.Function(
		"success",
		func(a ...any) (any, error) {
			failed, err := hasFailure(stepExecMetas)(a...)
			return !failed.(bool), err // nolint: forcetypeassert
		},
		new(func() bool),
	)
}

// Status returns an expr.Option that provides a `status()` function
// for use in expressions.
//
// The `status()` function retrieves the status of a specific step by its alias
// within the current promotion context; returning its value as a string.
func Status(
	currentStepAlias string,
	stepExecMetas kargoapi.StepExecutionMetadataList,
) expr.Option {
	return expr.Function(
		"status",
		getStatus(currentStepAlias, stepExecMetas),
		new(func(alias string) string),
	)
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

// getCommitFromFreight returns a function that finds Git commits based on repository URL
// and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate commit within the project context.
func getCommitFromFreight(
	ctx context.Context,
	cl client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) == 0 || len(a) > 2 {
			return nil, fmt.Errorf("expected 1-2 arguments, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		var desiredOrigin *kargoapi.FreightOrigin
		if len(a) == 2 {
			origin, ok := a[1].(kargoapi.FreightOrigin)
			if !ok {
				return nil, fmt.Errorf("second argument must be FreightOrigin, got %T", a[1])
			}
			desiredOrigin = &origin
		}

		return freight.FindCommit(
			ctx,
			cl,
			project,
			freightReqs,
			desiredOrigin,
			freightRefs,
			repoURL,
		)
	}
}

// getCommitFromDiscoveredArtifacts returns a function that finds Git commits based on repository URL.
func getCommitFromDiscoveredArtifacts(artifacts *kargoapi.DiscoveredArtifacts) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		if artifacts == nil {
			return nil, nil
		}

		repoURL = urls.NormalizeGit(repoURL)
		for _, ca := range artifacts.Git {
			if urls.NormalizeGit(ca.RepoURL) != repoURL {
				continue
			}
			if len(ca.Commits) > 0 {
				return ca.Commits[0], nil
			}
		}
		return nil, nil
	}
}

// getImageFromFreight returns a function that finds container images based on repository
// URL and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate image within the project context.
func getImageFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightRequests []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) == 0 || len(a) > 2 {
			return nil, fmt.Errorf("expected 1-2 arguments, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		var desiredOrigin *kargoapi.FreightOrigin
		if len(a) == 2 {
			origin, ok := a[1].(kargoapi.FreightOrigin)
			if !ok {
				return nil, fmt.Errorf("second argument must be FreightOrigin, got %T", a[1])
			}
			desiredOrigin = &origin
		}

		return freight.FindImage(
			ctx,
			c,
			project,
			freightRequests,
			desiredOrigin,
			freightRefs,
			repoURL,
		)
	}
}

// getImageFromDiscoveredArtifacts returns a function that finds the latest container image based on repository URL.
func getImageFromDiscoveredArtifacts(artifacts *kargoapi.DiscoveredArtifacts) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		if artifacts == nil {
			return nil, nil
		}

		repoURL = urls.NormalizeImage(repoURL)
		for _, ia := range artifacts.Images {
			if urls.NormalizeImage(ia.RepoURL) != repoURL {
				continue
			}
			if len(ia.References) > 0 {
				return ia.References[0], nil
			}
		}
		return nil, nil
	}
}

// getChartFromFreight returns a function that finds Helm charts based on repository URL,
// optional chart name, and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate chart within the project context.
func getChartFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) == 0 || len(a) > 3 {
			return nil, fmt.Errorf("expected 1-3 arguments, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		var chartName string
		var desiredOrigin *kargoapi.FreightOrigin

		if len(a) >= 2 {
			if name, ok := a[1].(string); ok {
				chartName = name
			} else if origin, ok := a[1].(kargoapi.FreightOrigin); ok {
				desiredOrigin = &origin
			} else {
				return nil, fmt.Errorf("second argument must be string or FreightOrigin, got %T", a[1])
			}
		}

		if len(a) == 3 {
			if chartName == "" {
				return nil, fmt.Errorf("when using three arguments, second argument must be string, got %T", a[1])
			}
			origin, ok := a[2].(kargoapi.FreightOrigin)
			if !ok {
				return nil, fmt.Errorf("third argument must be FreightOrigin, got %T", a[2])
			}
			desiredOrigin = &origin
		}

		return freight.FindChart(
			ctx,
			c,
			project,
			freightReqs,
			desiredOrigin,
			freightRefs,
			repoURL,
			chartName,
		)
	}
}

// getChartFromDiscoveredArtifacts returns a function that finds the latest Helm chart based on repository URL.
func getChartFromDiscoveredArtifacts(artifacts *kargoapi.DiscoveredArtifacts) exprFn {
	return func(a ...any) (any, error) {
		if len(a) == 0 || len(a) > 2 {
			return nil, fmt.Errorf("expected 1-2 arguments, got %d", len(a))
		}

		repoURL, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		var chartName string
		if len(a) == 2 {
			chartName, ok = a[1].(string)
			if !ok {
				return nil, fmt.Errorf("second argument must be string, got %T", a[1])
			}
		}

		if artifacts == nil {
			return nil, nil
		}

		repoURL = urls.NormalizeChart(repoURL)
		for _, ca := range artifacts.Charts {
			if urls.NormalizeChart(ca.RepoURL) != repoURL || (ca.Name != chartName && chartName != "") {
				continue
			}
			if len(ca.Versions) > 0 {
				return kargoapi.Chart{
					RepoURL: repoURL,
					Name:    ca.Name,
					Version: ca.Versions[0],
				}, nil
			}
		}
		return nil, nil
	}
}

// getArtifactFromFreight returns a function that finds an artifact based on the
// provided subscription name and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate artifact within the project context.
func getArtifactFromFreight(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) == 0 || len(a) > 2 {
			return nil, fmt.Errorf("expected 1-2 arguments, got %d", len(a))
		}

		subName, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		var desiredOrigin *kargoapi.FreightOrigin
		if len(a) == 2 {
			origin, ok := a[1].(kargoapi.FreightOrigin)
			if !ok {
				return nil,
					fmt.Errorf("second argument must be FreightOrigin, got %T", a[1])
			}
			desiredOrigin = &origin
		}

		artifact, err := freight.FindArtifact(
			ctx,
			c,
			project,
			freightReqs,
			desiredOrigin,
			freightRefs,
			subName,
		)
		if err != nil {
			return nil, fmt.Errorf("error finding artifact from subscription %s: %w", subName, err)
		}
		if artifact == nil {
			return nil, nil
		}

		// artifact.Metadata is just JSON. Unpack it into a map[string]any so it's
		// easily accessible from within an expression.
		exprArtifact := expressionFriendlyArtifactReference{
			ArtifactType:     artifact.ArtifactType,
			SubscriptionName: artifact.SubscriptionName,
			Version:          artifact.Version,
			Metadata:         map[string]any{},
		}
		if artifact.Metadata != nil {
			if err := json.Unmarshal(
				artifact.Metadata.Raw,
				&exprArtifact.Metadata,
			); err != nil {
				return nil, fmt.Errorf(
					"error unmarshaling metadata for artifact from subscription %s: %w",
					subName, err,
				)
			}
		}

		return exprArtifact, nil
	}
}

// getArtifactFromDiscoveredArtifacts returns a function that finds an artifact
// based on the provided subscription name.
func getArtifactFromDiscoveredArtifacts(
	artifacts *kargoapi.DiscoveredArtifacts,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		subName, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument must be string, got %T", a[0])
		}

		if artifacts == nil {
			return nil, nil
		}

		var artifact *kargoapi.ArtifactReference
		for _, result := range artifacts.Results {
			if result.SubscriptionName != subName {
				continue
			}
			if len(result.ArtifactReferences) > 0 {
				artifact = &result.ArtifactReferences[0]
			}
		}
		if artifact == nil {
			return nil, nil
		}

		// artifact.Metadata is just JSON. Unpack it into a map[string]any so it's
		// easily accessible from within an expression.
		exprArtifact := expressionFriendlyArtifactReference{
			ArtifactType:     artifact.ArtifactType,
			SubscriptionName: artifact.SubscriptionName,
			Version:          artifact.Version,
			Metadata:         map[string]any{},
		}
		if artifact.Metadata != nil {
			if err := json.Unmarshal(
				artifact.Metadata.Raw,
				&exprArtifact.Metadata,
			); err != nil {
				return nil, fmt.Errorf(
					"error unmarshaling artifact details for subscription %s: %w",
					subName, err,
				)
			}
		}

		return exprArtifact, nil
	}
}

// getConfigMap returns a function that retrieves a ConfigMap by its name
// within the specified project namespace. If the ConfigMap is not found,
// it returns an empty map.
//
// If a cache is provided, it will be used to store the retrieved ConfigMap
// data to avoid repeated API calls. The cache key is generated based on a
// prefix, project name, and ConfigMap name. Because of this, the same cache
// can be shared with other functions that accept a cache parameter (e.g.,
// getSecret) without worrying about key collisions.
func getConfigMap(ctx context.Context, c client.Client, cache *gocache.Cache, project string) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		name, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
		}

		cacheKey := getCacheKey(cacheKeyPrefixConfigMap, project, name)
		if cache != nil {
			if cachedData, ok := cache.Get(cacheKey); ok {
				if cachedData == nil {
					return map[string]string{}, nil
				}
				if data, ok := cachedData.(map[string]string); ok {
					return maps.Clone(data), nil
				}
			}
		}

		var cfgMap corev1.ConfigMap
		if err := c.Get(
			ctx,
			client.ObjectKey{
				Namespace: project,
				Name:      name,
			},
			&cfgMap,
		); err != nil {
			if apierrors.IsNotFound(err) {
				return map[string]string{}, nil
			}
			return nil, fmt.Errorf("failed to get configmap %s: %w", name, err)
		}

		if cache != nil {
			cache.Set(cacheKey, maps.Clone(cfgMap.Data), gocache.NoExpiration)
		}

		return cfgMap.Data, nil
	}
}

// getSecret returns a function that retrieves a Secret by its name within the
// specified project namespace. If the Secret is not found, it returns an empty
// map.
//
// If a cache is provided, it will be used to store the retrieved Secret data to
// avoid repeated API calls. The cache key is generated based on a prefix,
// project name, and Secret name. Because of this, the same cache can be shared
// with other functions that accept a cache parameter (e.g., getConfigMap)
// without worrying about key collisions.
func getSecret(
	ctx context.Context,
	c client.Client,
	cache *gocache.Cache,
	project string,
	hasDirectAccess bool,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		name, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
		}

		cacheKey := getCacheKey(cacheKeyPrefixSecret, project, name)
		if cache != nil {
			cachedData, ok := cache.Get(cacheKey)
			if ok {
				if cachedData == nil {
					return map[string]string{}, nil
				}
				if data, ok := cachedData.(map[string]string); ok {
					return maps.Clone(data), nil
				}
			}
		}

		var secret corev1.Secret
		if err := c.Get(
			ctx,
			client.ObjectKey{
				Namespace: project,
				Name:      name,
			},
			&secret,
		); err != nil {
			if apierrors.IsNotFound(err) {
				return map[string]string{}, nil
			}
			return nil, fmt.Errorf("failed to get secret %s: %w", name, err)
		}

		// limit shared secret access to generic credentials only
		if hasDirectAccess || isGenericSecretType(secret) {
			data := make(map[string]string)
			for k, v := range secret.Data {
				data[k] = string(v)
			}
			if cache != nil {
				cache.Set(cacheKey, maps.Clone(data), gocache.NoExpiration)
			}
			return data, nil
		}
		return map[string]string{}, nil
	}
}

func isGenericSecretType(secret corev1.Secret) bool {
	return secret.Labels[kargoapi.LabelKeyCredentialType] == kargoapi.LabelValueCredentialTypeGeneric
}

func hasFailure(stepExecMetas kargoapi.StepExecutionMetadataList) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 0 {
			return nil, fmt.Errorf("expected 0 arguments, got %d", len(a))
		}
		return stepExecMetas.HasFailures(), nil
	}
}

func getStatus(
	currentStepAlias string,
	stepExecMetas kargoapi.StepExecutionMetadataList,
) exprFn {
	var currentStepNamespace string
	if parts := strings.Split(currentStepAlias, kargo.PromotionAliasSeparator); len(parts) == 2 {
		currentStepNamespace = parts[0]
	}
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return "", fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		alias, ok := a[0].(string)
		if !ok {
			return "", fmt.Errorf("argument must be string, got %T", a[0])
		}

		if alias == "" {
			return "", fmt.Errorf("argument must not be empty")
		}

		for _, stepExecMeta := range stepExecMetas {
			stepShortAlias := stepExecMeta.Alias
			var stepNamespace string
			if parts := strings.Split(stepExecMeta.Alias, kargo.PromotionAliasSeparator); len(parts) == 2 {
				stepNamespace = parts[0]
				stepShortAlias = parts[1]
			}
			if stepNamespace == currentStepNamespace && stepShortAlias == alias {
				return string(stepExecMeta.Status), nil
			}
		}
		return "", nil
	}
}

// semverDiff compares two semantic version strings and returns a string
// indicating the magnitude of difference between them -- one of: "Major",
// "Minor", "Patch", "Metadata", "None", or "Incomparable" if either or both
// arguments are not valid semantic versions.
func semverDiff(a ...any) (any, error) {
	if len(a) != 2 {
		return nil, fmt.Errorf("expected 2 arguments, got %d", len(a))
	}

	ver1Str, ok := a[0].(string)
	if !ok {
		return nil, fmt.Errorf("first argument must be string, got %T", a[0])
	}

	ver2Str, ok := a[1].(string)
	if !ok {
		return nil, fmt.Errorf("second argument must be string, got %T", a[1])
	}

	ver1, err := semver.NewVersion(ver1Str)
	if err != nil {
		return "Incomparable", nil
	}
	ver2, err := semver.NewVersion(ver2Str)
	if err != nil {
		return "Incomparable", nil
	}
	if ver1.Major() != ver2.Major() {
		return "Major", nil
	}
	if ver1.Minor() != ver2.Minor() {
		return "Minor", nil
	}
	if ver1.Patch() != ver2.Patch() {
		return "Patch", nil
	}
	if ver1.Metadata() != ver2.Metadata() {
		return "Metadata", nil
	}
	return "None", nil
}

// semverParse parses a semantic version string and returns a *semver.Version
// struct. This enables users to access version component methods like Major(),
// Minor(), Patch(), as well as utility methods like IncMajor(), IncMinor(),
// and IncPatch() for version manipulation in expressions.
func semverParse(a ...any) (any, error) {
	if len(a) != 1 {
		return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
	}

	verStr, ok := a[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be string, got %T", a[0])
	}

	ver, err := semver.NewVersion(verStr)
	if err != nil {
		return nil, fmt.Errorf("invalid semantic version: %w", err)
	}

	return ver, nil
}

const (
	cacheKeyPrefixConfigMap = "ConfigMap"
	cacheKeyPrefixSecret    = "Secret"
)

// getCacheKey generates a cache key for the given prefix, project, and name.
// The cache key is a string formatted as "<prefix>/<project>/<name>".
func getCacheKey(prefix, project, name string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, project, name)
}

// expressionFriendlyArtifactReference exists because the Metadata field of an
// actual kargoapi.ArtifactReference is of type *apiextensionsv1.JSON and its
// contents are not easy to access within an expression. This similar type has a
// Metadata field of type map[string]any instead, which IS easy to access within
// an expression.
type expressionFriendlyArtifactReference struct {
	ArtifactType     string
	SubscriptionName string
	Version          string
	Metadata         map[string]any
}
