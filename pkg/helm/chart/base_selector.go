package chart

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// baseSelector is a base implementation of Selector that provides common
// functionality for all Selector implementations. It is not intended to be used
// directly.
type baseSelector struct {
	repoURL        string
	constraint     *semver.Constraints
	discoveryLimit int
}

func newBaseSelector(
	sub kargoapi.ChartSubscription,
) (*baseSelector, error) {
	s := &baseSelector{
		repoURL:        sub.RepoURL,
		discoveryLimit: int(sub.DiscoveryLimit),
	}
	if sub.SemverConstraint != "" {
		var err error
		if s.constraint, err = semver.NewConstraint(
			sub.SemverConstraint,
		); err != nil {
			return nil,
				fmt.Errorf("error parsing constraint %q: %w", sub.SemverConstraint, err)
		}
	}
	return s, nil
}

// MatchesVersion implements Selector.
func (b *baseSelector) MatchesVersion(version string) bool {
	sv, err := semver.NewVersion(version)
	return err == nil && b.matchesSemver(sv)
}

func (b *baseSelector) matchesSemver(sv *semver.Version) bool {
	return b.constraint == nil || b.constraint.Check(sv)
}

// filterTags evaluates all provided semantic versions against the constraints
// defined by the b.MatchesVersion method, returning only those that satisfied
// those constraints.
func (b *baseSelector) filterSemvers(
	semvers semver.Collection,
) semver.Collection {
	var filtered = make(semver.Collection, 0, len(semvers))
	for _, sv := range semvers {
		if b.matchesSemver(sv) {
			filtered = append(filtered, sv)
		}
	}
	return slices.Clip(filtered)
}

// sort sorts the provided semantic versions from greatest to least in place.
func (b *baseSelector) sort(semvers semver.Collection) {
	slices.SortFunc(semvers, func(lhs, rhs *semver.Version) int {
		if comp := rhs.Compare(lhs); comp != 0 {
			return comp
		}
		// If the semvers tie, break the tie lexically using the original strings
		// used to construct the semvers. This ensures a deterministic comparison of
		// equivalent semvers, e.g., "1.0.0" > "1.0". The semver package's built-in
		// sort does not do this!
		return strings.Compare(rhs.Original(), lhs.Original())
	})
}

// semversToVersionStrings converts the provided list of semantic versions into
// a list of string representations.
func (b *baseSelector) semversToVersionStrings(
	semvers semver.Collection,
) []string {
	if b.discoveryLimit > 0 && len(semvers) > b.discoveryLimit {
		semvers = semvers[:b.discoveryLimit]
	}
	versions := make([]string, len(semvers))
	for i, semverVersion := range semvers {
		original := semverVersion.Original()
		versions[i] = original
	}
	return versions
}
