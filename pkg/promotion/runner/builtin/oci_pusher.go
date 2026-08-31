package builtin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/kelseyhightower/envconfig"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	libfmt "github.com/akuity/kargo/pkg/fmt"
	"github.com/akuity/kargo/pkg/image/mutate"
	intio "github.com/akuity/kargo/pkg/io"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	builtin "github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindOCIPush = "oci-push"

// ociPusherConfig holds environment-based configuration for the oci-push step
// runner. A value of -1 for MaxArtifactSize disables the size limit entirely;
// a value of 0 disables all pushes that transfer blobs.
type ociPusherConfig struct {
	MaxArtifactSize int64 `envconfig:"MAX_OCI_PUSH_ARTIFACT_SIZE" default:"1073741824"` // 1 GiB
}

func init() {
	cfg := ociPusherConfig{}
	envconfig.MustProcess("", &cfg)
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindOCIPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: func(
				caps promotion.StepRunnerCapabilities,
			) promotion.StepRunner {
				return newOCIPusher(caps, cfg)
			},
		},
	)
}

// ociPusher is an implementation of the promotion.StepRunner interface that
// pushes OCI artifacts to a registry. It can copy/retag artifacts (container
// images and Helm charts) between registries, or push a local file from the
// workspace as a single-layer OCI artifact.
type ociPusher struct {
	schemaLoader    gojsonschema.JSONLoader
	credsDB         credentials.Database
	maxArtifactSize int64 // maximum compressed artifact size in bytes
}

// newOCIPusher returns an implementation of the promotion.StepRunner interface
// that pushes OCI artifacts to a registry. It uses the provided credentials
// database to authenticate with source and destination registries.
func newOCIPusher(
	caps promotion.StepRunnerCapabilities,
	cfg ociPusherConfig,
) promotion.StepRunner {
	return &ociPusher{
		credsDB:         caps.CredsDB,
		schemaLoader:    getConfigSchemaLoader(stepKindOCIPush),
		maxArtifactSize: cfg.MaxArtifactSize,
	}
}

// Run implements the promotion.StepRunner interface.
func (p *ociPusher) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := p.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return p.run(ctx, stepCtx, cfg)
}

// convert validates the ociPusher configuration against a JSON schema and
// converts it into a builtin.OCIPushConfig struct.
func (p *ociPusher) convert(cfg promotion.Config) (builtin.OCIPushConfig, error) {
	return validateAndConvert[builtin.OCIPushConfig](p.schemaLoader, cfg, stepKindOCIPush)
}

// run executes the ociPusher step with the provided configuration.
func (p *ociPusher) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIPushConfig,
) (promotion.StepResult, error) {
	dstRef, dstCredType, err := parseOCIReference(cfg.DestRef)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("failed to parse destination reference %q: %w", cfg.DestRef, err),
			}
	}

	// Validate the source reference before any credential lookups so a
	// malformed reference fails terminally instead of being masked by a
	// retryable credential error.
	var (
		srcRef      name.Reference
		srcCredType credentials.Type
	)
	if cfg.SrcPath == "" {
		if srcRef, srcCredType, err = parseOCIReference(cfg.SrcRef); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
				&promotion.TerminalError{
					Err: fmt.Errorf("failed to parse source reference %q: %w", cfg.SrcRef, err),
				}
		}
	}

	dstOpts, err := buildOCIRemoteOptions(
		ctx, p.credsDB, stepCtx.Project, dstRef, dstCredType, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	var (
		digest v1.Hash
		status kargoapi.PromotionStepStatus
	)
	if cfg.SrcPath != "" {
		digest, status, err = p.pushLocalFile(stepCtx, cfg, dstRef, dstOpts)
	} else {
		digest, status, err = p.pushRemoteArtifact(
			ctx, stepCtx, cfg, srcRef, srcCredType, dstRef, dstOpts,
		)
	}
	if err != nil {
		return promotion.StepResult{Status: status}, err
	}

	// Extract tag from destination reference if available.
	var tag string
	if t, ok := dstRef.(name.Tag); ok {
		tag = t.TagStr()
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"image":  dstRef.String(),
			"digest": digest.String(),
			"tag":    tag,
		},
	}, nil
}

// pushRemoteArtifact copies/retags the artifact identified by srcRef to the
// destination reference. It returns the digest of the pushed artifact and, on
// failure, the promotion step status to report alongside the error.
func (p *ociPusher) pushRemoteArtifact(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIPushConfig,
	srcRef name.Reference,
	srcCredType credentials.Type,
	dstRef name.Reference,
	dstOpts []remote.Option,
) (v1.Hash, kargoapi.PromotionStepStatus, error) {
	srcOpts, err := buildOCIRemoteOptions(
		ctx, p.credsDB, stepCtx.Project, srcRef, srcCredType, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored, err
	}

	desc, err := remote.Get(srcRef, srcOpts...)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to get source artifact %q: %w", cfg.SrcRef, err)
	}

	digest, err := p.push(desc, srcRef, dstRef, cfg.Annotations, dstOpts)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored, err
	}
	return digest, kargoapi.PromotionStepStatusSucceeded, nil
}

const (
	// defaultLayerMediaType is the media type applied to the artifact layer when
	// pushing a local file via srcPath and no media type is configured.
	defaultLayerMediaType = types.OCILayer
	// emptyConfigMediaType is the media type of the OCI "empty" descriptor. An
	// artifact carries no image config.
	emptyConfigMediaType = types.MediaType("application/vnd.oci.empty.v1+json")
	// emptyConfigBlob is the content of the OCI "empty" descriptor.
	emptyConfigBlob = "{}"
	// defaultArtifactType is declared when no artifact type is configured. The
	// image spec requires an artifactType whenever the config is the empty
	// descriptor; this is the value ORAS uses for an artifact of unknown type.
	defaultArtifactType = "application/vnd.unknown.artifact.v1"
)

// pushLocalFile pushes a local file from the workspace to the destination
// reference as a single-layer OCI artifact -- not an image, so the manifest
// carries no image config and the file's bytes are pushed verbatim. It returns
// the digest of the pushed artifact and, on failure, the promotion step status
// to report alongside the error.
func (p *ociPusher) pushLocalFile(
	stepCtx *promotion.StepContext,
	cfg builtin.OCIPushConfig,
	dstRef name.Reference,
	dstOpts []remote.Option,
) (v1.Hash, kargoapi.PromotionStepStatus, error) {
	root, err := os.OpenRoot(stepCtx.WorkDir)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusFailed, &promotion.TerminalError{
			Err: fmt.Errorf(
				"failed to open workspace: %w",
				fs.SanitizePathError(err, stepCtx.WorkDir),
			),
		}
	}
	defer root.Close()

	f, err := root.Open(cfg.SrcPath)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusFailed, &promotion.TerminalError{
			Err: fmt.Errorf("failed to open source path %q: %w", cfg.SrcPath, err),
		}
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusFailed, &promotion.TerminalError{
			Err: fmt.Errorf("failed to stat source path %q: %w", cfg.SrcPath, err),
		}
	}
	if info.IsDir() {
		return v1.Hash{}, kargoapi.PromotionStepStatusFailed, &promotion.TerminalError{
			Err: fmt.Errorf("source path %q is a directory; expected a file", cfg.SrcPath),
		}
	}

	// A local push always transfers the blob, so enforce the size limit
	// unconditionally, mirroring the cross-repository check on the retag path:
	// zero disables pushing entirely and a negative value disables the check.
	// This stat-based check is only a fast fail; newFileLayer re-enforces the
	// limit on the bytes actually hashed, which catches a file that grows after
	// this check.
	if p.maxArtifactSize == 0 {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored, &promotion.TerminalError{
			Err: fmt.Errorf("local artifact push is disabled"),
		}
	}
	if p.maxArtifactSize > 0 && info.Size() > p.maxArtifactSize {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored, &promotion.TerminalError{
			Err: fmt.Errorf(
				"artifact size %s exceeds maximum allowed size of %s",
				libfmt.FormatByteCount(info.Size()), libfmt.FormatByteCount(p.maxArtifactSize),
			),
		}
	}

	layerMediaType := defaultLayerMediaType
	if cfg.MediaType != "" {
		layerMediaType = types.MediaType(cfg.MediaType)
	}

	// A configured artifact type is mirrored onto the config media type, where
	// Flux and the pre-OCI-1.1 Helm/ORAS convention record it. Absent one, the
	// config is the empty descriptor, which requires an artifactType.
	artifactType := cfg.ArtifactType
	configMediaType := types.MediaType(artifactType)
	if artifactType == "" {
		artifactType = defaultArtifactType
		configMediaType = emptyConfigMediaType
	}

	// A single manifest, so index-scoped keys are ignored, as on the retag path.
	scopes := p.parseAnnotationScopes(cfg.Annotations)

	configLayer := static.NewLayer([]byte(emptyConfigBlob), configMediaType)
	artifactLayer, err := newFileLayer(root, cfg.SrcPath, f, layerMediaType, p.maxArtifactSize)
	if err != nil {
		if _, ok := errors.AsType[*intio.BodyTooLargeError](err); ok {
			return v1.Hash{}, kargoapi.PromotionStepStatusErrored, &promotion.TerminalError{
				Err: fmt.Errorf(
					"artifact size exceeds maximum allowed size of %s",
					libfmt.FormatByteCount(p.maxArtifactSize),
				),
			}
		}
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to read source path %q: %w", cfg.SrcPath, err)
	}

	configDesc, err := partial.Descriptor(configLayer)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to describe artifact config: %w", err)
	}
	artifactDesc, err := partial.Descriptor(artifactLayer)
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to describe artifact layer: %w", err)
	}

	rawManifest, err := json.Marshal(v1.Manifest{
		SchemaVersion: 2,
		MediaType:     types.OCIManifestSchema1,
		ArtifactType:  artifactType,
		Config:        *configDesc,
		Layers:        []v1.Descriptor{*artifactDesc},
		Annotations:   scopes.manifest,
	})
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to build artifact manifest: %w", err)
	}
	digest, _, err := v1.SHA256(bytes.NewReader(rawManifest))
	if err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to compute artifact digest: %w", err)
	}

	// Upload both blobs before the manifest that references them. The blobs are
	// unreachable until then, so the manifest is the commit point: a failure
	// here leaves the destination tag as it was.
	for _, layer := range []v1.Layer{configLayer, artifactLayer} {
		if err = remote.WriteLayer(dstRef.Context(), layer, dstOpts...); err != nil {
			return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
				fmt.Errorf("failed to push artifact blob to %q: %w", dstRef.Context().String(), err)
		}
	}
	if err = remote.Put(
		dstRef,
		&taggableManifest{raw: rawManifest, mediaType: types.OCIManifestSchema1},
		dstOpts...,
	); err != nil {
		return v1.Hash{}, kargoapi.PromotionStepStatusErrored,
			fmt.Errorf("failed to push artifact to %q: %w", dstRef.String(), err)
	}

	return digest, kargoapi.PromotionStepStatusSucceeded, nil
}

// fileLayer is a v1.Layer backed by a file, streamed from disk on each read
// rather than held in memory. Its content is opaque, so the compressed and
// uncompressed forms are identical.
type fileLayer struct {
	open      func() (io.ReadCloser, error)
	mediaType types.MediaType
	digest    v1.Hash
	size      int64
}

// newFileLayer hashes and measures the file's content from f, which must read
// the file at path from its start and remains owned (and closed) by the
// caller; later reads re-open path under root. It reads at most limit bytes
// and returns an *intio.BodyTooLargeError if the content exceeds that.
// Enforcing the limit on the bytes actually hashed closes the window between
// a caller's stat-based pre-check and the read. A negative limit disables the
// check; zero is a real zero-byte limit, so a caller treating zero as
// "disabled" must guard before calling.
func newFileLayer(
	root *os.Root,
	path string,
	f io.Reader,
	mediaType types.MediaType,
	limit int64,
) (*fileLayer, error) {
	if limit < 0 {
		limit = math.MaxInt64
	}
	hasher := sha256.New()
	size, err := intio.LimitCopy(hasher, io.NopCloser(f), limit)
	if err != nil {
		return nil, err
	}
	return &fileLayer{
		open:      func() (io.ReadCloser, error) { return root.Open(path) },
		mediaType: mediaType,
		digest:    v1.Hash{Algorithm: "sha256", Hex: hex.EncodeToString(hasher.Sum(nil))},
		size:      size,
	}, nil
}

func (l *fileLayer) MediaType() (types.MediaType, error)  { return l.mediaType, nil }
func (l *fileLayer) Digest() (v1.Hash, error)             { return l.digest, nil }
func (l *fileLayer) DiffID() (v1.Hash, error)             { return l.digest, nil }
func (l *fileLayer) Size() (int64, error)                 { return l.size, nil }
func (l *fileLayer) Compressed() (io.ReadCloser, error)   { return l.open() }
func (l *fileLayer) Uncompressed() (io.ReadCloser, error) { return l.open() }

// taggableManifest pushes a pre-built manifest as-is. ggcr's mutate package can
// only assemble image manifests, which an artifact is not.
type taggableManifest struct {
	raw       []byte
	mediaType types.MediaType
}

func (t *taggableManifest) RawManifest() ([]byte, error) { return t.raw, nil }

func (t *taggableManifest) MediaType() (types.MediaType, error) { return t.mediaType, nil }

// annotationScopes holds annotations separated by their target scope.
// Keys prefixed with "index:" target the image index manifest, keys prefixed
// with "manifest:" or unprefixed target image manifests.
type annotationScopes struct {
	index    map[string]string // applied to the image index manifest
	manifest map[string]string // applied to each image manifest
}

// parseAnnotationScopes splits annotation keys by their scope prefix.
// Keys prefixed with "index:" are routed to the index manifest, keys prefixed
// with "manifest:" or unprefixed are routed to image manifests.
func (p *ociPusher) parseAnnotationScopes(annotations map[string]string) annotationScopes {
	scopes := annotationScopes{
		index:    make(map[string]string),
		manifest: make(map[string]string),
	}
	for k, v := range annotations {
		switch {
		case strings.HasPrefix(k, "index:"):
			scopes.index[strings.TrimPrefix(k, "index:")] = v
		case strings.HasPrefix(k, "manifest:"):
			scopes.manifest[strings.TrimPrefix(k, "manifest:")] = v
		default:
			scopes.manifest[k] = v
		}
	}
	return scopes
}

// imageSize returns the total compressed size of an image (config + layers)
// using only manifest metadata — no blob downloads are performed.
func (p *ociPusher) imageSize(img v1.Image) (int64, error) {
	m, err := img.Manifest()
	if err != nil {
		return 0, err
	}
	var total int64
	total += m.Config.Size
	for _, l := range m.Layers {
		total += l.Size
	}
	return total, nil
}

// indexSize returns the total compressed size across all child images of an
// image index. Each child manifest is fetched to read its layer sizes, but no
// blobs are downloaded.
func (p *ociPusher) indexSize(idx v1.ImageIndex) (int64, error) {
	im, err := idx.IndexManifest()
	if err != nil {
		return 0, err
	}
	var total int64
	for _, desc := range im.Manifests {
		img, err := idx.Image(desc.Digest)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve child image %s: %w", desc.Digest, err)
		}
		sz, err := p.imageSize(img)
		if err != nil {
			return 0, fmt.Errorf("failed to compute size of child image %s: %w", desc.Digest, err)
		}
		total += sz
	}
	return total, nil
}

// artifactSize returns the total compressed size of an OCI artifact (config +
// layers) from its descriptor metadata. For image indexes, this includes the
// sum across all child images. No blobs are downloaded.
func (p *ociPusher) artifactSize(desc *remote.Descriptor) (int64, error) {
	switch {
	case desc.MediaType.IsImage():
		img, err := desc.Image()
		if err != nil {
			return 0, fmt.Errorf("failed to resolve source image: %w", err)
		}
		return p.imageSize(img)
	case desc.MediaType.IsIndex():
		idx, err := desc.ImageIndex()
		if err != nil {
			return 0, fmt.Errorf("failed to resolve source image index: %w", err)
		}
		return p.indexSize(idx)
	default:
		return 0, &promotion.TerminalError{
			Err: fmt.Errorf("unsupported media type %q", desc.MediaType),
		}
	}
}

// push pushes the described artifact to the destination reference, optionally
// applying scoped annotations to the manifest.
func (p *ociPusher) push(
	desc *remote.Descriptor,
	srcRef, dstRef name.Reference,
	annotations map[string]string,
	dstOpts []remote.Option,
) (v1.Hash, error) {
	// Enforce the size limit only when copying across repositories (registry +
	// path). Within the same repository the blobs are already present, so no
	// large transfer occurs. A negative maxArtifactSize disables the check.
	if p.maxArtifactSize >= 0 && srcRef.Context().String() != dstRef.Context().String() {
		if p.maxArtifactSize == 0 {
			return v1.Hash{}, &promotion.TerminalError{
				Err: fmt.Errorf("cross-repository push is disabled"),
			}
		}
		sz, err := p.artifactSize(desc)
		if err != nil {
			return v1.Hash{}, err
		}
		if sz > p.maxArtifactSize {
			return v1.Hash{}, &promotion.TerminalError{
				Err: fmt.Errorf(
					"compressed artifact size %s exceeds maximum allowed size of %s",
					libfmt.FormatByteCount(sz), libfmt.FormatByteCount(p.maxArtifactSize),
				),
			}
		}
	}

	scopes := p.parseAnnotationScopes(annotations)

	switch {
	case desc.MediaType.IsImage():
		img, err := desc.Image()
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to resolve source image: %w", err)
		}
		annotated, err := mutate.Annotations(img, nil, scopes.manifest)
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to annotate image: %w", err)
		}
		img = annotated.(v1.Image) //nolint:forcetypeassert
		if err = remote.Write(dstRef, img, dstOpts...); err != nil {
			return v1.Hash{}, fmt.Errorf("failed to push image to %q: %w", dstRef.String(), err)
		}
		return img.Digest()

	case desc.MediaType.IsIndex():
		idx, err := desc.ImageIndex()
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to resolve source image index: %w", err)
		}
		annotated, err := mutate.Annotations(idx, scopes.index, scopes.manifest)
		if err != nil {
			return v1.Hash{}, fmt.Errorf("failed to annotate index: %w", err)
		}
		idx = annotated.(v1.ImageIndex) //nolint:forcetypeassert
		if err = remote.WriteIndex(dstRef, idx, dstOpts...); err != nil {
			return v1.Hash{}, fmt.Errorf("failed to push image index to %q: %w", dstRef.String(), err)
		}
		return idx.Digest()

	default:
		return v1.Hash{}, &promotion.TerminalError{
			Err: fmt.Errorf("unsupported media type %q", desc.MediaType),
		}
	}
}
