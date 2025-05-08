package function

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/controller/freight"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
	}
}

// DataOperations returns a slice of expr.Option containing functions for
// data operations, such as accessing ConfigMaps and Secrets.
func DataOperations(ctx context.Context, c client.Client, project string) []expr.Option {
	return []expr.Option{
		ConfigMap(ctx, c, project),
		Secret(ctx, c, project),
	}
}

// StatusOperations returns a slice of expr.Option containing functions for
// assessing the status of all preceding steps.
func StatusOperations(stepExecMetas kargoapi.StepExecutionMetadataList) []expr.Option {
	return []expr.Option{
		Always(),
		Failure(stepExecMetas),
		Success(stepExecMetas),
		Status(stepExecMetas),
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

// ConfigMap returns an expr.Option that provides a `configMap()` function for
// use in expressions.
func ConfigMap(ctx context.Context, c client.Client, project string) expr.Option {
	return expr.Function(
		"configMap",
		getConfigMap(ctx, c, project),
		new(func(name string) map[string]string),
	)
}

// Secret returns an expr.Option that provides a `secret()` function for use in
// expressions.
func Secret(ctx context.Context, c client.Client, project string) expr.Option {
	return expr.Function(
		"secret",
		getSecret(ctx, c, project),
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

func Status(stepExecMetas kargoapi.StepExecutionMetadataList) expr.Option {
	return expr.Function(
		"status",
		getStatus(stepExecMetas),
		new(func(alias string) kargoapi.PromotionStepStatus),
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

func getConfigMap(ctx context.Context, c client.Client, project string) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		name, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
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

		return cfgMap.Data, nil
	}
}

func getSecret(ctx context.Context, c client.Client, project string) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		name, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
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

		data := make(map[string]string, len(secret.Data))
		for k, v := range secret.Data {
			data[k] = string(v)
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

func getStatus(stepExecMetas kargoapi.StepExecutionMetadataList) exprFn {
	return func(a ...any) (any, error) {
		if len(a) != 1 {
			return nil, fmt.Errorf("expected 1 argument, got %d", len(a))
		}

		alias, ok := a[0].(string)
		if !ok {
			return nil, fmt.Errorf("argument must be string, got %T", a[0])
		}

		if alias == "" {
			return nil, fmt.Errorf("argument must not be empty")
		}

		for _, stepExecMeta := range stepExecMetas {
			if stepExecMeta.Alias == alias {
				return stepExecMeta.Status, nil
			}
		}
		return "", nil
	}
}
