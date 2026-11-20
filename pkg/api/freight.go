package api

import (
	"crypto/sha1" // nolint:gosec
	"fmt"
	"path"
	"slices"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/urls"
)

// GenerateFreightID deterministically calculates a piece of Freight's ID based
// on its contents and returns it.
func GenerateFreightID(f *kargoapi.Freight) string {
	size := len(f.Commits) + len(f.Images) + len(f.Charts) + len(f.Artifacts)
	hashParts := make([]string, 0, size)
	for _, commit := range f.Commits {
		hashParts = append(hashParts, commitHashPart(commit))
	}
	for _, image := range f.Images {
		hashParts = append(hashParts, imageHashPart(image))
	}
	for _, chart := range f.Charts {
		hashParts = append(hashParts, chartHashPart(chart))
	}
	for _, artifact := range f.Artifacts {
		hashParts = append(hashParts, artifactHashPart(artifact))
	}
	slices.Sort(hashParts)
	return fmt.Sprintf(
		"%x",
		sha1.Sum([]byte( // nolint:gosec
			fmt.Sprintf("%s:%s", f.Origin.String(), strings.Join(hashParts, "|")),
		)),
	)
}

// chartHashPart returns a string that uniquely identifies a specific version of
// a specific Helm chart.
func chartHashPart(chart kargoapi.Chart) string {
	// path.Join accounts for the possibility that chart.Name is empty
	base := fmt.Sprintf(
		"%s:%s",
		path.Join(urls.NormalizeChart(chart.RepoURL), chart.Name),
		chart.Version,
	)
	if chart.SubscriptionName != "" {
		return base + ":" + chart.SubscriptionName
	}
	return base
}

// commitHashPart returns a string that uniquely identifies a specific commit
// from a specific Git repository.
func commitHashPart(commit kargoapi.GitCommit) string {
	var base string
	if commit.Tag != "" {
		// If we have a tag, incorporate it into the canonical representation of a
		// commit used when calculating Freight ID. This is necessary because one
		// commit could have multiple tags. Suppose we have already detected a
		// commit with a tag v1.0.0-rc.1 and produced the corresponding Freight.
		// Later, that same commit is tagged as v1.0.0. If we don't incorporate
		// the tag into the ID, we will never produce a new/distinct piece of
		// Freight for the new tag.
		base = fmt.Sprintf(
			"%s:%s:%s",
			urls.NormalizeGit(commit.RepoURL), commit.Tag, commit.ID,
		)
	} else {
		base = fmt.Sprintf("%s:%s", urls.NormalizeGit(commit.RepoURL), commit.ID)
	}
	if commit.SubscriptionName != "" {
		return base + ":" + commit.SubscriptionName
	}
	return base
}

// imageHashPart returns a string that uniquely identifies a specific revision\
// of a container image from a specific container image repository.
func imageHashPart(img kargoapi.Image) string {
	// Note: This isn't the usual image representation using EITHER :<tag> OR
	// @<digest>. It is possible to have found an image with a tag that is already
	// known, but with a new digest -- as in the case of "mutable" tags like
	// "latest". It is equally possible to have found an image with a digest that
	// is already known, but has been re-tagged. To cover both cases, we
	// incorporate BOTH tag and digest into the canonical representation of an
	// image used when calculating Freight ID.
	base := fmt.Sprintf("%s:%s@%s", img.RepoURL, img.Tag, img.Digest)
	if img.SubscriptionName != "" {
		return base + ":" + img.SubscriptionName
	}
	return base
}

// artifactHashPart returns a string that uniquely identifies a specific
// revision of an artifact.
func artifactHashPart(ref kargoapi.ArtifactReference) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		ref.ArtifactType, ref.SubscriptionName, ref.Version,
	)
}
