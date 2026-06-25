package builtin

import (
	"fmt"
	"time"
)

// resolvePollInterval resolves the suggested poll interval for a step that
// reports itself Running while waiting for some condition to be satisfied. An
// explicitly configured interval takes precedence; otherwise the step's default
// is used.
//
// The returned duration is only a SUGGESTION: the Promotion reconciler enforces
// a lower bound on it (see calculateRequeueInterval) and may reconcile sooner in
// response to other events. Steps therefore do not enforce a floor themselves.
func resolvePollInterval(configured string, defaultInterval time.Duration) (time.Duration, error) {
	if configured == "" {
		return defaultInterval, nil
	}
	interval, err := time.ParseDuration(configured)
	if err != nil {
		// The configuration is validated against a JSON schema before reaching
		// this point, so a parse error here really should not happen.
		return 0, fmt.Errorf("error parsing pollInterval: %w", err)
	}
	return interval, nil
}
