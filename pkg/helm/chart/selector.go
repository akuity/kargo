package chart

import (
	"context"
	"fmt"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/helm"
)

// Selector is an interface for selecting chart versions from a Helm chart
// repository.
type Selector interface {
	// MatchesVersion returns a boolean value indicating whether or not the
	// Selector would consider a chart with the specified semantic version
	// eligible for selection.
	MatchesVersion(string) bool
	// Select selects charts from a Helm chart repository.
	Select(context.Context) ([]string, error)
}

// NewSelector returns some implementation of the Selector interface that
// selects chart versions from a Helm chart repository based on the provided
// subscription.
func NewSelector(
	sub kargoapi.ChartSubscription,
	creds *helm.Credentials,
) (Selector, error) {
	switch {
	case strings.HasPrefix(sub.RepoURL, "http://"),
		strings.HasPrefix(sub.RepoURL, "https://"):
		return newHTTPSelector(sub, creds)
	case strings.HasPrefix(sub.RepoURL, "oci://"):
		return newOCISelector(sub, creds)
	default:
		return nil, fmt.Errorf("repository URL %q is invalid", sub.RepoURL)
	}
}
