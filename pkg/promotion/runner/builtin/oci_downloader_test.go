package builtin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_ociDownloader_validate(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "imageRef is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): imageRef is required",
			},
		},
		{
			name:   "outPath is not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): outPath is required",
			},
		},
		{
			name: "valid config",
			config: promotion.Config{
				"imageRef": "registry.example.com/image:tag",
				"outPath":  "output/file.tar",
			},
		},
		{
			name: "valid config with OCI protocol",
			config: promotion.Config{
				"imageRef": "oci://registry.example.com/image:tag",
				"outPath":  "output/file.tar",
			},
		},
		{
			name: "valid config with optional fields",
			config: promotion.Config{
				"imageRef":              "registry.example.com/image:tag",
				"outPath":               "output/file.tar",
				"mediaType":             "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"insecureSkipTLSVerify": true,
			},
		},
	}

	r := newOCIDownloader(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*ociDownloader)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_ociDownloader_parseImageReference(t *testing.T) {
	tests := []struct {
		name       string
		imageRef   string
		assertions func(*testing.T, name.Reference, credentials.Type, error)
	}{
		{
			name:     "standard registry reference",
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeImage, credType)
				assert.Equal(t, "registry.example.com/image:tag", ref.String())
			},
		},
		{
			name:     "OCI Helm reference",
			imageRef: "oci://registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeHelm, credType)
				assert.Equal(t, "registry.example.com/chart:1.0.0", ref.String())
			},
		},
		{
			name:     "invalid reference",
			imageRef: "invalid::reference",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				assert.ErrorContains(t, err, "invalid image reference")
				assert.Nil(t, ref)
				assert.Empty(t, credType)
			},
		},
		{
			name:     "OCI reference with port",
			imageRef: "oci://localhost:5000/chart:latest",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeHelm, credType)
				assert.Equal(t, "localhost:5000/chart:latest", ref.String())
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, credType, err := runner.parseImageReference(tt.imageRef)
			tt.assertions(t, ref, credType, err)
		})
	}
}

func Test_ociDownloader_resolveImage(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.OCIDownloadConfig
		credsDB    credentials.Database
		assertions func(*testing.T, v1.Image, error)
	}{
		{
			name: "invalid image reference",
			cfg: builtin.OCIDownloadConfig{
				ImageRef: "invalid::reference",
			},
			credsDB: &credentials.FakeDB{},
			assertions: func(t *testing.T, img v1.Image, err error) {
				assert.ErrorContains(t, err, "failed to parse image reference")
				assert.Nil(t, img)
			},
		},
		{
			name: "credentials error",
			cfg: builtin.OCIDownloadConfig{
				ImageRef: "registry.example.com/image:tag",
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return nil, errors.New("credentials error")
				},
			},
			assertions: func(t *testing.T, img v1.Image, err error) {
				assert.ErrorContains(t, err, "error obtaining credentials")
				assert.Nil(t, img)
			},
		},
		{
			name: "OCI Helm reference credentials lookup",
			cfg: builtin.OCIDownloadConfig{
				ImageRef: "oci://registry.example.com/chart:1.0.0",
			},
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					// Verify the credential type and URL format for OCI Helm
					assert.Equal(t, credentials.TypeHelm, credType)
					assert.Equal(t, "oci://registry.example.com/chart", repoURL)
					return nil, errors.New("test credentials error")
				},
			},
			assertions: func(t *testing.T, img v1.Image, err error) {
				assert.ErrorContains(t, err, "error obtaining credentials")
				assert.Nil(t, img)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &ociDownloader{credsDB: tt.credsDB}

			stepCtx := &promotion.StepContext{
				Project: "fake-project",
			}

			img, err := runner.resolveImage(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, img, err)
		})
	}
}

func Test_ociDownloader_prepareOutputPath(t *testing.T) {
	tests := []struct {
		name           string
		outPath        string
		allowOverwrite bool
		setup          func(*testing.T, string)
		assertions     func(*testing.T, string, string, error)
	}{
		{
			name:    "valid relative path",
			outPath: "output/file.tar",
			assertions: func(t *testing.T, workDir, absPath string, err error) {
				require.NoError(t, err)
				assert.True(t, strings.HasPrefix(absPath, workDir))
				// Check that the directory was created
				dir := path.Dir(absPath)
				_, statErr := os.Stat(dir)
				assert.NoError(t, statErr)
			},
		},
		{
			name:    "valid nested path",
			outPath: "deep/nested/output/file.tar",
			assertions: func(t *testing.T, workDir, absPath string, err error) {
				require.NoError(t, err)
				assert.True(t, strings.HasPrefix(absPath, workDir))
				// Check that the directory was created
				dir := path.Dir(absPath)
				_, statErr := os.Stat(dir)
				assert.NoError(t, statErr)
			},
		},
		{
			name:    "path traversal attempt",
			outPath: "../../../etc/passwd",
			assertions: func(t *testing.T, workDir, absPath string, err error) {
				require.NoError(t, err)
				assert.Equal(t, filepath.Join(workDir, "etc", "passwd"), absPath)
			},
		},
		{
			name:    "path already exists",
			outPath: "output/file.tar",
			setup: func(t *testing.T, workDir string) {
				// Create the output file to simulate an existing path
				existingFile := filepath.Join(workDir, "output", "file.tar")
				require.NoError(t, os.MkdirAll(path.Dir(existingFile), 0o700))
				require.NoError(t, os.WriteFile(existingFile, []byte("existing content"), 0o600))
			},
			allowOverwrite: false,
			assertions: func(t *testing.T, _, absPath string, err error) {
				assert.ErrorContains(t, err, "file already exists")
				assert.Empty(t, absPath)
			},
		},
		{
			name:    "path already exists with overwrite allowed",
			outPath: "output/file.tar",
			setup: func(t *testing.T, workDir string) {
				// Create the output file to simulate an existing path
				existingFile := filepath.Join(workDir, "output", "file.tar")
				require.NoError(t, os.MkdirAll(path.Dir(existingFile), 0o700))
				require.NoError(t, os.WriteFile(existingFile, []byte("existing content"), 0o600))
			},
			allowOverwrite: true,
			assertions: func(t *testing.T, workDir, absPath string, err error) {
				require.NoError(t, err)
				assert.True(t, strings.HasPrefix(absPath, workDir))
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, workDir)
			}

			absPath, err := runner.prepareOutputPath(workDir, tt.outPath, tt.allowOverwrite)
			tt.assertions(t, workDir, absPath, err)
		})
	}
}

func Test_ociDownloader_buildRemoteOptions(t *testing.T) {
	tests := []struct {
		name       string
		credsDB    credentials.Database
		cfg        builtin.OCIDownloadConfig
		assertions func(*testing.T, []remote.Option, error)
	}{
		{
			name:    "basic options without auth",
			credsDB: &credentials.FakeDB{},
			cfg: builtin.OCIDownloadConfig{
				ImageRef: "registry.example.com/image:tag",
			},
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				require.NoError(t, err)
				assert.Len(t, opts, 2)
			},
		},
		{
			name: "options with authentication",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return &credentials.Credentials{
						Username: "user",
						Password: "pass",
					}, nil
				},
			},
			cfg: builtin.OCIDownloadConfig{
				ImageRef: "registry.example.com/image:tag",
			},
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				require.NoError(t, err)
				assert.Len(t, opts, 3)
			},
		},
		{
			name: "OCI Helm authentication",
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					// Verify the credential type and URL format for OCI Helm
					assert.Equal(t, credentials.TypeHelm, credType)
					assert.Equal(t, "oci://registry.example.com/chart", repoURL)
					return &credentials.Credentials{
						Username: "helm-user",
						Password: "helm-pass",
					}, nil
				},
			},
			cfg: builtin.OCIDownloadConfig{
				ImageRef: "oci://registry.example.com/chart:1.0.0",
			},
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				require.NoError(t, err)
				assert.Len(t, opts, 3)
			},
		},
		{
			name: "credentials error",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return nil, errors.New("credentials database error")
				},
			},
			cfg: builtin.OCIDownloadConfig{
				ImageRef:              "registry.example.com/image:tag",
				InsecureSkipTLSVerify: false,
			},
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				assert.ErrorContains(t, err, "error obtaining credentials")
				assert.Nil(t, opts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &ociDownloader{credsDB: tt.credsDB}

			ref, credType, err := runner.parseImageReference(tt.cfg.ImageRef)
			require.NoError(t, err)

			stepCtx := &promotion.StepContext{
				Project: "fake-project",
			}

			opts, err := runner.buildRemoteOptions(context.Background(), stepCtx, tt.cfg, ref, credType)
			tt.assertions(t, opts, err)
		})
	}
}

func Test_ociDownloader_buildHTTPTransport(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.OCIDownloadConfig
		assertions func(*testing.T, *http.Transport)
	}{
		{
			name: "default TLS verification",
			cfg: builtin.OCIDownloadConfig{
				InsecureSkipTLSVerify: false,
			},
			assertions: func(t *testing.T, transport *http.Transport) {
				require.NotNil(t, transport)
				if transport.TLSClientConfig != nil {
					assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
				}
			},
		},
		{
			name: "skip TLS verification",
			cfg: builtin.OCIDownloadConfig{
				InsecureSkipTLSVerify: true,
			},
			assertions: func(t *testing.T, transport *http.Transport) {
				require.NotNil(t, transport)
				require.NotNil(t, transport.TLSClientConfig)
				assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := runner.buildHTTPTransport(tt.cfg)
			tt.assertions(t, transport)
		})
	}
}

func Test_ociDownloader_getAuthOption(t *testing.T) {
	tests := []struct {
		name       string
		credsDB    credentials.Database
		imageRef   string
		assertions func(*testing.T, remote.Option, error)
	}{
		{
			name:     "no credentials for image",
			credsDB:  &credentials.FakeDB{},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				assert.Nil(t, opt)
			},
		},
		{
			name:     "no credentials for Helm",
			credsDB:  &credentials.FakeDB{},
			imageRef: "registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				assert.Nil(t, opt)
			},
		},
		{
			name: "valid image credentials",
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					assert.Equal(t, credentials.TypeImage, credType)
					assert.Equal(t, "registry.example.com/image", repoURL)
					return &credentials.Credentials{
						Username: "user",
						Password: "pass",
					}, nil
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				require.NotNil(t, opt)
			},
		},
		{
			name: "valid Helm credentials with OCI prefix",
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					assert.Equal(t, credentials.TypeHelm, credType)
					assert.Equal(t, "oci://registry.example.com/chart", repoURL)
					return &credentials.Credentials{
						Username: "helm-user",
						Password: "helm-pass",
					}, nil
				},
			},
			imageRef: "oci://registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				require.NotNil(t, opt)
			},
		},
		{
			name: "empty username and password",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return &credentials.Credentials{
						Username: "",
						Password: "",
					}, nil
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				assert.Nil(t, opt)
			},
		},
		{
			name: "credentials database error",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return nil, errors.New("credentials database error")
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				assert.ErrorContains(t, err, "error obtaining credentials")
				assert.ErrorContains(t, err, "credentials database error")
				assert.Nil(t, opt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &ociDownloader{credsDB: tt.credsDB}

			ref, credType, err := runner.parseImageReference(tt.imageRef)
			require.NoError(t, err)

			stepCtx := &promotion.StepContext{
				Project: "fake-project",
			}

			opt, err := runner.getAuthOption(context.Background(), stepCtx, ref, credType)
			tt.assertions(t, opt, err)
		})
	}
}

func Test_ociDownloader_extractLayerToFile(t *testing.T) {
	tests := []struct {
		name       string
		setupImg   func() v1.Image
		layers     []v1.Layer
		manifest   *v1.Manifest
		mediaType  string
		assertions func(*testing.T, string, error)
	}{
		{
			name: "successful extraction",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{
							static.NewLayer([]byte("layer content"), types.DockerLayer),
						}, nil
					},
					ManifestStub: func() (*v1.Manifest, error) {
						return createTestManifest([]types.MediaType{types.DockerLayer}), nil
					},
				}
			},
			mediaType: "",
			assertions: func(t *testing.T, absOutPath string, err error) {
				require.NoError(t, err)
				require.FileExists(t, absOutPath)
				content, readErr := os.ReadFile(absOutPath)
				require.NoError(t, readErr)
				assert.Equal(t, "layer content", string(content))
			},
		},
		{
			name: "specific media type extraction",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{
							static.NewLayer([]byte("layer1"), types.DockerLayer),
							static.NewLayer([]byte("layer2"), types.OCILayer),
						}, nil
					},
					ManifestStub: func() (*v1.Manifest, error) {
						return createTestManifest([]types.MediaType{
							types.DockerLayer,
							types.OCILayer,
						}), nil
					},
				}
			},
			mediaType: string(types.OCILayer),
			assertions: func(t *testing.T, absOutPath string, err error) {
				require.NoError(t, err)
				require.FileExists(t, absOutPath)
				content, readErr := os.ReadFile(absOutPath)
				require.NoError(t, readErr)
				assert.Equal(t, "layer2", string(content))
			},
		},
		{
			name: "manifest error",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					ManifestStub: func() (*v1.Manifest, error) {
						return nil, errors.New("manifest error")
					},
				}
			},
			mediaType: "",
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, "failed to get manifest")
			},
		},
		{
			name: "find layer error",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{}, nil
					},
					ManifestStub: func() (*v1.Manifest, error) {
						return createTestManifest(nil), nil
					},
				}
			},
			mediaType: "",
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, "failed to find target layer")
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absOutPath := path.Join(workDir, "output.tar")

			img := tt.setupImg()
			err := runner.extractLayerToFile(img, tt.mediaType, absOutPath)
			tt.assertions(t, absOutPath, err)
		})
	}
}

func Test_ociDownloader_findTargetLayer(t *testing.T) {
	tests := []struct {
		name            string
		setupImg        func() v1.Image
		manifest        *v1.Manifest
		targetMediaType string
		assertions      func(*testing.T, v1.Layer, error)
	}{
		{
			name: "no specific media type returns first layer",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{
							static.NewLayer([]byte("layer1"), types.DockerLayer),
							static.NewLayer([]byte("layer2"), types.OCILayer),
						}, nil
					},
				}
			},
			manifest: createTestManifest([]types.MediaType{
				types.DockerLayer,
				types.OCILayer,
			}),
			targetMediaType: "",
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				require.NoError(t, err)
				require.NotNil(t, layer)
				content, readErr := layer.Compressed()
				require.NoError(t, readErr)
				data, readErr := io.ReadAll(content)
				require.NoError(t, readErr)
				assert.Equal(t, "layer1", string(data))
			},
		},
		{
			name: "specific media type found",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{
							static.NewLayer([]byte("layer1"), types.DockerLayer),
							static.NewLayer([]byte("layer2"), types.OCILayer),
						}, nil
					},
				}
			},
			manifest: createTestManifest([]types.MediaType{
				types.DockerLayer,
				types.OCILayer,
			}),
			targetMediaType: string(types.OCILayer),
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				require.NoError(t, err)

				require.NotNil(t, layer)
				content, readErr := layer.Compressed()
				require.NoError(t, readErr)

				data, readErr := io.ReadAll(content)
				require.NoError(t, readErr)
				assert.Equal(t, "layer2", string(data))
			},
		},
		{
			name: "specific media type not found",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{
							static.NewLayer([]byte("layer1"), types.DockerLayer),
						}, nil
					},
				}
			},
			manifest: createTestManifest([]types.MediaType{
				types.DockerLayer,
			}),
			targetMediaType: string(types.OCILayer),
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				assert.ErrorContains(t, err, "no layer found with media type")
				assert.Nil(t, layer)
			},
		},
		{
			name: "no layers",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{}, nil
					},
				}
			},
			manifest: createTestManifest(nil),
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				assert.ErrorContains(t, err, "image has no layers")
				assert.Nil(t, layer)
			},
		},
		{
			name: "layers error",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return nil, errors.New("layers error")
					},
				}
			},
			manifest: createTestManifest([]types.MediaType{types.DockerLayer}),
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				assert.ErrorContains(t, err, "failed to get image layers")
				assert.Nil(t, layer)
			},
		},
		{
			name: "layer index out of range",
			setupImg: func() v1.Image {
				return &fake.FakeImage{
					LayersStub: func() ([]v1.Layer, error) {
						return []v1.Layer{
							static.NewLayer([]byte("layer1"), types.DockerLayer),
						}, nil
					},
				}
			},
			manifest: &v1.Manifest{
				SchemaVersion: 2,
				MediaType:     types.DockerManifestSchema2,
				Layers: []v1.Descriptor{
					{MediaType: types.DockerLayer},
					{MediaType: types.OCILayer}, // This will cause index out of range
				},
			},
			targetMediaType: string(types.OCILayer),
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				assert.ErrorContains(t, err, "layer index")
				assert.ErrorContains(t, err, "out of range")
				assert.Nil(t, layer)
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := tt.setupImg()
			layer, err := runner.findTargetLayer(img, tt.manifest, tt.targetMediaType)
			tt.assertions(t, layer, err)
		})
	}
}

func Test_ociDownloader_writeLayerToFile(t *testing.T) {
	tests := []struct {
		name       string
		layerData  string
		setupPath  func(string) string
		assertions func(*testing.T, string, error)
	}{
		{
			name:      "successful write",
			layerData: "test layer content",
			setupPath: func(workDir string) string {
				return filepath.Join(workDir, "output.tar")
			},
			assertions: func(t *testing.T, absOutPath string, err error) {
				require.NoError(t, err)

				// Verify file exists and has correct content
				require.FileExists(t, absOutPath)
				content, readErr := os.ReadFile(absOutPath)
				require.NoError(t, readErr)
				assert.Equal(t, "test layer content", string(content))
			},
		},
		{
			name:      "empty content",
			layerData: "",
			setupPath: func(workDir string) string {
				return filepath.Join(workDir, "output.tar")
			},
			assertions: func(t *testing.T, absOutPath string, err error) {
				require.NoError(t, err)

				// Verify file exists and is empty
				require.FileExists(t, absOutPath)
				content, readErr := os.ReadFile(absOutPath)
				require.NoError(t, readErr)
				assert.Empty(t, string(content))
			},
		},
		{
			name:      "temp file creation error",
			layerData: "test content",
			setupPath: func(workDir string) string {
				// Try to create temp file in non-existent directory
				return filepath.Join(workDir, "nonexistent", "subdir", "output.tar")
			},
			assertions: func(t *testing.T, _ string, err error) {
				assert.ErrorContains(t, err, "failed to create temporary file")
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absOutPath := tt.setupPath(workDir)

			layer := static.NewLayer(
				[]byte(tt.layerData),
				types.DockerLayer,
			)

			err := runner.writeLayerToFile(layer, absOutPath)
			tt.assertions(t, absOutPath, err)
		})
	}
}

func Test_ociDownloader_createTempFile(t *testing.T) {
	tests := []struct {
		name       string
		assertions func(*testing.T, *os.File, string, error)
	}{
		{
			name: "successful temp file creation",
			assertions: func(t *testing.T, tempFile *os.File, tempPath string, err error) {
				require.NoError(t, err)
				require.NotNil(t, tempFile)
				require.NotEmpty(t, tempPath)

				// Check file exists and has correct permissions
				info, statErr := os.Stat(tempPath)
				require.NoError(t, statErr)
				assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

				t.Cleanup(func() {
					_ = tempFile.Close()
				})
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absOutPath := filepath.Join(workDir, "output.tar.gz")
			tempFile, tempPath, err := runner.createTempFile(absOutPath)
			tt.assertions(t, tempFile, tempPath, err)
		})
	}
}

func Test_ociDownloader_copyLayerToFile(t *testing.T) {
	tests := []struct {
		name       string
		layer      v1.Layer
		assertions func(*testing.T, *os.File, error)
	}{
		{
			name:  "successful copy",
			layer: static.NewLayer([]byte("test layer content"), types.DockerLayer),
			assertions: func(t *testing.T, f *os.File, err error) {
				require.NoError(t, err)

				// Verify content was written
				_, seekErr := f.Seek(0, 0)
				require.NoError(t, seekErr)

				content, readErr := io.ReadAll(f)
				require.NoError(t, readErr)
				assert.Equal(t, "test layer content", string(content))
			},
		},
		{
			name:  "empty layer",
			layer: static.NewLayer([]byte(""), types.DockerLayer),
			assertions: func(t *testing.T, _ *os.File, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "layer exceeds size limit",
			layer: &fakeSizeLayer{
				size: maxOCIArtifactSize + 1,
				data: []byte("test content"),
			},
			assertions: func(t *testing.T, _ *os.File, err error) {
				assert.Error(t, err)
				var termErr *promotion.TerminalError
				assert.True(t, errors.As(err, &termErr))
				assert.ErrorContains(t, termErr, "exceeds maximum allowed size")
			},
		},
		{
			name: "layer within size limit",
			layer: &fakeSizeLayer{
				size: maxOCIArtifactSize - 1,
				data: []byte("test content"),
			},
			assertions: func(t *testing.T, f *os.File, err error) {
				require.NoError(t, err)

				// Verify content was written
				_, seekErr := f.Seek(0, 0)
				require.NoError(t, seekErr)

				content, readErr := io.ReadAll(f)
				require.NoError(t, readErr)
				assert.Equal(t, "test content", string(content))
			},
		},
	}

	runner := &ociDownloader{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			tempFile, err := os.CreateTemp(workDir, "test-*.tmp")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = tempFile.Close()
			})

			err = runner.copyLayerToFile(tt.layer, tempFile)
			tt.assertions(t, tempFile, err)
		})
	}
}

func createTestManifest(layerMediaTypes []types.MediaType) *v1.Manifest {
	layers := make([]v1.Descriptor, len(layerMediaTypes))
	for i, mt := range layerMediaTypes {
		layers[i] = v1.Descriptor{
			MediaType: mt,
			Size:      100,
			Digest:    v1.Hash{},
		}
	}

	return &v1.Manifest{
		SchemaVersion: 2,
		MediaType:     types.DockerManifestSchema2,
		Layers:        layers,
	}
}

type fakeSizeLayer struct {
	size int64
	data []byte
}

func (l *fakeSizeLayer) Digest() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (l *fakeSizeLayer) DiffID() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (l *fakeSizeLayer) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(l.data))), nil
}

func (l *fakeSizeLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(l.data))), nil
}

func (l *fakeSizeLayer) Size() (int64, error) {
	return l.size, nil
}

func (l *fakeSizeLayer) MediaType() (types.MediaType, error) {
	return types.DockerLayer, nil
}
