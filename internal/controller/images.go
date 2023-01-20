package controller

import (
	"context"

	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/kube"
	argoLog "github.com/argoproj-labs/argocd-image-updater/pkg/log"
	"github.com/argoproj-labs/argocd-image-updater/pkg/options"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry"
	"github.com/argoproj-labs/argocd-image-updater/pkg/tag"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func init() {
	err := argoLog.SetLogLevel("ERROR")
	if err != nil {
		log.Fatal(err)
	}
}

func (e *environmentReconciler) getLatestImages(
	ctx context.Context,
	env *api.Environment,
) ([]api.Image, error) {
	if env.Spec.Subscriptions == nil ||
		env.Spec.Subscriptions.Repos == nil ||
		len(env.Spec.Subscriptions.Repos.Images) == 0 {
		return nil, nil
	}

	logger := e.logger.WithFields(log.Fields{
		"environment": env.Name,
		"namespace":   env.Namespace,
	})

	images := make([]api.Image, len(env.Spec.Subscriptions.Repos.Images))

	for i, sub := range env.Spec.Subscriptions.Repos.Images {
		imgLogger := logger.WithFields(log.Fields{
			"image": sub.RepoURL,
		})

		img := image.NewFromIdentifier(sub.RepoURL)

		vc := &image.VersionConstraint{
			Constraint: sub.SemverConstraint,
			Strategy:   img.ParseUpdateStrategy(sub.UpdateStrategy),
		}
		if sub.AllowTags != "" {
			vc.MatchFunc, vc.MatchArgs = img.ParseMatchfunc(sub.AllowTags)
		}
		vc.IgnoreList = sub.IgnoreTags
		vc.Options = options.NewManifestOptions()
		if sub.Platform != "" {
			os, arch, variant, err := image.ParsePlatform(sub.Platform)
			if err != nil {
				return nil, errors.Wrapf(
					err,
					"error parsing platform %q for image %q",
					sub.Platform,
					sub.RepoURL,
				)
			}
			vc.Options = vc.Options.WithPlatform(os, arch, variant)
		}
		vc.Options = vc.Options.WithMetadata(vc.Strategy.NeedsMetadata())

		rep, err := registry.GetRegistryEndpoint(img.RegistryURL)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting container registry endpoint for image %q",
				sub.RepoURL,
			)
		}
		imgLogger.Debug("acquired registry endpoint")

		creds, err := e.getImageRepoCredentialsFn(ctx, env.Namespace, sub, rep)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error getting credentials for image %q",
				sub.RepoURL,
			)
		}
		imgLogger.Debug("acquired credentials for registry/repository")

		regClient, err := registry.NewClient(rep, creds.Username, creds.Password)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error creating registry client for image %q",
				sub.RepoURL,
			)
		}
		imgLogger.Debug("created registry client")

		tags, err := e.getImageTagsFn(rep, img, regClient, vc)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error fetching tags for image %q",
				sub.RepoURL,
			)
		}
		imgLogger.Debug("fetched tags for image")

		upImg, err := e.getNewestImageTagFn(img, vc, tags)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error finding newest tag for %q",
				sub.RepoURL,
			)
		}
		if upImg != nil {
			imgLogger.WithField("tag", upImg.TagName).
				Debug("found tag for latest suitable image version")
		} else {
			imgLogger.Error("found no suitable image version")
			return nil, errors.Errorf(
				"found no suitable version of image %q",
				sub.RepoURL,
			)
		}

		images[i] = api.Image{
			RepoURL: sub.RepoURL,
			Tag:     upImg.TagName,
		}
	}

	return images, nil
}

func (e *environmentReconciler) getImageRepoCredentials(
	ctx context.Context,
	namespace string,
	sub api.ImageSubscription,
	rep *registry.RegistryEndpoint,
) (image.Credential, error) {
	creds := image.Credential{}

	kc := kube.NewKubernetesClient(ctx, e.kubeClient, nil, "")

	if err := rep.SetEndpointCredentials(kc); err != nil {
		return creds, errors.Wrapf(
			err,
			"error setting registry credentials for image %q",
			sub.RepoURL,
		)
	}

	if sub.PullSecret != "" {
		credSrc := image.CredentialSource{
			Type:            image.CredentialSourcePullSecret,
			SecretNamespace: namespace,
			SecretName:      sub.PullSecret,
			SecretField:     ".dockerconfigjson",
		}
		credsPtr, err := credSrc.FetchCredentials(rep.RegistryAPI, kc)
		if err != nil {
			return creds, errors.Wrapf(
				err,
				"error fetching credentials for image %q",
				sub.RepoURL,
			)
		}
		creds = *credsPtr
	}

	return creds, nil
}

func getImageTags(
	rep *registry.RegistryEndpoint,
	img *image.ContainerImage,
	regClient registry.RegistryClient,
	vc *image.VersionConstraint,
) (*tag.ImageTagList, error) {
	return rep.GetTags(img, regClient, vc)
}

func getNewestImageTag(
	img *image.ContainerImage,
	vc *image.VersionConstraint,
	tags *tag.ImageTagList,
) (*tag.ImageTag, error) {
	return img.GetNewestVersionFromTags(vc, tags)
}
