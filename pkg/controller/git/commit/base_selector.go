package commit

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/pattern"
)

// baseSelector is a base implementation of Selector that provides common
// functionality for all Selector implementations. It is not intended to be used
// directly.
type baseSelector struct {
	repoURL               string
	creds                 *git.RepoCredentials
	insecureSkipTLSVerify bool
	filterExpression      *vm.Program
	includePaths          pattern.Matcher
	excludePaths          pattern.Matcher
	discoveryLimit        int

	gitCloneFn func(
		repoURL string,
		clientOpts *git.ClientOptions,
		cloneOpts *git.CloneOptions,
	) (git.Repo, error)
}

func newBaseSelector(
	sub kargoapi.GitSubscription,
	creds *git.RepoCredentials,
) (*baseSelector, error) {
	s := &baseSelector{
		repoURL:               sub.RepoURL,
		creds:                 creds,
		insecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
		discoveryLimit:        int(sub.DiscoveryLimit),
		gitCloneFn:            git.Clone,
	}
	var err error
	if sub.ExpressionFilter != "" {
		s.filterExpression, err = expr.Compile(sub.ExpressionFilter)
		if err != nil {
			return nil, fmt.Errorf("error compiling filter expression: %w", err)
		}
	}
	if s.includePaths, err = getPathSelectors(sub.IncludePaths); err != nil {
		return nil, fmt.Errorf("error parsing include path selectors: %w", err)
	}
	if s.excludePaths, err = getPathSelectors(sub.ExcludePaths); err != nil {
		return nil, fmt.Errorf("error parsing exclude path selectors: %w", err)
	}
	return s, nil
}

// MatchesPaths implements Selector.
func (b *baseSelector) MatchesPaths(paths []string) bool {
	for _, path := range paths {
		// If include is nil, all paths are implicitly included
		// Otherwise, check if the path matches the include pattern
		if b.includePaths != nil && !b.includePaths.Matches(path) {
			// Path not included, skip to next path
			continue
		}
		// Path is included (either implicitly or explicitly)
		// Now check if it should be excluded
		if b.excludePaths != nil && b.excludePaths.Matches(path) {
			// Path is explicitly excluded, skip to next path
			continue
		}
		// If we reach here, the path is included and not excluded
		return true
	}
	// None of the paths match our criteria
	return false
}

// getLoggerContext returns key/value pairs that can be used by any selector to
// enrich loggers with valuable context.
func (b *baseSelector) getLoggerContext() []any {
	return []any{
		"repo", b.repoURL,
		"pathConstrained", b.includePaths != nil || b.excludePaths != nil,
	}
}
