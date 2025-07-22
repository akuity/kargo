package builtin

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/promotion"
)

func Test_ociPuller_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           promotion.Config
		expectedProblems []string
	}{
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
			name: "valid config with optional fields",
			config: promotion.Config{
				"imageRef":              "registry.example.com/image:tag",
				"outPath":               "output/file.tar",
				"mediaType":             "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"insecureSkipTLSVerify": true,
			},
		},
	}

	r := newOCIPuller(nil)
	runner, ok := r.(*ociPuller)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func Test_ociPuller_prepareOutputPath(t *testing.T) {
	tests := []struct {
		name       string
		outPath    string
		assertions func(*testing.T, string, string, error)
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
	}

	runner := &ociPuller{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absPath, err := runner.prepareOutputPath(workDir, tt.outPath)
			tt.assertions(t, workDir, absPath, err)
		})
	}
}

func Test_ociPuller_extractLayerToFile(t *testing.T) {
	tests := []struct {
		name       string
		layers     []v1.Layer
		manifest   *v1.Manifest
		mediaType  string
		assertions func(*testing.T, string, error)
	}{
		{
			name: "successful extraction",
			layers: []v1.Layer{
				static.NewLayer([]byte("layer content"), types.DockerLayer),
			},
			manifest:  createTestManifest([]types.MediaType{types.DockerLayer}),
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
			layers: []v1.Layer{
				static.NewLayer([]byte("layer1"), types.DockerLayer),
				static.NewLayer([]byte("layer2"), types.OCILayer),
			},
			manifest: createTestManifest([]types.MediaType{
				types.DockerLayer,
				types.OCILayer,
			}),
			mediaType: string(types.OCILayer),
			assertions: func(t *testing.T, absOutPath string, err error) {
				require.NoError(t, err)
				require.FileExists(t, absOutPath)
				content, readErr := os.ReadFile(absOutPath)
				require.NoError(t, readErr)
				assert.Equal(t, "layer2", string(content))
			},
		},
	}

	runner := &ociPuller{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absOutPath := path.Join(workDir, "output.tar")

			img := &fake.FakeImage{
				LayersStub: func() ([]v1.Layer, error) {
					return tt.layers, nil
				},
				ManifestStub: func() (*v1.Manifest, error) {
					return tt.manifest, nil
				},
			}

			err := runner.extractLayerToFile(img, tt.mediaType, absOutPath)
			tt.assertions(t, absOutPath, err)
		})
	}
}

func Test_ociPuller_writeLayerToFile(t *testing.T) {
	tests := []struct {
		name       string
		layerData  string
		assertions func(*testing.T, string, error)
	}{
		{
			name:      "successful write",
			layerData: "test layer content",
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
			assertions: func(t *testing.T, absOutPath string, err error) {
				require.NoError(t, err)

				// Verify file exists and is empty
				require.FileExists(t, absOutPath)
				content, readErr := os.ReadFile(absOutPath)
				require.NoError(t, readErr)
				assert.Empty(t, string(content))
			},
		},
	}

	runner := &ociPuller{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absOutPath := filepath.Join(workDir, "output.tar")

			layer := static.NewLayer(
				[]byte(tt.layerData),
				types.DockerLayer,
			)

			err := runner.writeLayerToFile(layer, absOutPath)
			tt.assertions(t, absOutPath, err)
		})
	}
}

func Test_ociPuller_createTempFile(t *testing.T) {
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

	runner := &ociPuller{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			absOutPath := filepath.Join(workDir, "output.tar.gz")
			tempFile, tempPath, err := runner.createTempFile(absOutPath)
			tt.assertions(t, tempFile, tempPath, err)
		})
	}
}

func Test_ociPuller_copyLayerToFile(t *testing.T) {
	tests := []struct {
		name       string
		layerData  string
		assertions func(*testing.T, *os.File, error)
	}{
		{
			name:      "successful copy",
			layerData: "test layer content",
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
			name:      "empty layer",
			layerData: "",
			assertions: func(t *testing.T, _ *os.File, err error) {
				assert.NoError(t, err)
			},
		},
	}

	runner := &ociPuller{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			tempFile, err := os.CreateTemp(workDir, "test-*.tmp")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = tempFile.Close()
			})

			layer := static.NewLayer(
				[]byte(tt.layerData),
				types.DockerLayer,
			)

			err = runner.copyLayerToFile(layer, tempFile)
			tt.assertions(t, tempFile, err)
		})
	}
}

func Test_ociPuller_findTargetLayer(t *testing.T) {
	tests := []struct {
		name            string
		layers          []v1.Layer
		manifest        *v1.Manifest
		targetMediaType string
		assertions      func(*testing.T, v1.Layer, error)
	}{
		{
			name: "no specific media type returns first layer",
			layers: []v1.Layer{
				static.NewLayer([]byte("layer1"), types.DockerLayer),
				static.NewLayer([]byte("layer2"), types.OCILayer),
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
			layers: []v1.Layer{
				static.NewLayer([]byte("layer1"), types.DockerLayer),
				static.NewLayer([]byte("layer2"), types.OCILayer),
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
			layers: []v1.Layer{
				static.NewLayer([]byte("layer1"), types.DockerLayer),
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
			name:     "no layers",
			layers:   []v1.Layer{},
			manifest: createTestManifest(nil),
			assertions: func(t *testing.T, layer v1.Layer, err error) {
				assert.ErrorContains(t, err, "image has no layers")
				assert.Nil(t, layer)
			},
		},
	}

	runner := &ociPuller{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := &fake.FakeImage{
				LayersStub: func() ([]v1.Layer, error) {
					return tt.layers, nil
				},
				ManifestStub: func() (*v1.Manifest, error) {
					return tt.manifest, nil
				},
			}

			layer, err := runner.findTargetLayer(img, tt.manifest, tt.targetMediaType)
			tt.assertions(t, layer, err)
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
