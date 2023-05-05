package images

import (
	"sort"

	"github.com/Masterminds/semver"
	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/tag"
)

// Everything in this file is a workaround for non-deterministic sorting in the
// Masterminds/semver package. More specifically, everything in this file is a
// replacement for particular Argo CD Image Updater functionality that relies on
// the non-deterministic sort from Masterminds/semver.
//
// These workarounds can be removed after
// https://github.com/argoproj-labs/argocd-image-updater/pull/559
// is merged AND that change makes it into an Image Updater release.

// getNewestVersionFromTags is cribbed from Argo CD Image Updater and minimally
// modified to fix the non-deterministic semver sorts.
func getNewestVersionFromTags(
	img *image.ContainerImage,
	vc *image.VersionConstraint,
	tagList *tag.ImageTagList,
) (string, error) {
	var availableTags []string
	switch vc.Strategy {
	case image.StrategySemVer:
		availableTags = sortBySemVer(tagList)
	case image.StrategyName:
		availableTags = sortByName(tagList)
	case image.StrategyLatest:
		availableTags = sortByDate(tagList)
	case image.StrategyDigest:
		availableTags = sortByName(tagList)
	}

	considerTags := []string{}

	// It makes no sense to proceed if we have no available tags
	if len(availableTags) == 0 {
		return "", nil
	}

	// The given constraint MUST match a semver constraint
	var semverConstraint *semver.Constraints
	var err error
	if vc.Strategy == image.StrategySemVer {
		// TODO: Shall we really ensure a valid semver on the current tag?
		// This prevents updating from a non-semver tag currently.
		if img.ImageTag != nil && img.ImageTag.TagName != "" {
			if _, err = semver.NewVersion(img.ImageTag.TagName); err != nil {
				return "", err
			}
		}

		if vc.Constraint != "" {
			if vc.Strategy == image.StrategySemVer {
				semverConstraint, err = semver.NewConstraint(vc.Constraint)
				if err != nil {
					return "", err
				}
			}
		}
	}

	// Loop through all tags to check whether it's an update candidate.
	for _, tag := range availableTags {
		if vc.Strategy == image.StrategySemVer {
			// Non-parseable tag does not mean error - just skip it
			ver, err := semver.NewVersion(tag)
			if err != nil {
				continue
			}

			// If we have a version constraint, check image tag against it. If the
			// constraint is not satisfied, skip tag.
			if semverConstraint != nil {
				if !semverConstraint.Check(ver) {
					continue
				}
			}
		} else if vc.Strategy == image.StrategyDigest {
			if tag != vc.Constraint {
				continue
			}
		}

		// Append tag as update candidate
		considerTags = append(considerTags, tag)
	}

	// If we found tags to consider, return the most recent tag found according
	// to the update strategy.
	if len(considerTags) > 0 {
		return considerTags[len(considerTags)-1], nil
	}

	return "", nil
}

// sortBySemVer is a replacement for similar functionality in Argo CD Image
// Updater. It relies on a workaround for Masterminds/semver package's
// non-deterministic sorts.
func sortBySemVer(tags *tag.ImageTagList) []string {
	semvers := []*semver.Version{}
	for _, tagName := range tags.Tags() {
		semver, err := semver.NewVersion(tagName)
		if err != nil {
			continue
		}
		semvers = append(semvers, semver)
	}
	sort.Sort(semverCollection(semvers))
	sortedTagNames := make([]string, len(semvers))
	for i, semver := range semvers {
		sortedTagNames[i] = semver.Original()
	}
	return sortedTagNames
}

// sortByName is a replacement for similar functionality in Argo CD Image
// Updater. It relies on Argo CD Image Updater to do most of its work. The main
// difference is that it returns just tag names instead of tag.ImageTag objects.
func sortByName(tags *tag.ImageTagList) []string {
	sortedTags := tags.SortByName()
	sortedTagNames := make([]string, len(sortedTags))
	for i, tag := range sortedTags {
		sortedTagNames[i] = tag.TagName
	}
	return sortedTagNames
}

// sortByDate is a replacement for similar functionality in Argo CD Image
// Updater. It relies on Argo CD Image Updater to do most of its work. The main
// difference is that it returns just tag names instead of tag.ImageTag objects.
func sortByDate(tags *tag.ImageTagList) []string {
	sortedTags := tags.SortByDate()
	sortedTagNames := make([]string, len(sortedTags))
	for i, tag := range sortedTags {
		sortedTagNames[i] = tag.TagName
	}
	return sortedTagNames
}
