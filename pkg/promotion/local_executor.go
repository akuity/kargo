package promotion

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
)

// LocalStepExecutor is a concrete implementation of StepExecutor that
// executes steps locally using step runners registered in a registry.
type LocalStepExecutor struct {
	registry StepRunnerRegistry

	kargoClient  client.Client
	argoCDClient client.Client
	credsDB      credentials.Database
}

// NewLocalStepExecutor creates a new LocalStepExecutor with the provided
// step runner registry. This executor will use the registered runners to
// execute steps in the promotion process.
func NewLocalStepExecutor(
	registry StepRunnerRegistry,
	kargoClient, argoCDClient client.Client,
	credsDB credentials.Database,
) *LocalStepExecutor {
	return &LocalStepExecutor{
		registry:     registry,
		kargoClient:  kargoClient,
		argoCDClient: argoCDClient,
		credsDB:      credsDB,
	}
}

// ExecuteStep executes a single promotion Step using the registered step
// runner for the Step's kind. It handles any errors that occur during execution
// and returns a StepResult indicating the outcome of the step execution.
func (e *LocalStepExecutor) ExecuteStep(
	ctx context.Context,
	req StepExecutionRequest,
) (result StepResult, err error) {
	reg, err := e.registry.Get(req.Step.Kind)
	if err != nil {
		return StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error getting runner for step kind %q: %w", req.Step.Kind, err)
	}

	capabilities := StepRunnerCapabilities{}
	for _, capability := range reg.Metadata.RequiredCapabilities {
		switch capability {
		case StepCapabilityAccessControlPlane:
			capabilities.KargoClient = e.kargoClient
		case StepCapabilityAccessArgoCD:
			capabilities.ArgoCDClient = e.argoCDClient
		case StepCapabilityAccessCredentials:
			capabilities.CredsDB = e.credsDB
		}
	}

	factory := reg.Value
	runner := factory(capabilities)

	func() {
		defer func() {
			if r := recover(); r != nil {
				result = StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}
				err = &TerminalError{
					Err: fmt.Errorf("step panicked: %v", r),
				}
			}
		}()

		result, err = runner.Run(ctx, &req.Context)
	}()

	if err != nil {
		err = fmt.Errorf("error running step %q: %w", req.Step.Alias, err)
	}

	return result, err
}
