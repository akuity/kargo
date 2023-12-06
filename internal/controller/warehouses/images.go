package warehouses

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) getLatestImages(
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
			sub.TagSelectionStrategy,
			sub.SemverConstraint,
			sub.AllowTags,
			sub.IgnoreTags, // TODO: KR: Fix this
			sub.Platform,
			regCreds,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting latest suitable tag for image %q",
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
		logger.WithField("tag", tag).
			Debug("found latest suitable image tag")
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
	tagSelectionStrategy kargoapi.ImageTagSelectionStrategy,
	constraint string,
	allowTagsRegex string,
	ignoreTags []string,
	platform string,
	creds *image.Credentials,
) (string, string, error) {
	tc, err := image.NewTagSelector(
		repoURL,
		image.TagSelectionStrategy(tagSelectionStrategy),
		&image.TagSelectorOptions{
			Constraint: constraint,
			AllowRegex: allowTagsRegex,
			Ignore:     ignoreTags,
			Platform:   platform,
			Creds:      creds,
		},
	)
	if err != nil {
		return "", "", errors.Wrapf(
			err,
			"error creating tag constraint for image %q",
			repoURL,
		)
	}
	tag, err := tc.SelectTag(ctx)
	if err != nil {
		return "", "", errors.Wrapf(
			err,
			"error fetching newest applicable tag for image %q",
			repoURL,
		)
	}
	if tag == nil {
		return "", "", errors.Errorf("found no applicable tags for image %q", repoURL)
	}
	return tag.Name, tag.Digest.String(), nil
}
