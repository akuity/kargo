package verification

import (
	"context"
	"errors"
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
	Kind              kargoapi.AnalysisRunKind
	Status            AnalysisRunStatus
}

type AnalysisRunStatus struct {
	CompletedAt *metav1.Time
	Phase       string
	Message     string
}

type AnalysisRunner interface {
	Validate(action string) error
	GetAnalysisRunStatus(context.Context, kargoapi.AnalysisRunReference) (*AnalysisRunStatus, error)
	AbortAnalysisRun(context.Context, kargoapi.AnalysisRunReference) error
	FindExistingAnalysisRun(context.Context, types.NamespacedName, string) (*AnalysisRun, error)
	CreateAnalysisRun(
		context.Context,
		kargoapi.Stage,
		kargoapi.FreightCollection,
		string,
	) (run *AnalysisRun, errorMessage string, err error)
	ClearAnalysisRunsForStage(context.Context, *kargoapi.Stage) error
}

var ErrInvalid = errors.New("analysis Runner invalid")

func (ver Verifier) analysisRunnerFor(
	runRef kargoapi.AnalysisRunReference,
	action string,
) (AnalysisRunner, error) {
	var runnerKind kargoapi.AnalysisRunKind
	if runRef.Kind == "" {
		runnerKind = kargoapi.AnalysisRunKindRun
	} else {
		runnerKind = runRef.Kind
	}

	runner, ok := ver.analysisRunners[runnerKind]
	if !ok {
		return nil, fmt.Errorf("verifier does not have analysis runner for %v", runnerKind)
	}

	if validateErr := runner.Validate(action); validateErr != nil {
		return nil, validateErr
	}
	return runner, nil
}

func (ver Verifier) getAnalysisRunStatus(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) (*AnalysisRunStatus, error) {
	runner, err := ver.analysisRunnerFor(runRef, "get verification result")
	if err != nil {
		return nil, err
	}
	return runner.GetAnalysisRunStatus(ctx, runRef)
}

func (ver Verifier) abortAnalysisRun(
	ctx context.Context,
	runRef kargoapi.AnalysisRunReference,
) error {
	runner, err := ver.analysisRunnerFor(runRef, "abort verification")
	if err != nil {
		return err
	}
	return runner.AbortAnalysisRun(ctx, runRef)
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
	validateErrs := []error{}
	for _, runner := range ver.analysisRunners {
		// Since we scan all runners, some might be invalid. Only return validate error if no runs found.
		if validateErr := runner.Validate("find verification"); validateErr != nil {
			validateErrs = append(validateErrs, validateErr)
		}
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
		// If some runners are invalid, they might have contained the run we need
		if len(validateErrs) > 0 {
			return nil, validateErrs[0]
		}
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
	runner, ok, err := ver.selectRunnerForCreateFn(stage, ver.analysisRunners)
	if err != nil {
		// We don't return error if a runner is invalid to prevent reconciler restarts
		var returnErr error
		if !errors.Is(err, ErrInvalid) {
			returnErr = err
		}
		return nil, err.Error(), returnErr
	}
	if !ok {
		return nil, "", fmt.Errorf("analysis runner not found for stage %q", stage.Name)
	}
	return runner.CreateAnalysisRun(ctx, stage, freightCollection, lastPromoName)
}

func selectRunnerForCreate(
	_ kargoapi.Stage,
	runners map[kargoapi.AnalysisRunKind]AnalysisRunner,
) (AnalysisRunner, bool, error) {
	runner, ok := runners[kargoapi.AnalysisRunKindRun]
	if validateErr := runner.Validate("start verification"); validateErr != nil {
		return nil, false, validateErr
	}

	return runner, ok, nil
}
