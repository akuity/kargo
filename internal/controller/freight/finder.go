package freight

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libGit "github.com/akuity/kargo/internal/git"
)

func FindCommit(
	ctx context.Context,
	cl client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	desiredOrigin *kargoapi.FreightOrigin,
	freight []kargoapi.FreightReference,
	repoURL string,
	mustFind bool,
) (*kargoapi.GitCommit, error) {
	repoURL = libGit.NormalizeURL(repoURL)
	// If no origin was explicitly identified, we need to look at all possible
	// origins. If there's only one that could provide the commit we're looking
	// for, great. If there's more than one, there's ambiguity and we need to
	// return an error.
	if desiredOrigin == nil {
		for i := range freightReqs {
			requestedFreight := freightReqs[i]
			warehouse, err := kargoapi.GetWarehouse(
				ctx,
				cl,
				types.NamespacedName{
					Name:      requestedFreight.Origin.Name,
					Namespace: project,
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"error getting Warehouse %q in namespace %q: %w",
					requestedFreight.Origin.Name, project, err,
				)
			}
			if warehouse == nil {
				return nil, fmt.Errorf(
					"Warehouse %q not found in namespace %q",
					requestedFreight.Origin.Name, project,
				)
			}
			for _, sub := range warehouse.Spec.Subscriptions {
				if sub.Git != nil && libGit.NormalizeURL(sub.Git.RepoURL) == repoURL {
					if desiredOrigin != nil {
						return nil, fmt.Errorf(
							"multiple requested Freight could potentially provide a "+
								"commit from repo %s; please update promotion steps to "+
								"disambiguate",
							repoURL,
						)
					}
					desiredOrigin = &requestedFreight.Origin
				}
			}
		}
	}
	if desiredOrigin == nil {
		if mustFind {
			// There is no chance of finding the commit we're looking for.
			return nil, fmt.Errorf("commit from repo %s not found in referenced Freight", repoURL)
		}
		return nil, nil // The caller will decide how to deal with this
	}
	// We know exactly what we're after, so this should be easy
	for i := range freight {
		f := &freight[i]
		if f.Origin.Equals(desiredOrigin) {
			for j := range f.Commits {
				c := &f.Commits[j]
				if libGit.NormalizeURL(c.RepoURL) == repoURL {
					return c, nil
				}
			}
		}
	}
	// If we get to here, we looked at all the FreightReferences and didn't find
	// any that came from the desired origin. This could be because no Freight
	// from the desired origin has been promoted yet.
	if mustFind {
		return nil, fmt.Errorf("commit from repo %s not found in referenced Freight", repoURL)
	}
	return nil, nil // The caller will decide how to deal with this
}

func FindImage(
	ctx context.Context,
	cl client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	desiredOrigin *kargoapi.FreightOrigin,
	freight []kargoapi.FreightReference,
	repoURL string,
	mustFind bool,
) (*kargoapi.Image, error) {
	// If no origin was explicitly identified, we need to look at all possible
	// origins. If there's only one that could provide the commit we're looking
	// for, great. If there's more than one, there's ambiguity, and we need to
	// return an error.
	if desiredOrigin == nil {
		for i := range freightReqs {
			requestedFreight := freightReqs[i]
			warehouse, err := kargoapi.GetWarehouse(
				ctx,
				cl,
				types.NamespacedName{
					Name:      requestedFreight.Origin.Name,
					Namespace: project,
				},
			)
			if err != nil {
				return nil, err
			}
			if warehouse == nil {
				return nil, fmt.Errorf(
					"Warehouse %q not found in namespace %q",
					requestedFreight.Origin.Name, project,
				)
			}
			for _, sub := range warehouse.Spec.Subscriptions {
				if sub.Image != nil && sub.Image.RepoURL == repoURL {
					if desiredOrigin != nil {
						return nil, fmt.Errorf(
							"multiple requested Freight could potentially provide a container image from "+
								"repository %s: please provide a Freight origin to disambiguate",
							repoURL,
						)
					}
					desiredOrigin = &requestedFreight.Origin
				}
			}
		}
	}
	if desiredOrigin == nil {
		// There is no chance of finding the commit we're looking for. Just return
		// nil and let the caller decide what to do.
		if mustFind {
			return nil, fmt.Errorf("image from repo %s not found in referenced Freight", repoURL)
		}
		return nil, nil // The caller will decide how to deal with this
	}
	// We know exactly what we're after, so this should be easy
	for _, f := range freight {
		if f.Origin.Equals(desiredOrigin) {
			for _, i := range f.Images {
				if i.RepoURL == repoURL {
					return &i, nil
				}
			}
		}
	}
	// If we get to here, we looked at all the FreightReferences and didn't find
	// any that came from the desired origin. This could be because no Freight
	// from the desired origin has been promoted yet.
	if mustFind {
		return nil, fmt.Errorf("image from repo %s not found in referenced Freight", repoURL)
	}
	return nil, nil // The caller will decide how to deal with this
}

func FindChart(
	ctx context.Context,
	cl client.Client,
	project string,
	freightReqs []kargoapi.FreightRequest,
	desiredOrigin *kargoapi.FreightOrigin,
	freight []kargoapi.FreightReference,
	repoURL string,
	chartName string,
	mustFind bool,
) (*kargoapi.Chart, error) {
	// If no origin was explicitly identified, we need to look at all possible
	// origins. If there's only one that could provide the commit we're looking
	// for, great. If there's more than one, there's ambiguity, and we need to
	// return an error.
	if desiredOrigin == nil {
		for i := range freightReqs {
			requestedFreight := freightReqs[i]
			warehouse, err := kargoapi.GetWarehouse(
				ctx,
				cl,
				types.NamespacedName{
					Name:      requestedFreight.Origin.Name,
					Namespace: project,
				},
			)
			if err != nil {
				return nil, err
			}
			if warehouse == nil {
				return nil, fmt.Errorf(
					"Warehouse %q not found in namespace %q",
					requestedFreight.Origin.Name, project,
				)
			}
			for _, sub := range warehouse.Spec.Subscriptions {
				if sub.Chart != nil && sub.Chart.RepoURL == repoURL && sub.Chart.Name == chartName {
					if desiredOrigin != nil {
						return nil, fmt.Errorf(
							"multiple requested Freight could potentially provide a chart from "+
								"repository %s: please provide a Freight origin to disambiguate",
							repoURL,
						)
					}
					desiredOrigin = &requestedFreight.Origin
				}
			}
		}
		if desiredOrigin == nil {
			// There is no chance of finding the chart version we're looking for.
			if mustFind {
				if chartName == "" {
					return nil, fmt.Errorf("chart from repo %s not found in referenced Freight", repoURL)
				}
				return nil, fmt.Errorf("chart %q from repo %s not found in referenced Freight", chartName, repoURL)
			}
			return nil, nil // The caller will decide how to deal with this
		}
	}
	// We know exactly what we're after, so this should be easy
	for _, f := range freight {
		if f.Origin.Equals(desiredOrigin) {
			for _, c := range f.Charts {
				if c.RepoURL == repoURL && c.Name == chartName {
					return &c, nil
				}
			}
		}
	}
	// If we get to here, we looked at all the FreightReferences and didn't find
	// any that came from the desired origin. This could be because no Freight
	// from the desired origin has been promoted yet.
	if mustFind {
		if chartName == "" {
			return nil, fmt.Errorf("chart from repo %s not found in referenced Freight", repoURL)
		}
		return nil, fmt.Errorf("chart %q from repo %s not found in referenced Freight", chartName, repoURL)
	}
	return nil, nil // The caller will decide how to deal with this
}
