package warehouses

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-cd/v2/util/git"
	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/images"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) getLatestImages(
	ctx context.Context,
	namespace string,
	subs []kargoapi.RepoSubscription,
) ([]kargoapi.Image, error) {
	imgs := make([]kargoapi.Image, len(subs))
	for i, s := range subs {
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
		var regCreds *images.Credentials
		if ok {
			regCreds = &images.Credentials{
				Username: creds.Username,
				Password: creds.Password,
			}
			logger.Debug("obtained credentials for image repo")
		} else {
			logger.Debug("found no credentials for image repo")
		}

		tag, err := r.getLatestTagFn(
			sub.RepoURL,
			sub.UpdateStrategy,
			sub.SemverConstraint,
			sub.AllowTags,
			sub.IgnoreTags,
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
		imgs[i] = kargoapi.Image{
			RepoURL:    sub.RepoURL,
			GitRepoURL: r.getImageSourceURL(sub.GitRepoURL, tag),
			Tag:        tag,
		}
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
