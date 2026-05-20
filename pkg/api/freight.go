package api

import (
	"context"
	"crypto/sha1" // nolint:gosec
	"fmt"
	"path"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
		// Incorporate the tag so commits with multiple tags produce
		// distinct Freight even when the commit SHA is the same.
		base = fmt.Sprintf("%s:%s:%s", urls.NormalizeGit(commit.RepoURL), commit.Tag, commit.ID)
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

// GetFreightByNameOrAlias returns a pointer to the Freight resource specified
// by the project, and name OR alias arguments. If no such resource is found,
// nil is returned instead.
func GetFreightByNameOrAlias(
	ctx context.Context,
	c client.Client,
	project string,
	name string,
	alias string,
) (*kargoapi.Freight, error) {
	if name != "" {
		return GetFreight(
			ctx,
			c,
			types.NamespacedName{
				Namespace: project,
				Name:      name,
			},
		)
	}
	return GetFreightByAlias(ctx, c, project, alias)
}

// GetFreight returns a pointer to the Freight resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetFreight(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Freight, error) {
	freight := kargoapi.Freight{}
	if err := c.Get(ctx, namespacedName, &freight); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Freight %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &freight, nil
}

// GetFreightByAlias returns a pointer to the Freight resource specified by the
// project and alias arguments. If no such resource is found, nil is returned
// instead.
func GetFreightByAlias(
	ctx context.Context,
	c client.Client,
	project string,
	alias string,
) (*kargoapi.Freight, error) {
	freightList := kargoapi.FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{
			kargoapi.LabelKeyAlias: alias,
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight with alias %q in namespace %q: %w",
			alias,
			project,
			err,
		)
	}
	if len(freightList.Items) == 0 {
		return nil, nil
	}
	return &freightList.Items[0], nil
}

// ListFreightByCurrentStage returns a list of Freight resources that think
// they're currently in use by the Stage specified.
func ListFreightByCurrentStage(
	ctx context.Context,
	c client.Client,
	stage *kargoapi.Stage,
) ([]kargoapi.Freight, error) {
	freightList := kargoapi.FreightList{}
	if err := c.List(
		ctx,
		&freightList,
		client.InNamespace(stage.Namespace),
		client.MatchingFields{"currentlyIn": stage.Name},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight in namespace %q with current stage %q: %w",
			stage.Namespace,
			stage.Name,
			err,
		)
	}
	return freightList.Items, nil
}
