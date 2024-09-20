package argocd

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
)

// GetDesiredRevision returns the desired revisions for all sources of the given
// Application. For a single-source Application, the returned slice will have
// precisely one value. For a multi-source Application, the returned slice will
// have the same length as and will be indexed identically to the Application's
// Sources slice. For any source whose desired revision cannot be determined,
// the slice will contain an empty string at the corresponding index.
func GetDesiredRevisions(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	update *kargoapi.ArgoCDAppUpdate,
	app *argocd.Application,
	frght []kargoapi.FreightReference,
) ([]string, error) {
	// Note that frght was provided as an argument instead of being plucked
	// directly from stage.Status, because this gives us the flexibility to use
	// this function for finding the revision to sync to either in the context of
	// a health check (current freight) or in the context of a promotion (new
	// freight).
	if app == nil {
		return nil, nil
	}
	sources := app.Spec.Sources
	if len(sources) == 0 && app.Spec.Source != nil {
		sources = []argocd.ApplicationSource{*app.Spec.Source}
	}
	if len(sources) == 0 {
		return nil, nil
	}
	revisions := make([]string, len(sources))
	for i, src := range sources {
		var desiredOrigin *kargoapi.FreightOrigin
		// If there is a source update that targets this source, it might be
		// specific about which origin the desired revision should come from.
		sourceUpdate := findSourceUpdate(update, src)
		if sourceUpdate != nil {
			desiredOrigin = freight.GetDesiredOrigin(stage, sourceUpdate)
		} else {
			desiredOrigin = freight.GetDesiredOrigin(stage, update)
		}
		desiredRevision, err := getDesiredRevisionForSource(
			ctx,
			cl,
			stage,
			&src,
			desiredOrigin,
			frght,
		)
		if err != nil {
			return nil, err
		}
		revisions[i] = desiredRevision
	}
	return revisions, nil
}

func getDesiredRevisionForSource(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
	src *argocd.ApplicationSource,
	desiredOrigin *kargoapi.FreightOrigin,
	frght []kargoapi.FreightReference,
) (string, error) {
	switch {
	case src.Chart != "":
		// This source points to a Helm chart.
		repoURL := src.RepoURL
		chartName := src.Chart
		if !strings.Contains(repoURL, "://") {
			// In Argo CD ApplicationSource, if a repo URL specifies no protocol and a
			// chart name is set (already confirmed at this point), we can assume that
			// the repo URL is an OCI registry URL. Kargo Warehouses and Freight,
			// however, do use oci:// at the beginning of such URLs.
			//
			// Additionally, where OCI is concerned, an ApplicationSource's repoURL is
			// really a registry URL, and the chart name is a repository within that
			// registry. Warehouses and Freight, however, handle things more correctly
			// where a repoURL points directly to a repository and chart name is
			// irrelevant / blank. We need to account for this when we search our
			// Freight for the chart.
			repoURL = fmt.Sprintf(
				"oci://%s/%s",
				strings.TrimSuffix(repoURL, "/"),
				chartName,
			)
			chartName = ""
		}
		chart, err := freight.FindChart(
			ctx,
			cl,
			stage.Namespace,
			stage.Spec.RequestedFreight,
			desiredOrigin,
			frght,
			repoURL,
			chartName,
		)
		if err != nil {
			return "",
				fmt.Errorf("error finding chart from repo %q: %w", repoURL, err)
		}
		if chart == nil {
			return "", nil
		}
		return chart.Version, nil
	case src.RepoURL != "":
		// This source points to a Git repository.
		commit, err := freight.FindCommit(
			ctx,
			cl,
			stage.Namespace,
			stage.Spec.RequestedFreight,
			desiredOrigin,
			frght,
			src.RepoURL,
		)
		if err != nil {
			return "",
				fmt.Errorf("error finding commit from repo %q: %w", src.RepoURL, err)
		}
		if commit == nil {
			return "", nil
		}
		if commit.HealthCheckCommit != "" {
			return commit.HealthCheckCommit, nil
		}
		return commit.ID, nil
	}
	// If we end up here, no desired revision was found.
	return "", nil
}

// findSourceUpdate finds and returns the ArgoCDSourceUpdate that targets the
// given source. If no such update exists, it returns nil.
func findSourceUpdate(
	update *kargoapi.ArgoCDAppUpdate,
	src argocd.ApplicationSource,
) *kargoapi.ArgoCDSourceUpdate {
	for i := range update.SourceUpdates {
		sourceUpdate := &update.SourceUpdates[i]
		if sourceUpdate.RepoURL == src.RepoURL && sourceUpdate.Chart == src.Chart {
			return sourceUpdate
		}
	}
	return nil
}
