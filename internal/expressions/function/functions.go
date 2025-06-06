package function

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/expr-lang/expr"
	gocache "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	"github.com/akuity/kargo/internal/kargo"
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
		CommitFrom(ctx, c, project, freightRequests, freightRefs),
		ImageFrom(ctx, c, project, freightRequests, freightRefs),
		ChartFrom(ctx, c, project, freightRequests, freightRefs),
		FreightMetadata(ctx, c, project),
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
		Secret(ctx, c, cache, project),
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

// Warehouse returns an expr.Option that provides a `warehouse()` function
// for use in expressions.
//
// The warehouse function creates a v1alpha1.FreightOrigin of kind
// v1alpha1.Warehouse with the specified name.
func Warehouse() expr.Option {
	return expr.Function("warehouse", warehouse, new(func(name string) kargoapi.FreightOrigin))
}

// CommitFrom returns an expr.Option that provides a `commitFrom()` function
// for use in expressions.
//
// The commitFrom function finds Git commits based on repository URL and
// optional origin, using the provided freight requests and references within
// the project context.
func CommitFrom(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommit(ctx, c, project, freightReqs, freightRefs),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.GitCommit),
		new(func(repoURL string) kargoapi.GitCommit),
	)
}

// ImageFrom returns an expr.Option that provides an `imageFrom()` function for
// use in expressions.
//
// The imageFrom function finds container images based on repository URL and
// optional origin, using the provided freight requests and references within
// the project context.
func ImageFrom(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"imageFrom",
		getImage(ctx, c, project, freightReqs, freightRefs),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.Image),
		new(func(repoURL string) kargoapi.Image),
	)
}

// ChartFrom returns an expr.Option that provides a `chartFrom()` function for
// use in expressions.
//
// The chartFrom function finds Helm charts based on repository URL, optional
// chart name, and optional origin, using the provided freight requests and
// references within the project context.
func ChartFrom(
	ctx context.Context,
	c client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	freightRefs []kargoapi.FreightReference,
) expr.Option {
	return expr.Function(
		"chartFrom",
		getChart(ctx, c, project, freightReqs, freightRefs),
		new(func(repoURL string, chartName string, origin kargoapi.FreightOrigin) kargoapi.Chart),
		new(func(repoURL string, chartName string) kargoapi.Chart),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.Chart),
		new(func(repoURL string) kargoapi.Chart),
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
		new(func(freightRefName, key string) any),
	)
}

func freightMetadata(
	ctx context.Context,
	c client.Client,
	project string,
) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 2 {
			return nil, fmt.Errorf("expected 2 argument, got %d", len(a))
		}

		freightRefName, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
		}

		if freightRefName == "" {
			return nil, fmt.Errorf("freight ref name must not be empty")
		}

		key, ok := a[1].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[1])
		}
		if key == "" {
			return nil, fmt.Errorf("metadata key must not be empty")
		}

		freightData := kargoapi.Freight{}

		if err := c.Get(ctx, client.ObjectKey{
			Namespace: project,
			Name:      freightRefName,
		}, &freightData); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to get freight %s: %w", freightRefName, err)
		}

		var data any
		found, err := freightData.Status.GetMetadata(key, &data)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata %s from freight %s: %w", key, freightRefName, err)
		}
		if !found {
			return nil, nil
		}

		return data, nil
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
		getSecret(ctx, c, cache, project),
		new(func(name string) map[string]string),
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

// getCommit returns a function that finds Git commits based on repository URL
// and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate commit within the project context.
func getCommit(
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

// getImage returns a function that finds container images based on repository
// URL and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate image within the project context.
func getImage(
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

// getChart returns a function that finds Helm charts based on repository URL,
// optional chart name, and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate chart within the project context.
func getChart(
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
			if kubeerr.IsNotFound(err) {
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
func getSecret(ctx context.Context, c client.Client, cache *gocache.Cache, project string) exprFn {
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
			if kubeerr.IsNotFound(err) {
				return map[string]string{}, nil
			}
			return nil, fmt.Errorf("failed to get secret %s: %w", name, err)
		}

		data := make(map[string]string)
		for k, v := range secret.Data {
			data[k] = string(v)
		}

		if cache != nil {
			cache.Set(cacheKey, maps.Clone(data), gocache.NoExpiration)
		}

		return data, nil
	}
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

const (
	cacheKeyPrefixConfigMap = "ConfigMap"
	cacheKeyPrefixSecret    = "Secret"
)

// getCacheKey generates a cache key for the given prefix, project, and name.
// The cache key is a string formatted as "<prefix>/<project>/<name>".
func getCacheKey(prefix, project, name string) string {
	return fmt.Sprintf("%s/%s/%s", prefix, project, name)
}
