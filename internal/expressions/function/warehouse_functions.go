package function

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/expr-lang/expr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func CommitFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse *kargoapi.Warehouse,
	discoveredCommits []kargoapi.GitDiscoveryResult,
) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommitFromWarehouse(ctx, c, project, warehouse, discoveredCommits),
		new(func(repoURL string, warehouse kargoapi.Warehouse) kargoapi.GitCommit),
		new(func(repoURL string) kargoapi.GitCommit),
	)
}

// ImageFromWarehouse returns an expr.Option that provides an `imageFrom()` function for
// use in expressions.
//
// The imageFrom function finds container images based on repository URL and
// optional origin, using the provided freight requests and references within
// the project context.
func ImageFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse *kargoapi.Warehouse,
	discoveredImages []kargoapi.ImageDiscoveryResult,
) expr.Option {
	return expr.Function(
		"imageFrom",
		getImageFromWarehouse(ctx, c, project, warehouse, discoveredImages),
		new(func(repoURL string, warehouse kargoapi.Warehouse) kargoapi.Image),
		new(func(repoURL string) kargoapi.Image),
	)
}

// ChartFromWarehouse returns an expr.Option that provides a `chartFrom()` function for
// use in expressions.
//
// The chartFrom function finds Helm charts based on repository URL, optional
// chart name, and optional origin, using the provided freight requests and
// references within the project context.
func ChartFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse *kargoapi.Warehouse,
	discoveredCharts []kargoapi.ChartDiscoveryResult,
) expr.Option {
	return expr.Function(
		"chartFrom",
		getChartFromWarehouse(ctx, c, project, warehouse, discoveredCharts),
		new(func(repoURL string, chartName string, origin kargoapi.FreightOrigin) kargoapi.Chart),
		new(func(repoURL string, chartName string) kargoapi.Chart),
		new(func(repoURL string, origin kargoapi.FreightOrigin) kargoapi.Chart),
		new(func(repoURL string) kargoapi.Chart),
	)
}

// getCommitFromWarehouse returns a function that finds Git commits based on repository URL
// and optional origin.
//
// The returned function uses warehouse to locate the
// appropriate commit within the project context.
func getCommitFromWarehouse(
	ctx context.Context,
	cl client.Client,
	project string,
	warehouse *kargoapi.Warehouse,
	discoveredCommits []kargoapi.GitDiscoveryResult,
) exprFn {
	return func(a ...any) (any, error) {
		// TODO: implement
		return nil, nil
	}
}

// getImageFromWarehouse returns a function that finds container images based on repository
// URL and optional origin.
//
// The returned function uses the warehouse and references to locate the
// appropriate image within the project context.
func getImageFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse *kargoapi.Warehouse,
	discoveredImages []kargoapi.ImageDiscoveryResult,
) exprFn {
	return func(a ...any) (any, error) {
		// TODO: implement
		return nil, nil
	}
}

// getChartFromWarehouse returns a function that finds Helm charts based on repository URL,
// optional chart name, and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate chart within the project context.
func getChartFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse *kargoapi.Warehouse,
	discoveredCharts []kargoapi.ChartDiscoveryResult,
) exprFn {
	return func(a ...any) (any, error) {
		// TODO: implement
		return nil, nil
	}
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
