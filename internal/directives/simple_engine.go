package directives

import (
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/credentials"
)

// ReservedStepAliasRegex is a regular expression that matches step aliases that
// are reserved for internal use.
var ReservedStepAliasRegex = regexp.MustCompile(`^(step|task)-\d+$`)

// SimpleEngine is a simple engine that executes a list of PromotionSteps in
// sequence.
type SimpleEngine struct {
	registry      *StepRunnerRegistry
	credentialsDB credentials.Database
	kargoClient   client.Client
	argoCDClient  client.Client
}

// NewSimpleEngine returns a new SimpleEngine that uses the package's built-in
// StepRunnerRegistry.
func NewSimpleEngine(
	credentialsDB credentials.Database,
	kargoClient client.Client,
	argoCDClient client.Client,
) *SimpleEngine {
	return &SimpleEngine{
		registry:      builtins,
		credentialsDB: credentialsDB,
		kargoClient:   kargoClient,
		argoCDClient:  argoCDClient,
	}
}
