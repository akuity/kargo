package image

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/patrickmn/go-cache"
	"go.uber.org/ratelimit"
	"golang.org/x/sync/semaphore"

	"github.com/akuity/kargo/internal/logging"
)

const (
	// maxMetadataConcurrency is the maximum number of concurrent goroutines that
	// can be used to fetch metadata. Per Go's documentation, goroutines are very
	// cheap (practical to spawn tens of thousands), AND we have a rate limiter in
	// place for each registry, so there's no reason for this number not to be
	// pretty large.
	maxMetadataConcurrency = 1000

	unknown = "unknown"
)

var metaSem = semaphore.NewWeighted(maxMetadataConcurrency)

// repositoryClient is a client for retrieving information from a specific image
// container repository.
type repositoryClient struct {
	registry      *registry
	repoURL       string
	repoRef       name.Reference
	remoteOptions []remote.Option

	// The following behaviors are overridable for testing purposes:

	getImageByTagFn func(
		context.Context,
		string,
		*platformConstraint,
	) (*Image, error)

	getImageByDigestFn func(
		context.Context,
		string,
		*platformConstraint,
	) (*Image, error)

	getImageFromRemoteDescFn func(
		context.Context,
		*remote.Descriptor,
		*platformConstraint,
	) (*Image, error)

	getImageFromV1ImageIndexFn func(
		ctx context.Context,
		digest string,
		idx v1.ImageIndex,
		platform *platformConstraint,
	) (*Image, error)

	getImageFromV1ImageFn func(
		digest string,
		img v1.Image,
		platform *platformConstraint,
	) (*Image, error)

	remoteListFn func(name.Repository, ...remote.Option) ([]string, error)

	remoteGetFn func(name.Reference, ...remote.Option) (*remote.Descriptor, error)
}

// newRepositoryClient parses the provided repository URL to infer registry
// information and image name. This information is used to initialize and
// return a new repository client.
func newRepositoryClient(
	repoURL string,
	insecureSkipTLSVerify bool,
	creds *Credentials,
) (*repositoryClient, error) {
	repoRef, err := name.ParseReference(repoURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing image repo URL %s: %w", repoURL, err)
	}
	reg := getRegistry(repoRef.Context().RegistryStr())

	httpTransport := cleanhttp.DefaultTransport()
	if insecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: insecureSkipTLSVerify, // nolint: gosec
		}
	}

	if creds == nil {
		creds = &Credentials{}
	}
	var auth authn.Authenticator = &authn.Basic{
		Username: creds.Username,
		Password: creds.Password,
	}

	r := &repositoryClient{
		registry: reg,
		repoURL:  repoURL,
		repoRef:  repoRef,
		remoteOptions: []remote.Option{
			remote.WithTransport(&rateLimitedRoundTripper{
				limiter:              reg.rateLimiter,
				internalRoundTripper: httpTransport,
			}),
			remote.WithAuth(auth),
		},
	}

	r.getImageByTagFn = r.getImageByTag
	r.getImageByDigestFn = r.getImageByDigest
	r.getImageFromRemoteDescFn = r.getImageFromRemoteDesc
	r.getImageFromV1ImageIndexFn = r.getImageFromV1ImageIndex
	r.getImageFromV1ImageFn = r.getImageFromV1Image
	r.remoteListFn = remote.List
	r.remoteGetFn = remote.Get

	return r, nil
}

func (r *repositoryClient) getTags(ctx context.Context) ([]string, error) {
	opts := append(r.remoteOptions, remote.WithContext(ctx))
	tags, err := r.remoteListFn(r.repoRef.Context(), opts...)
	if err != nil {
		return nil, fmt.Errorf("error listing tags for repo URL %s: %w", r.repoURL, err)
	}
	return tags, nil
}

// getImageByTag retrieves an Image by tag. This function uses no cache since
// tags can be mutable.
func (r *repositoryClient) getImageByTag(
	ctx context.Context,
	tag string,
	platform *platformConstraint,
) (*Image, error) {
	repoRef := r.repoRef.Context().Tag(tag)
	opts := append(r.remoteOptions, remote.WithContext(ctx))
	desc, err := r.remoteGetFn(repoRef, opts...)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting image descriptor for tag %q from repo URL %s: %w",
			tag, r.repoURL, err,
		)
	}
	img, err := r.getImageFromRemoteDescFn(ctx, desc, platform)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting image from descriptor for tag %q from repo URL %s: %w",
			tag, r.repoURL, err,
		)
	}
	if img != nil {
		img.Tag = tag
	}
	return img, nil
}

// getImageByDigest retrieves an Image for a given digest. This function uses a
// cache since information retrieved by digest will never change.
func (r *repositoryClient) getImageByDigest(
	ctx context.Context,
	digest string,
	platform *platformConstraint,
) (*Image, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Trace(
		"retrieving image",
		"digest", digest,
	)

	if entry, exists := r.registry.imageCache.Get(digest); exists {
		image := entry.(Image) // nolint: forcetypeassert
		return &image, nil
	}

	logger.Trace(
		"image NOT found in cache",
		"digest", digest,
	)

	repoRef := r.repoRef.Context().Digest(digest)
	opts := append(r.remoteOptions, remote.WithContext(ctx))
	desc, err := r.remoteGetFn(repoRef, opts...)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting image descriptor for digest %s from repo URL %s: %w",
			digest, r.repoURL, err,
		)
	}

	img, err := r.getImageFromRemoteDescFn(ctx, desc, platform)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting image from descriptor for digest %s from repo URL %s: %w",
			digest, r.repoURL, err,
		)
	}

	if img != nil {
		// Cache the image
		r.registry.imageCache.Set(digest, *img, cache.DefaultExpiration)
		logger.Trace(
			"cached image",
			"digest", digest,
		)
	}

	return img, nil
}

// getImageFromRemoteDesc gets an Image from a given remote.Descriptor.
func (r *repositoryClient) getImageFromRemoteDesc(
	ctx context.Context,
	desc *remote.Descriptor,
	platform *platformConstraint,
) (*Image, error) {
	switch desc.MediaType {
	case types.OCIImageIndex, types.DockerManifestList:
		idx, err := desc.ImageIndex()
		if err != nil {
			return nil, fmt.Errorf(
				"error getting image index from descriptor with digest %s: %w",
				desc.Digest.String(), err,
			)
		}

		img, err := r.getImageFromV1ImageIndexFn(ctx, desc.Digest.String(), idx, platform)
		if err != nil {
			return nil, err
		}

		// If the descriptor has annotations, merge them into the annotations
		// collected from the index and manifest. The descriptor's annotations
		// will have a lower precedence than any annotations collected for the
		// image.
		if img != nil && desc.Annotations != nil {
			baseAnnotations := desc.Annotations
			if img.Annotations != nil {
				maps.Copy(baseAnnotations, img.Annotations)
			}
			img.Annotations = baseAnnotations
		}

		return img, nil
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		img, err := desc.Image()
		if err != nil {
			return nil, fmt.Errorf(
				"error getting image from descriptor with digest %s: %w",
				desc.Digest.String(), err,
			)
		}

		finalImg, err := r.getImageFromV1ImageFn(desc.Digest.String(), img, platform)
		if err != nil {
			return nil, err
		}

		// If the descriptor has annotations, merge them into the annotations
		// collected from the index and manifest. The descriptor's annotations
		// will have a lower precedence than any annotations collected for the
		// image.
		if finalImg != nil && desc.Annotations != nil {
			baseAnnotations := desc.Annotations
			if finalImg.Annotations != nil {
				maps.Copy(baseAnnotations, finalImg.Annotations)
			}
			finalImg.Annotations = baseAnnotations
		}
		return finalImg, nil
	default:
		return nil, fmt.Errorf("unknown artifact type: %s", desc.MediaType)
	}
}

// getImageFromV1ImageIndex gets an Image from a given v1.ImageIndex. It is
// valid for this function to return nil if no image in the index matches the
// specified platform, if any.
func (r *repositoryClient) getImageFromV1ImageIndex(
	ctx context.Context,
	digest string,
	idx v1.ImageIndex,
	platform *platformConstraint,
) (*Image, error) {
	idxManifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf(
			"error getting index manifest from index with digest %s: %w",
			digest, err,
		)
	}

	// Extract annotations from the index manifest.
	annotations := idxManifest.Annotations

	refs := make([]v1.Descriptor, 0, len(idxManifest.Manifests))
	for _, ref := range idxManifest.Manifests {
		if ref.Platform == nil ||
			ref.Platform.OS == unknown || ref.Platform.OS == "" ||
			ref.Platform.Architecture == unknown || ref.Platform.Architecture == "" {
			// This reference doesn't look like a reference to an image. It might
			// be an attestation or something else. Skip it.
			continue
		}
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		return nil, errors.New("empty V2 manifest list or OCI index is not supported")
	}
	// If there's a platform constraint, find the ref that matches it and
	// that's the information we're really after.
	if platform != nil {
		var matchedRefs []v1.Descriptor
		for _, ref := range refs {
			if !platform.matches(ref.Platform.OS, ref.Platform.Architecture, ref.Platform.Variant) {
				continue
			}
			matchedRefs = append(matchedRefs, ref)
		}

		if len(matchedRefs) == 0 {
			// No refs matched the platform
			return nil, nil
		}

		if len(matchedRefs) > 1 {
			// This really shouldn't happen.
			return nil, fmt.Errorf(
				"expected only one reference to match platform %q, but found %d",
				platform.String(),
				len(matchedRefs),
			)
		}

		ref := matchedRefs[0]

		img, err := r.getImageByDigestFn(ctx, ref.Digest.String(), platform)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting image with digest %s: %w",
				ref.Digest.String(),
				err,
			)
		}
		if img == nil {
			// This really shouldn't happen.
			return nil, fmt.Errorf(
				"expected manifest for digest %s to match platform %q, but it did not",
				ref.Digest.String(),
				platform.String(),
			)
		}
		img.Digest = digest
		img.Annotations = annotations

		return img, nil
	}

	// If we get to here there was no platform constraint.

	// Manifest lists and indices don't have a createdAt timestamp, and we had no
	// platform constraint, so we'll follow ALL the references to find the most
	// recently pushed manifest's createdAt timestamp.
	var createdAt *time.Time
	for _, ref := range refs {
		img, err := r.getImageByDigestFn(ctx, ref.Digest.String(), platform)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting image with digest %s: %w", ref.Digest, err,
			)
		}
		if img == nil {
			// This really shouldn't happen.
			return nil, fmt.Errorf("found no image with digest %s", ref.Digest)
		}
		if createdAt == nil || img.CreatedAt.After(*createdAt) {
			createdAt = img.CreatedAt
		}

		// TODO(hidde): Without a platform constraint, we can not collect
		// annotations in a meaningful way. We should consider how to handle
		// this in the future.
	}

	return &Image{
		Digest:      digest,
		CreatedAt:   createdAt,
		Annotations: annotations,
	}, nil
}

// getImageFromV1Image gets an Image from a given v1.Image. It is valid for this
// function to return nil the image does not match the specified platform, if
// any.
func (r *repositoryClient) getImageFromV1Image(
	digest string,
	img v1.Image,
	platform *platformConstraint,
) (*Image, error) {
	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf(
			"error getting image config for image with digest %s: %w",
			digest, err,
		)
	}
	if platform != nil && !platform.matches(cfg.OS, cfg.Architecture, cfg.Variant) {
		// This image doesn't match the platform constraint.
		return nil, nil
	}

	// Extract annotations from the manifest
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf(
			"error getting manifest for image with digest %s: %w",
			digest, err,
		)
	}

	return &Image{
		Digest:      digest,
		CreatedAt:   &cfg.Created.Time,
		Annotations: manifest.Annotations,
	}, nil
}

// rateLimitedRoundTripper is a rate limited implementation of
// http.RoundTripper.
type rateLimitedRoundTripper struct {
	limiter              ratelimit.Limiter
	internalRoundTripper http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (r *rateLimitedRoundTripper) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	r.limiter.Take()
	return r.internalRoundTripper.RoundTrip(req)
}
