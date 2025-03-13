package health

import (
	"context"
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// simpleEngine is a simple implementation of the Engine interface that uses
// built-in Checkers.
type simpleEngine struct {
	registry checkerRegistry
}

// NewSimpleEngine returns a simple implementation of the Engine interface that
// uses built-in Checkers.
func NewSimpleEngine() Engine {
	return &simpleEngine{registry: checkerReg}
}

// CheckHealth implements the Engine interface.
func (e *simpleEngine) Check(
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
func (e *simpleEngine) executeHealthChecks(
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
func (e *simpleEngine) executeHealthCheck(
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
	criteria.Project = project
	criteria.Stage = stage
	return checker.Check(ctx, criteria)
}
