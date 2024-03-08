package warehouses

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
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
			return nil, errors.Wrapf(
				err,
				"error obtaining credentials for image repo %q",
				sub.RepoURL,
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

		tag, digest, err := r.getImageRefsFn(
			ctx,
			sub.RepoURL,
			sub.ImageSelectionStrategy,
			sub.SemverConstraint,
			sub.AllowTags,
			sub.IgnoreTags, // TODO: KR: Fix this
			sub.Platform,
			sub.InsecureSkipTLSVerify,
			regCreds,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting latest suitable image %q",
				sub.RepoURL,
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
	return fmt.Sprintf("%s/tree/%s", git.NormalizeGitURL(gitRepoURL), tag)
}

func getImageRefs(
	ctx context.Context,
	repoURL string,
	imageSelectionStrategy kargoapi.ImageSelectionStrategy,
	constraint string,
	allowTagsRegex string,
	ignoreTags []string,
	platform string,
	insecureSkipVerify bool,
	creds *image.Credentials,
) (string, string, error) {
	imageSelector, err := image.NewSelector(
		repoURL,
		image.SelectionStrategy(imageSelectionStrategy),
		&image.SelectorOptions{
			Constraint:         constraint,
			AllowRegex:         allowTagsRegex,
			Ignore:             ignoreTags,
			Platform:           platform,
			Creds:              creds,
			InsecureSkipVerify: insecureSkipVerify,
		},
	)
	if err != nil {
		return "", "", errors.Wrapf(
			err,
			"error creating image selector for image %q",
			repoURL,
		)
	}
	img, err := imageSelector.Select(ctx)
	if err != nil {
		return "", "", errors.Wrapf(
			err,
			"error fetching newest applicable image %q",
			repoURL,
		)
	}
	if img == nil {
		return "", "", errors.Errorf("found no applicable image %q", repoURL)
	}
	return img.Tag, img.Digest.String(), nil
}
