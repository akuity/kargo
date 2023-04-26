package images

import "github.com/Masterminds/semver"

// Everything in this file is a workaround for non-deterministic sorting in the
// Masterminds/semver package.
//
// These workarounds can be removed after
// https://github.com/argoproj-labs/argocd-image-updater/pull/559
// is merged AND that change makes it into an Image Updater release.

// semverCollection is a replacement for semver.Collection that breaks version
// comparison ties through a lexical comparison of the original version strings.
// Using this, instead of semver.Collection, when sorting will yield
// deterministic results that semver.Collection will not yield.
type semverCollection []*semver.Version

// Len returns the length of a collection. The number of Version instances
// on the slice.
func (s semverCollection) Len() int {
	return len(s)
}

// Less is needed for the sort interface to compare two Version objects on the
// slice. If checks if one is less than the other.
func (s semverCollection) Less(i, j int) bool {
	comp := s[i].Compare(s[j])
	if comp != 0 {
		return comp < 0
	}
	return s[i].Original() < s[j].Original()
}

// Swap is needed for the sort interface to replace the Version objects
// at two different positions in the slice.
func (s semverCollection) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
