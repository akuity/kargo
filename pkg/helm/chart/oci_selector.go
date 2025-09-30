package chart

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/helm"
)

// ociSelector is an implementation of Selector that interacts with OCI Helm
// chart repositories.
type ociSelector struct {
	*baseSelector
	repo *remote.Repository
}

func newOCISelector(
	sub kargoapi.ChartSubscription,
	creds *helm.Credentials,
) (Selector, error) {
	base, err := newBaseSelector(sub)
	if err != nil {
		return nil, fmt.Errorf("error building base selector: %w", err)
	}

	ref, err := registry.ParseReference(strings.TrimPrefix(sub.RepoURL, "oci://"))
	if err != nil {
		return nil,
			fmt.Errorf("error parsing repository URL %q: %w", sub.RepoURL, err)
	}

	authorizer := helm.NewEphemeralAuthorizer()
	if creds != nil {
		if err = authorizer.Login(context.Background(), ref.Host(), creds.Username, creds.Password); err != nil {
			return nil, fmt.Errorf(
				"error logging in to OCI repository %q: %w",
				sub.RepoURL,
				err,
			)
		}
	}

	return &ociSelector{
		baseSelector: base,
		repo: &remote.Repository{
			Reference: ref,
			Client:    authorizer,
		},
	}, nil
}

// Select implements Selector.
func (o *ociSelector) Select(ctx context.Context) ([]string, error) {
	semvers := make(semver.Collection, 0, o.repo.TagListPageSize)
	if err := o.repo.Tags(ctx, "", func(tags []string) error {
		for _, tag := range tags {
			// OCI artifact tags are not allowed to contain the "+" character, which is
			// used by SemVer to separate the version from the build metadata. To work
			// around this, Helm uses "_" instead of "+".
			if sv, err := semver.StrictNewVersion(strings.ReplaceAll(tag, "_", "+")); err == nil {
				semvers = append(semvers, sv)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf(
			"error retrieving versions of chart from repository %q: %w",
			o.repoURL,
			err,
		)
	}
	semvers = o.filterSemvers(semvers)
	o.sort(semvers)
	return o.semversToVersionStrings(semvers), nil
}
