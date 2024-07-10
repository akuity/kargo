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
	stage *kargoapi.Stage,
	desiredOrigin *kargoapi.FreightOrigin,
	freight []kargoapi.FreightReference,
	repoURL string,
) (*kargoapi.GitCommit, error) {
	repoURL = libGit.NormalizeURL(repoURL)
	// If no origin was explicitly identified, we need to look at all possible
	// origins. If there's only one that could provide the commit we're looking
	// for, great. If there's more than one, there's ambiguity and we need to
	// return an error.
	if desiredOrigin == nil {
		for i := range stage.Spec.RequestedFreight {
			requestedFreight := stage.Spec.RequestedFreight[i]
			warehouse, err := kargoapi.GetWarehouse(
				ctx,
				cl,
				types.NamespacedName{
					Name:      requestedFreight.Origin.Name,
					Namespace: stage.Namespace,
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"error getting Warehouse %q in namespace %q: %w",
					requestedFreight.Origin.Name, stage.Namespace, err,
				)
			}
			if warehouse == nil {
				return nil, fmt.Errorf(
					"Warehouse %q not found in namespace %q",
					requestedFreight.Origin.Name, stage.Namespace,
				)
			}
			for _, sub := range warehouse.Spec.Subscriptions {
				if sub.Git != nil && libGit.NormalizeURL(sub.Git.RepoURL) == repoURL {
					if desiredOrigin != nil {
						return nil, fmt.Errorf(
							"multiple requested Freight could potentially provide a "+
								"commit from repo %s; please update promotion mechanisms to "+
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
		// There is no chance of finding the commit we're looking for. Just return
		// nil and let the caller decide what to do.
		return nil, nil
	}
	// We know exactly what we're after, so this should be easy
	for _, f := range freight {
		if f.Origin.Equals(desiredOrigin) {
			for _, c := range f.Commits {
				if libGit.NormalizeURL(c.RepoURL) == repoURL {
					return &c, nil
				}
			}
		}
	}
	// If we get to here, we looked at all the FreightReferences and didn't find
	// any that came from the desired origin. This could be because no Freight
	// from the desired origin has been promoted yet. Return nil and let the
	// caller decide what to do.
	return nil, nil
}

func FindImage(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	desiredOrigin *kargoapi.FreightOrigin,
	freight []kargoapi.FreightReference,
	repoURL string,
) (*kargoapi.Image, error) {
	// If no origin was explicitly identified, we need to look at all possible
	// origins. If there's only one that could provide the commit we're looking
	// for, great. If there's more than one, there's ambiguity and we need to
	// return an error.
	if desiredOrigin == nil {
		for i := range stage.Spec.RequestedFreight {
			requestedFreight := stage.Spec.RequestedFreight[i]
			warehouse, err := kargoapi.GetWarehouse(
				ctx,
				cl,
				types.NamespacedName{
					Name:      requestedFreight.Origin.Name,
					Namespace: stage.Namespace,
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"error getting Warehouse %q in namespace %q: %w",
					requestedFreight.Origin.Name, stage.Namespace, err,
				)
			}
			if warehouse == nil {
				return nil, fmt.Errorf(
					"Warehouse %q not found in namespace %q",
					requestedFreight.Origin.Name, stage.Namespace,
				)
			}
			for _, sub := range warehouse.Spec.Subscriptions {
				if sub.Image != nil && sub.Image.RepoURL == repoURL {
					if desiredOrigin != nil {
						return nil, fmt.Errorf(
							"multiple requested Freight could potentially provide a "+
								"container image from repo %s; please update promotion "+
								"mechanisms to disambiguate",
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
		return nil, nil
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
	// from the desired origin has been promoted yet. Return nil and let the
	// caller decide what to do.
	return nil, nil
}

func FindChart(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	desiredOrigin *kargoapi.FreightOrigin,
	freight []kargoapi.FreightReference,
	repoURL string,
	chartName string,
) (*kargoapi.Chart, error) {
	// If no origin was explicitly identified, we need to look at all possible
	// origins. If there's only one that could provide the commit we're looking
	// for, great. If there's more than one, there's ambiguity and we need to
	// return an error.
	if desiredOrigin == nil {
		for i := range stage.Spec.RequestedFreight {
			requestedFreight := stage.Spec.RequestedFreight[i]
			warehouse, err := kargoapi.GetWarehouse(
				ctx,
				cl,
				types.NamespacedName{
					Name:      requestedFreight.Origin.Name,
					Namespace: stage.Namespace,
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"error getting Warehouse %q in namespace %q: %w",
					requestedFreight.Origin.Name, stage.Namespace, err,
				)
			}
			if warehouse == nil {
				return nil, fmt.Errorf(
					"Warehouse %q not found in namespace %q",
					requestedFreight.Origin.Name, stage.Namespace,
				)
			}
			for _, sub := range warehouse.Spec.Subscriptions {
				if sub.Chart != nil && sub.Chart.RepoURL == repoURL && sub.Chart.Name == chartName {
					if desiredOrigin != nil {
						return nil, fmt.Errorf(
							"multiple requested Freight could potentially provide a Helm "+
								"chart from repo %s; please update promotion mechanisms to "+
								"disambiguate",
							repoURL,
						)
					}
					desiredOrigin = &requestedFreight.Origin
				}
			}
		}
		if desiredOrigin == nil {
			// There is no chance of finding the commit we're looking for. Just return
			// nil and let the caller decide what to do.
			return nil, nil
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
	// from the desired origin has been promoted yet. Return nil and let the
	// caller decide what to do.
	return nil, nil
}
