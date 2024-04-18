package warehouses

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) selectImages(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.Image, error) {
	imgs := make([]kargoapi.Image, 0, len(subs))
	for _, s := range subs {
		if s.Image == nil {
			continue
		}

		sub := s.Image

		logger := logging.LoggerFromContext(ctx).WithField("repo", sub.RepoURL)

		creds, ok, err :=
			r.credentialsDB.Get(ctx, namespace, credentials.TypeImage, sub.RepoURL)
		if err != nil {
			return nil, fmt.Errorf(
				"error obtaining credentials for image repo %q: %w",
				sub.RepoURL,
				err,
			)
		}
		var regCreds *image.Credentials
		if ok {
			regCreds = &image.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}
			logger.Debug("obtained credentials for image repo")
		} else {
			logger.Debug("found no credentials for image repo")
		}

		tag, digest, err := r.getImageRefsFn(ctx, *sub, regCreds)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting latest suitable image %q: %w",
				sub.RepoURL,
				err,
			)
		}
		imgs = append(
			imgs,
			kargoapi.Image{
				RepoURL:    sub.RepoURL,
				GitRepoURL: r.getImageSourceURL(sub.GitRepoURL, tag),
				Tag:        tag,
				Digest:     digest,
			},
		)
		logger.WithFields(log.Fields{
			"tag":    tag,
			"digest": digest,
		}).Debug("found latest suitable image")
	}
	return imgs, nil
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

func getGithubImageSourceURL(gitRepoURL, tag string) string {
	return fmt.Sprintf("%s/tree/%s", git.NormalizeURL(gitRepoURL), tag)
}

func getImageRefs(
	ctx context.Context,
	sub kargoapi.ImageSubscription,
	creds *image.Credentials,
) (string, string, error) {
	imageSelector, err := image.NewSelector(
		sub.RepoURL,
		image.SelectionStrategy(sub.ImageSelectionStrategy),
		&image.SelectorOptions{
			Constraint:            sub.SemverConstraint,
			AllowRegex:            sub.AllowTags,
			Ignore:                sub.IgnoreTags,
			Platform:              sub.Platform,
			Creds:                 creds,
			InsecureSkipTLSVerify: sub.InsecureSkipTLSVerify,
		},
	)
	if err != nil {
		return "", "", fmt.Errorf(
			"error creating image selector for image %q: %w",
			sub.RepoURL,
			err,
		)
	}
	img, err := imageSelector.Select(ctx)
	if err != nil {
		return "", "", fmt.Errorf(
			"error fetching newest applicable image %q: %w",
			sub.RepoURL,
			err,
		)
	}
	if img == nil {
		return "", "", fmt.Errorf("found no applicable image %q", sub.RepoURL)
	}
	return img.Tag, img.Digest.String(), nil
}
