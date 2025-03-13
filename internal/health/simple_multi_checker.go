package health

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// simpleMultiChecker is a simple implementation of the MultiChecker interface
// that delegates health checks to built-in Checkers.
type simpleMultiChecker struct {
	registry checkerRegistry
}

// NewSimpleMultiChecker returns a simple implementation of the MultiChecker
// interface that uses built-in Checkers.
func NewSimpleMultiChecker() MultiChecker {
	return &simpleMultiChecker{registry: checkerReg}
}

// Check implements the MultiChecker interface.
func (e *simpleMultiChecker) Check(
	ctx context.Context,
	project string,
	stage string,
	checks []Criteria,
) kargoapi.Health {
	status, issues, output := e.executeHealthChecks(ctx, project, stage, checks)
	if len(output) == 0 {
		return kargoapi.Health{
			Status: status,
			Issues: issues,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal health output: %s", err.Error()))
	}

	return kargoapi.Health{
		Status: status,
		Issues: issues,
		Output: &apiextensionsv1.JSON{Raw: b},
	}
}

// executeHealthChecks executes a list of HealthChecks in sequence.
func (e *simpleMultiChecker) executeHealthChecks(
	ctx context.Context,
	project string,
	stage string,
	checks []Criteria,
) (kargoapi.HealthState, []string, []map[string]any) {
	var (
		aggregatedStatus = kargoapi.HealthStateHealthy
		aggregatedIssues []string
		aggregatedOutput = make([]map[string]any, 0, len(checks))
	)

	for _, check := range checks {
		select {
		case <-ctx.Done():
			aggregatedStatus = aggregatedStatus.Merge(kargoapi.HealthStateUnknown)
			aggregatedIssues = append(aggregatedIssues, ctx.Err().Error())
			return aggregatedStatus, aggregatedIssues, aggregatedOutput
		default:
		}

		result := e.executeHealthCheck(ctx, project, stage, check)
		aggregatedStatus = aggregatedStatus.Merge(result.Status)
		aggregatedIssues = append(aggregatedIssues, result.Issues...)

		if result.Output != nil {
			aggregatedOutput = append(aggregatedOutput, result.Output)
		}
	}

	return aggregatedStatus, aggregatedIssues, aggregatedOutput
}

// executeHealthCheck executes a single HealthCheck.
func (e *simpleMultiChecker) executeHealthCheck(
	ctx context.Context,
	project string,
	stage string,
	criteria Criteria,
) Result {
	checker := e.registry.getChecker(criteria.Kind)
	if checker == nil {
		return Result{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				fmt.Sprintf("no health checker registered for health check kind %q", criteria.Kind),
			},
		}
	}
	return checker.Check(ctx, project, stage, criteria)
}
