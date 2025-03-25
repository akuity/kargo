package warehouses

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/logging"
)

// discoverImages discovers the latest suitable images for the given image
// subscriptions. It returns a list of image discovery results, one for each
// subscription.
func (r *reconciler) discoverImages(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.ImageDiscoveryResult, error) {
	results := make([]kargoapi.ImageDiscoveryResult, 0, len(subs))

	for _, s := range subs {
		if s.Image == nil {
			continue
		}
		sub := *s.Image

		logger := logging.LoggerFromContext(ctx).WithValues("repo", sub.RepoURL)

		// Obtain credentials for the image repository.
		creds, err := r.credentialsDB.Get(ctx, namespace, credentials.TypeImage, sub.RepoURL)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining credentials for image repo %q: %w",
				sub.RepoURL,
				err,
			)
		}
		var regCreds *image.Credentials
		if creds != nil {
			regCreds = &image.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}
			logger.Debug("obtained credentials for image repo")
		} else {
			logger.Debug("found no credentials for image repo")
		}

		// Enrich the logger with additional fields for this subscription.
		logger = logger.WithValues(imageDiscoveryLogFields(sub))

		// Discover the latest suitable images.
		images, err := r.discoverImageRefsFn(ctx, sub, regCreds)
		if err != nil {
			return nil, fmt.Errorf(
				"error discovering latest images %q: %w",
				sub.RepoURL,
				err,
			)
		}
		if len(images) == 0 {
			results = append(results, kargoapi.ImageDiscoveryResult{
				RepoURL:  sub.RepoURL,
				Platform: sub.Platform,
			})
			logger.Debug("discovered no images")
			continue
		}

		discoveredImages := make([]kargoapi.DiscoveredImageReference, 0, len(images))
		for _, img := range images {
			discovery := kargoapi.DiscoveredImageReference{
				Tag:         img.Tag,
				Digest:      img.Digest,
				Annotations: img.Annotations,
				GitRepoURL:  r.getImageSourceURL(sub.GitRepoURL, img.Tag),
			}
			if img.CreatedAt != nil {
				discovery.CreatedAt = &metav1.Time{Time: *img.CreatedAt}
			}
			discoveredImages = append(discoveredImages, discovery)
		}

		results = append(results, kargoapi.ImageDiscoveryResult{
			RepoURL:    sub.RepoURL,
			Platform:   sub.Platform,
			References: discoveredImages,
		})
		logger.Debug(
			"discovered images",
			"count", len(images),
		)
	}

	return results, nil
}

func (r *reconciler) discoverImageRefs(
	ctx context.Context,
	sub kargoapi.ImageSubscription,
	creds *image.Credentials,
) ([]image.Image, error) {
	imageSelector, err := imageSelectorForSubscription(sub, creds)
	if err != nil {
		return nil, fmt.Errorf(
			"error creating image selector for image %q: %w",
			sub.RepoURL,
			err,
		)
	}

	images, err := imageSelector.Select(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"error discovering newest applicable images %q: %w",
			sub.RepoURL,
			err,
		)
	}
	return images, nil
}

const (
	githubURLPrefix = "https://github.com"
)

func (r *reconciler) getImageSourceURL(gitRepoURL, tag string) string {
	for baseUrl, fn := range r.imageSourceURLFnsByBaseURL {
		if strings.HasPrefix(gitRepoURL, baseUrl) {
			return fn(gitRepoURL, tag)
		}
	}
	return ""
}

func imageDiscoveryLogFields(sub kargoapi.ImageSubscription) []any {
	f := []any{
		"imageSelectionStrategy", sub.ImageSelectionStrategy,
		"platformConstrained", sub.Platform != "",
	}
	switch sub.ImageSelectionStrategy {
	case kargoapi.ImageSelectionStrategySemVer, kargoapi.ImageSelectionStrategyDigest:
		f = append(
			f,
			"semverConstraint", sub.SemverConstraint,
		)
	case kargoapi.ImageSelectionStrategyLexical, kargoapi.ImageSelectionStrategyNewestBuild:
		f = append(
			f,
			"tagConstrained", sub.AllowTags != "" || len(sub.IgnoreTags) > 0,
		)
	}
	return f
}

func imageSelectorForSubscription(
	sub kargoapi.ImageSubscription,
	creds *image.Credentials,
) (image.Selector, error) {
	return image.NewSelector(
		sub.RepoURL,
		image.SelectionStrategy(sub.ImageSelectionStrategy),
		&image.SelectorOptions{
			StrictSemvers:         sub.StrictSemvers,
			Constraint:            sub.SemverConstraint,
			AllowRegex:            sub.AllowTags,
			Ignore:                sub.IgnoreTags,
			Platform:              sub.Platform,
			Creds:                 creds,
			InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
			DiscoveryLimit:        int(sub.DiscoveryLimit),
		},
	)
}

func getGithubImageSourceURL(gitRepoURL, tag string) string {
	return fmt.Sprintf("%s/tree/%s", git.NormalizeURL(gitRepoURL), tag)
}
