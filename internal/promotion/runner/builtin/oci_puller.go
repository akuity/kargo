package builtin

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	intio "github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	maxArtifactSize = 100 << 20
)

// ociPuller is an implementation of the promotion.StepRunner interface
// that pulls OCI artifacts from a registry.
type ociPuller struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newOCIPuller returns an implementation of the promotion.StepRunner interface
// that pulls OCI artifacts from a registry. It uses the provided credentials
// database to authenticate with the registry.
func newOCIPuller(credsDB credentials.Database) promotion.StepRunner {
	r := &ociPuller{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (o *ociPuller) Name() string {
	return "oci-pull"
}

// Run implements the promotion.StepRunner interface.
func (o *ociPuller) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := o.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	cfg, err := promotion.ConfigToStruct[builtin.OCIPullConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not convert config into %s config: %w", o.Name(), err)
	}

	return o.run(ctx, stepCtx, cfg)
}

// validate validates ociPuller configuration against a JSON schema.
func (o *ociPuller) validate(cfg promotion.Config) error {
	return validate(o.schemaLoader, gojsonschema.NewGoLoader(cfg), o.Name())
}

// run executes the ociPuller step with the provided configuration.
func (o *ociPuller) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIPullConfig,
) (promotion.StepResult, error) {
	absOutPath, err := o.prepareOutputPath(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	img, err := o.pullImage(ctx, stepCtx, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	if err = o.extractLayerToFile(img, cfg.MediaType, absOutPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

// prepareOutputPath validates and prepares the output path for the artifact.
func (o *ociPuller) prepareOutputPath(workDir, outPath string) (string, error) {
	absOutPath, err := securejoin.SecureJoin(workDir, outPath)
	if err != nil {
		return "", fmt.Errorf("failed to join path %q: %w", outPath, err)
	}

	destDir := filepath.Dir(absOutPath)
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	return absOutPath, nil
}

// pullImage pulls the OCI image/artifact from the registry.
func (o *ociPuller) pullImage(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.OCIPullConfig,
) (v1.Image, error) {
	ref, err := name.ParseReference(cfg.ImageRef)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference %q: %w", cfg.ImageRef, err)
	}

	remoteOpts, err := o.buildRemoteOptions(ctx, stepCtx, ref, cfg)
	if err != nil {
		return nil, err
	}

	img, err := remote.Image(ref, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %q: %w", cfg.ImageRef, err)
	}

	return img, nil
}

// buildRemoteOptions constructs the remote options for image pulling.
func (o *ociPuller) buildRemoteOptions(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	ref name.Reference,
	cfg builtin.OCIPullConfig,
) ([]remote.Option, error) {
	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(o.buildHTTPTransport(cfg)),
	}

	// Configure authentication
	if authOpt, err := o.getAuthOption(ctx, stepCtx, ref); err != nil {
		return nil, err
	} else if authOpt != nil {
		remoteOpts = append(remoteOpts, authOpt)
	}

	return remoteOpts, nil
}

// getAuthOption retrieves and configures authentication for the registry.
func (o *ociPuller) getAuthOption(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	ref name.Reference,
) (remote.Option, error) {
	repoURL := ref.Context().String()

	creds, err := o.credsDB.Get(ctx, stepCtx.Project, credentials.TypeImage, repoURL)
	if err != nil {
		return nil, fmt.Errorf("error obtaining credentials for image repo %q: %w", repoURL, err)
	}

	if creds != nil && (creds.Username != "" || creds.Password != "") {
		return remote.WithAuth(&authn.Basic{
			Username: creds.Username,
			Password: creds.Password,
		}), nil
	}

	return nil, nil
}

// buildHTTPTransport creates a new HTTP transport with TLS settings based on
// the configuration.
func (o *ociPuller) buildHTTPTransport(cfg builtin.OCIPullConfig) *http.Transport {
	httpTransport := cleanhttp.DefaultTransport()
	if cfg.InsecureSkipTLSVerify {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
	}
	return httpTransport
}

// extractLayerToFile extracts the target layer from the image to the specified
// file.
func (o *ociPuller) extractLayerToFile(img v1.Image, mediaType, absOutPath string) error {
	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	targetLayer, err := o.findTargetLayer(img, manifest, mediaType)
	if err != nil {
		return fmt.Errorf("failed to find target layer: %w", err)
	}

	return o.writeLayerToFile(targetLayer, absOutPath)
}

// writeLayerToFile writes the layer content to a file using atomic operations.
func (o *ociPuller) writeLayerToFile(layer v1.Layer, absOutPath string) error {
	tempFile, tempPath, err := o.createTempFile(absOutPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err = o.copyLayerToFile(layer, tempFile); err != nil {
		return err
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err = fs.SimpleAtomicMove(tempPath, absOutPath); err != nil {
		return fmt.Errorf("failed to move file to final destination: %w", err)
	}

	return nil
}

// createTempFile creates a temporary file in the same directory as the target.
func (o *ociPuller) createTempFile(absOutPath string) (*os.File, string, error) {
	destDir := filepath.Dir(absOutPath)
	baseFile := filepath.Base(absOutPath)

	tempFile, err := os.CreateTemp(destDir, baseFile+".tmp")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	if err = tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return nil, "", fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	return tempFile, tempFile.Name(), nil
}

// copyLayerToFile copies layer content to the file with size limits.
func (o *ociPuller) copyLayerToFile(layer v1.Layer, tempFile *os.File) error {
	size, err := layer.Size()
	if err != nil {
		return fmt.Errorf("failed to get layer size: %w", err)
	}
	if size > maxArtifactSize {
		return &promotion.TerminalError{
			Err: fmt.Errorf("layer size %d exceeds maximum allowed size of %d bytes", size, maxArtifactSize),
		}
	}

	layerReader, err := layer.Compressed()
	if err != nil {
		return fmt.Errorf("failed to get layer content: %w", err)
	}

	if _, err = intio.LimitCopy(tempFile, layerReader, maxArtifactSize); err != nil {
		if errors.Is(err, &intio.BodyTooLargeError{}) {
			return &promotion.TerminalError{
				Err: fmt.Errorf("failed to copy layer content: %w", err),
			}
		}
		return fmt.Errorf("failed to copy layer content: %w", err)
	}

	return nil
}

// findTargetLayer finds the appropriate layer based on media type preference.
func (o *ociPuller) findTargetLayer(
	img v1.Image,
	manifest *v1.Manifest,
	targetMediaType string,
) (v1.Layer, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to get image layers: %w", err)
	}

	if len(layers) == 0 {
		return nil, errors.New("image has no layers")
	}

	// If a specific media type is requested, find the first matching layer
	if targetMediaType != "" {
		for i, layerDesc := range manifest.Layers {
			if string(layerDesc.MediaType) == targetMediaType {
				if i >= len(layers) {
					return nil, fmt.Errorf("layer index %d out of range", i)
				}
				return layers[i], nil
			}
		}
		return nil, fmt.Errorf("no layer found with media type %q", targetMediaType)
	}

	// If no specific media type requested, return the first layer
	return layers[0], nil
}
