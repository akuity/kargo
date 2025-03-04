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
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
)

// GenerateFreightID deterministically calculates a piece of Freight's ID based
// on its contents and returns it.
func GenerateFreightID(f *kargoapi.Freight) string {
	size := len(f.Commits) + len(f.Images) + len(f.Charts)
	artifacts := make([]string, 0, size)
	for _, commit := range f.Commits {
		if commit.Tag != "" {
			// If we have a tag, incorporate it into the canonical representation of a
			// commit used when calculating Freight ID. This is necessary because one
			// commit could have multiple tags. Suppose we have already detected a
			// commit with a tag v1.0.0-rc.1 and produced the corresponding Freight.
			// Later, that same commit is tagged as v1.0.0. If we don't incorporate
			// the tag into the ID, we will never produce a new/distinct piece of
			// Freight for the new tag.
			artifacts = append(
				artifacts,
				fmt.Sprintf("%s:%s:%s", git.NormalizeURL(commit.RepoURL), commit.Tag, commit.ID),
			)
		} else {
			artifacts = append(
				artifacts,
				fmt.Sprintf("%s:%s", git.NormalizeURL(commit.RepoURL), commit.ID),
			)
		}
	}
	for _, image := range f.Images {
		artifacts = append(
			artifacts,
			// Note: This isn't the usual image representation using EITHER :<tag> OR @<digest>.
			// It is possible to have found an image with a tag that is already known, but with a
			// new digest -- as in the case of "mutable" tags like "latest". It is equally possible to
			// have found an image with a digest that is already known, but has been re-tagged.
			// To cover both cases, we incorporate BOTH tag and digest into the canonical
			// representation of an image used when calculating Freight ID.
			fmt.Sprintf("%s:%s@%s", image.RepoURL, image.Tag, image.Digest),
		)
	}
	for _, chart := range f.Charts {
		artifacts = append(
			artifacts,
			fmt.Sprintf(
				"%s:%s",
				// path.Join accounts for the possibility that chart.Name is empty
				path.Join(helm.NormalizeChartRepositoryURL(chart.RepoURL), chart.Name),
				chart.Version,
			),
		)
	}
	slices.Sort(artifacts)
	return fmt.Sprintf(
		"%x",
		sha1.Sum([]byte( // nolint:gosec
			fmt.Sprintf("%s:%s", f.Origin.String(), strings.Join(artifacts, "|")),
		)),
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
			kargoapi.AliasLabelKey: alias,
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
