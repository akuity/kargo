package verification

import (
	"context"
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type AnalysisRun struct {
	Name              string
	Namespace         string
	CreationTimestamp metav1.Time
	GVK               kargoapi.AnalysisRunGVK
	Status            AnalysisRunStatus
}

type AnalysisRunStatus struct {
	CompletedAt *metav1.Time
	Phase       string
	Message     string
}

type AnalysisRunner interface {
	GetAnalysisRunStatus(context.Context, kargoapi.AnalysisRunReference) (*AnalysisRunStatus, error)
	AbortAnalysisRun(context.Context, kargoapi.AnalysisRunReference) error
	GetAnalysisRunPromotionName(context.Context, kargoapi.AnalysisRunReference) (string, error)
	FindExistingAnalysisRun(context.Context, types.NamespacedName, string) (*AnalysisRun, error)
	CreateAnalysisRun(
		context.Context,
		kargoapi.Stage,
		kargoapi.FreightCollection,
		string,
	) (run *AnalysisRun, errorMessage string, err error)
}

func (ver Verifier) analysisRunnerFor(runRef kargoapi.AnalysisRunReference) (AnalysisRunner, error) {
	var runnerGVK kargoapi.AnalysisRunGVK
	switch runRef.GVK {
	case kargoapi.AnalysisRunGVKRun, kargoapi.AnalysisRunGVKLegacy:
		runnerGVK = kargoapi.AnalysisRunGVKRun
	case kargoapi.AnalysisRunGVKRequest:
		runnerGVK = kargoapi.AnalysisRunGVKRequest
	default:
		return nil, fmt.Errorf("unsupported AnalysisRun GVK: %q", runRef.GVK)
	}
	runner, ok := ver.analysisRunners[runnerGVK]
	if !ok {
		return nil, fmt.Errorf("verifier does not have analysis runner for %v", runnerGVK)
	}
	return runner, nil

}

func (ver Verifier) getAnalysisRunStatus(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (*AnalysisRunStatus, error) {
	runner, err := ver.analysisRunnerFor(runRef)
	if err != nil {
		return nil, err
	}
	return runner.GetAnalysisRunStatus(ctx, runRef)
}

func (ver Verifier) abortAnalysisRun(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) error {
	runner, err := ver.analysisRunnerFor(runRef)
	if err != nil {
		return err
	}
	return runner.AbortAnalysisRun(ctx, runRef)
}

func (ver Verifier) getAnalysisRunPromotionName(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (string, error) {
	runner, err := ver.analysisRunnerFor(runRef)
	if err != nil {
		return "", err
	}
	return runner.GetAnalysisRunPromotionName(ctx, runRef)
}

// findExistingAnalysisRun finds the most recent AnalysisRun for a Stage and
// Freight collection in the namespace of the Stage. If no AnalysisRun is found,
// it returns nil.
func (ver Verifier) findExistingAnalysisRun(
	ctx context.Context,
	stage types.NamespacedName,
	freightColID string,
) (*AnalysisRun, error) {
	// We don't have a reference to tell us exactly which analysis runner to use
	// We try to load existing run from all runners
	runs := []AnalysisRun{}
	for _, runner := range ver.analysisRunners {
		run, err := runner.FindExistingAnalysisRun(
			ctx,
			stage,
			freightColID)
		if err != nil {
			return nil, err
		}
		if run != nil {
			runs = append(runs, *run)
		}
	}
	switch len(runs) {
	case 0:
		return nil, nil
	case 1:
		return &runs[0], nil
	default:
		// There were runs from multiple analysis runners. Pick the most recent one.
		// AnalysisRuns created with AnalysisRequest should be filtered out when using Rollouts runner,
		// so there should not be an overlap.
		mostRecent := slices.MinFunc(runs, func(lhs, rhs AnalysisRun) int {
			return rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time)
		})
		return &mostRecent, nil
	}
}

func (ver Verifier) createAnalysisRun(
	ctx context.Context,
	stage kargoapi.Stage,
	freightCollection kargoapi.FreightCollection,
	lastPromoName string,
) (*AnalysisRun, string, error) {
	if stage.IsTargetAware() && stage.Spec.Targets.Verification != nil {
		return ver.createFleetAnalysisRun(ctx, stage, freightCollection, lastPromoName)
	}

	runner, ok := ver.analysisRunners[kargoapi.AnalysisRunGVKRun]
	if !ok {
		return nil, "", fmt.Errorf("analysis runner not found for %v", kargoapi.AnalysisRunGVKRun)
	}
	return runner.CreateAnalysisRun(ctx, stage, freightCollection, lastPromoName)
}

func (ver Verifier) createFleetAnalysisRun(
	ctx context.Context,
	stage kargoapi.Stage,
	freightCollection kargoapi.FreightCollection,
	lastPromoName string,
) (*AnalysisRun, string, error) {
	runMode := stage.Spec.Targets.Verification.VerificationRunMode
	switch runMode {
	case kargoapi.VerificationRunModePerStage, "":
		runner, ok := ver.analysisRunners[kargoapi.AnalysisRunGVKRun]
		if !ok {
			return nil, "", fmt.Errorf("analysis runner not found for %v", kargoapi.AnalysisRunGVKRun)
		}
		return runner.CreateAnalysisRun(ctx, stage, freightCollection, lastPromoName)
	case kargoapi.VerificationRunModePerTarget:
		runner, ok := ver.analysisRunners[kargoapi.AnalysisRunGVKRequest]
		if !ok {
			return nil, "", fmt.Errorf("analysis runner not found for %v", kargoapi.AnalysisRunGVKRequest)
		}
		return runner.CreateAnalysisRun(ctx, stage, freightCollection, lastPromoName)
	default:
		return nil, "", fmt.Errorf("unknown verification runMode: %q", runMode)
	}
}
