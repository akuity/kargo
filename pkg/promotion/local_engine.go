package promotion

import (
	"context"
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

// LocalEngine is an implementation of the Engine interface that uses
// built-in StepRunners locally.
type LocalEngine struct {
	orchestator Orchestrator
}

// NewLocalEngine returns an implementation of the Engine interface that
// uses built-in StepRunners locally.
func NewLocalEngine(
	kargoClient client.Client,
	argocdClient client.Client,
	credsDB credentials.Database,
	cacheFunc ExprDataCacheFn,
) *LocalEngine {
	return &LocalEngine{
		orchestator: NewLocalOrchestrator(
			DefaultStepRunnerRegistry,
			kargoClient,
			argocdClient,
			credsDB,
			cacheFunc,
		),
	}
}

// Promote implements the Engine interface.
func (e *LocalEngine) Promote(
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

	return e.orchestator.ExecuteSteps(ctx, promoCtx, steps)
}

// setupWorkDir creates a temporary working directory if one is not provided.
func (e *LocalEngine) setupWorkDir(existingDir string) (string, error) {
	if existingDir != "" {
		return existingDir, nil
	}

	workDir, err := os.MkdirTemp("", "run-")
	if err != nil {
		return "", fmt.Errorf("temporary working directory creation failed: %w", err)
	}
	return workDir, nil
}
