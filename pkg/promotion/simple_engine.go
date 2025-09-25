package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	gocache "github.com/patrickmn/go-cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
)

// ReservedStepAliasRegex is a regular expression that matches step aliases that
// are reserved for internal use.
var ReservedStepAliasRegex = regexp.MustCompile(`^(step|task)-\d+$`)

// ExprDataCacheFn is a function that returns a new cache to use in expression
// functions that consult the Kubernetes API.
//
// A new cache is created for each step execution, so that the cache is
// shared between all expression functions that are executed in the same step.
// This is important for performance, as our Kubernetes API client does not
// cache Secrets and ConfigMaps, but also for correctness, as the data may
// change between calls.
//
// It is allowed for the cache to be nil, in which case the expression functions
// will not cache their results.
type ExprDataCacheFn func() *gocache.Cache

// DefaultExprDataCacheFn returns a new gocache.Cache instance with
// default expiration and cleanup intervals. This is used as the default
// ExprDataCacheFn for the Engine.
func DefaultExprDataCacheFn() *gocache.Cache {
	return gocache.New(gocache.NoExpiration, gocache.NoExpiration)
}

// simpleEngine is a simple implementation of the Engine interface that uses
// built-in StepRunners.
type simpleEngine struct {
	orchestrator Orchestrator
}

// NewSimpleEngine returns a simple implementation of the Engine interface that
// uses built-in StepRunners.
func NewSimpleEngine(
	kargoClient client.Client,
	argocdClient client.Client,
	credsDB credentials.Database,
	cacheFunc ExprDataCacheFn,
) Engine {
	return &simpleEngine{
		orchestrator: NewLocalOrchestrator(stepRunnerReg, kargoClient, argocdClient, credsDB, cacheFunc),
	}
}

// Promote implements the Engine interface.
func (e *simpleEngine) Promote(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
) (_ Result, err error) {
	if promoCtx.WorkDir == "" {
		// If no working directory is provided, we create a temporary one.
		if promoCtx.WorkDir, err = e.setupWorkDir(promoCtx.WorkDir); err != nil {
			return Result{Status: kargoapi.PromotionPhaseErrored}, err
		}
		defer os.RemoveAll(promoCtx.WorkDir)
	}

	result, err := e.orchestrator.ExecuteSteps(ctx, promoCtx, steps)
	if err != nil {
		return Result{
			Status: kargoapi.PromotionPhaseErrored,
		}, fmt.Errorf("step execution failed: %w", err)
	}

	if result.Status == kargoapi.PromotionPhaseErrored {
		return result, errors.New(result.Message)
	}

	return result, nil
}

// setupWorkDir creates a temporary working directory if one is not provided.
func (e *simpleEngine) setupWorkDir(existingDir string) (string, error) {
	if existingDir != "" {
		return existingDir, nil
	}

	workDir, err := os.MkdirTemp("", "run-")
	if err != nil {
		return "", fmt.Errorf("temporary working directory creation failed: %w", err)
	}
	return workDir, nil
}
