package function

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	whctrl "github.com/akuity/kargo/internal/controller/warehouses"
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
	warehouse kargoapi.Warehouse,
) expr.Option {
	return expr.Function(
		"commitFrom",
		getCommitFromWarehouse(ctx, c, project, warehouse),
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
	warehouse kargoapi.Warehouse,
) expr.Option {
	return expr.Function(
		"imageFrom",
		getImageFromWarehouse(ctx, c, project, warehouse),
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
	warehouse kargoapi.Warehouse,
) expr.Option {
	return expr.Function(
		"chartFrom",
		getChartFromWarehouse(ctx, c, project, warehouse),
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
	warehouse kargoapi.Warehouse,
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

		return whctrl.FindCommit(
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


// getImageFromWarehouse returns a function that finds container images based on repository
// URL and optional origin.
//
// The returned function uses the warehouse and references to locate the
// appropriate image within the project context.
func getImageFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse kargoapi.Warehouse,
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

		return whctrl.FindImage(
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

// getChartFromWarehouse returns a function that finds Helm charts based on repository URL,
// optional chart name, and optional origin.
//
// The returned function uses freight requests and references to locate the
// appropriate chart within the project context.
func getChartFromWarehouse(
	ctx context.Context,
	c client.Client,
	project string,
	warehouse kargoapi.Warehouse,
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

		return whctrl.FindChart(
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
