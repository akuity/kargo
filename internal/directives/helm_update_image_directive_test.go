package directives

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_helmUpdateImageDirective_run(t *testing.T) {
	tests := []struct {
		name       string
		objects    []client.Object
		stepCtx    *StepContext
		cfg        HelmUpdateImageConfig
		files      map[string]string
		assertions func(*testing.T, string, Result, error)
	}{
		{
			name: "successful run with image updates",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "docker.io/library/nginx",
								},
							},
						},
					},
				},
			},
			stepCtx: &StepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Images: []kargoapi.Image{
								{RepoURL: "docker.io/library/nginx", Tag: "1.19.0"},
							},
						},
					},
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			cfg: HelmUpdateImageConfig{
				Path: "values.yaml",
				Images: []Image{
					{Key: "image.tag", Image: "docker.io/library/nginx", Value: Tag},
				},
			},
			files: map[string]string{
				"values.yaml": "image:\n  tag: oldtag\n",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, Result{Status: StatusSuccess}, result)
				content, err := os.ReadFile(path.Join(workDir, "values.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "tag: 1.19.0")
			},
		},
		{
			name: "no image updates",
			stepCtx: &StepContext{
				Project:         "test-project",
				Freight:         kargoapi.FreightCollection{},
				FreightRequests: []kargoapi.FreightRequest{},
			},
			cfg: HelmUpdateImageConfig{
				Path: "values.yaml",
				Images: []Image{
					{Key: "image.tag", Image: "docker.io/library/non-existent", Value: Tag},
				},
			},
			files: map[string]string{
				"values.yaml": "image:\n  tag: oldtag\n",
			},
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				assert.NoError(t, err)
				assert.Equal(t, Result{Status: StatusSuccess}, result)
				content, err := os.ReadFile(path.Join(workDir, "values.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "tag: oldtag")
			},
		},

		{
			name: "failed to generate image updates",
			stepCtx: &StepContext{
				KargoClient: fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
					Get: func(
						context.Context,
						client.WithWatch,
						client.ObjectKey,
						client.Object,
						...client.GetOption,
					) error {
						return fmt.Errorf("something went wrong")
					},
				}).Build(),
				Project: "test-project",
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "non-existent-warehouse"},
					},
				},
			},
			cfg: HelmUpdateImageConfig{
				Path: "values.yaml",
				Images: []Image{
					{
						Key:   "image.tag",
						Image: "docker.io/library/nginx",
						Value: Tag,
					},
				},
			},
			assertions: func(t *testing.T, _ string, result Result, err error) {
				require.ErrorContains(t, err, "failed to generate image updates")
				require.Errorf(t, err, "something went wrong")
				assert.Equal(t, Result{Status: StatusFailure}, result)
			},
		},
		{
			name: "failed to update values file",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "docker.io/library/nginx",
								},
							},
						},
					},
				},
			},
			stepCtx: &StepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Images: []kargoapi.Image{
								{RepoURL: "docker.io/library/nginx", Tag: "1.19.0"},
							},
						},
					},
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			cfg: HelmUpdateImageConfig{
				Path: "non-existent/values.yaml",
				Images: []Image{
					{Key: "image.tag", Image: "docker.io/library/nginx", Value: Tag},
				},
			},
			assertions: func(t *testing.T, _ string, result Result, err error) {
				assert.Error(t, err)
				assert.Equal(t, Result{Status: StatusFailure}, result)
				assert.Contains(t, err.Error(), "values file update failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := tt.stepCtx

			stepCtx.WorkDir = t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(path.Join(stepCtx.WorkDir, path.Dir(p)), 0o700))
				require.NoError(t, os.WriteFile(path.Join(stepCtx.WorkDir, p), []byte(c), 0o600))
			}

			if stepCtx.KargoClient == nil {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				stepCtx.KargoClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			}

			d := &helmUpdateImageDirective{}
			result, err := d.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func Test_helmUpdateImageDirective_generateImageUpdates(t *testing.T) {
	tests := []struct {
		name       string
		objects    []client.Object
		stepCtx    *StepContext
		cfg        HelmUpdateImageConfig
		assertions func(*testing.T, map[string]string, error)
	}{
		{
			name: "finds image update",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "docker.io/library/nginx",
								},
							},
						},
					},
				},
			},
			stepCtx: &StepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Images: []kargoapi.Image{
								{RepoURL: "docker.io/library/nginx", Tag: "1.19.0"},
							},
						},
					},
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			cfg: HelmUpdateImageConfig{
				Images: []Image{
					{Key: "image.tag", Image: "docker.io/library/nginx", Value: Tag},
				},
			},
			assertions: func(t *testing.T, changes map[string]string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"image.tag": "1.19.0"}, changes)
			},
		},
		{
			name: "image not found",
			stepCtx: &StepContext{
				Project:         "test-project",
				Freight:         kargoapi.FreightCollection{},
				FreightRequests: []kargoapi.FreightRequest{},
			},
			cfg: HelmUpdateImageConfig{
				Images: []Image{
					{Key: "image.tag", Image: "docker.io/library/non-existent", Value: Tag},
				},
			},
			assertions: func(t *testing.T, changes map[string]string, err error) {
				assert.NoError(t, err)
				assert.Empty(t, changes)
			},
		},
		{
			name: "multiple images, one not found",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "docker.io/library/nginx",
								},
							},
						},
					},
				},
			},
			stepCtx: &StepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Images: []kargoapi.Image{
								{RepoURL: "docker.io/library/nginx", Tag: "1.19.0"},
							},
						},
					},
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			cfg: HelmUpdateImageConfig{
				Images: []Image{
					{Key: "image1.tag", Image: "docker.io/library/nginx", Value: Tag},
					{Key: "image2.tag", Image: "docker.io/library/non-existent", Value: Tag},
				},
			},
			assertions: func(t *testing.T, changes map[string]string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"image1.tag": "1.19.0"}, changes)
			},
		},
		{
			name: "image with FromOrigin specified",
			objects: []client.Object{
				&kargoapi.Warehouse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-warehouse",
						Namespace: "test-project",
					},
					Spec: kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{
								Image: &kargoapi.ImageSubscription{
									RepoURL: "docker.io/library/origin-image",
								},
							},
						},
					},
				},
			},
			stepCtx: &StepContext{
				Project: "test-project",
				Freight: kargoapi.FreightCollection{
					Freight: map[string]kargoapi.FreightReference{
						"Warehouse/test-warehouse": {
							Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
							Images: []kargoapi.Image{
								{RepoURL: "docker.io/library/origin-image", Tag: "2.0.0"},
							},
						},
					},
				},
				FreightRequests: []kargoapi.FreightRequest{
					{
						Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			cfg: HelmUpdateImageConfig{
				Images: []Image{
					{
						Key:        "image.tag",
						Image:      "docker.io/library/origin-image",
						Value:      Tag,
						FromOrigin: &ChartFromOrigin{Kind: "Warehouse", Name: "test-warehouse"},
					},
				},
			},
			assertions: func(t *testing.T, changes map[string]string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"image.tag": "2.0.0"}, changes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))

			stepCtx := tt.stepCtx
			stepCtx.KargoClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			d := &helmUpdateImageDirective{}
			changes, err := d.generateImageUpdates(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, changes, err)
		})
	}
}

func Test_helmUpdateImageDirective_getDesiredOrigin(t *testing.T) {
	tests := []struct {
		name       string
		fromOrigin *ChartFromOrigin
		assertions func(*testing.T, *kargoapi.FreightOrigin)
	}{
		{
			name:       "nil origin",
			fromOrigin: nil,
			assertions: func(t *testing.T, origin *kargoapi.FreightOrigin) {
				assert.Nil(t, origin)
			},
		},
		{
			name: "valid origin",
			fromOrigin: &ChartFromOrigin{
				Kind: "Repository",
				Name: "test-repo",
			},
			assertions: func(t *testing.T, origin *kargoapi.FreightOrigin) {
				require.NotNil(t, origin)
				assert.Equal(t, kargoapi.FreightOriginKind("Repository"), origin.Kind)
				assert.Equal(t, "test-repo", origin.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &helmUpdateImageDirective{}
			origin := d.getDesiredOrigin(tt.fromOrigin)
			tt.assertions(t, origin)
		})
	}
}

func Test_helmUpdateImageDirective_getImageValue(t *testing.T) {
	tests := []struct {
		name       string
		image      *kargoapi.Image
		valueType  Value
		assertions func(*testing.T, string, error)
	}{
		{
			name: "image and tag",
			image: &kargoapi.Image{
				RepoURL: "docker.io/library/nginx",
				Tag:     "1.19",
			},
			valueType: ImageAndTag,
			assertions: func(t *testing.T, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "docker.io/library/nginx:1.19", value)
			},
		},
		{
			name: "tag only",
			image: &kargoapi.Image{
				Tag: "1.19",
			},
			valueType: Tag,
			assertions: func(t *testing.T, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "1.19", value)
			},
		},
		{
			name: "image and digest",
			image: &kargoapi.Image{
				RepoURL: "docker.io/library/nginx",
				Digest:  "sha256:abcdef1234567890",
			},
			valueType: ImageAndDigest,
			assertions: func(t *testing.T, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "docker.io/library/nginx@sha256:abcdef1234567890", value)
			},
		},
		{
			name: "digest only",
			image: &kargoapi.Image{
				Digest: "sha256:abcdef1234567890",
			},
			valueType: Digest,
			assertions: func(t *testing.T, value string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "sha256:abcdef1234567890", value)
			},
		},
		{
			name:      "unknown value type",
			image:     &kargoapi.Image{},
			valueType: "unknown",
			assertions: func(t *testing.T, value string, err error) {
				assert.Error(t, err)
				assert.Empty(t, value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &helmUpdateImageDirective{}
			value, err := d.getImageValue(tt.image, tt.valueType)
			tt.assertions(t, value, err)
		})
	}
}

func Test_helmUpdateImageDirective_updateValuesFile(t *testing.T) {
	tests := []struct {
		name          string
		valuesContent string
		changes       map[string]string
		assertions    func(*testing.T, string, error)
	}{
		{
			name:          "successful update",
			valuesContent: "key: value\n",
			changes:       map[string]string{"key": "newvalue"},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)

				require.FileExists(t, valuesFilePath)
				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "key: newvalue")
			},
		},
		{
			name:          "file does not exist",
			valuesContent: "",
			changes:       map[string]string{"key": "value"},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.ErrorContains(t, err, "no such file or directory")
				require.NoFileExists(t, valuesFilePath)
			},
		},
		{
			name:          "empty changes",
			valuesContent: "key: value\n",
			changes:       map[string]string{},
			assertions: func(t *testing.T, valuesFilePath string, err error) {
				require.NoError(t, err)
				require.FileExists(t, valuesFilePath)
				content, err := os.ReadFile(valuesFilePath)
				require.NoError(t, err)
				assert.Equal(t, "key: value\n", string(content))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			valuesFile := path.Join(workDir, "values.yaml")

			if tt.valuesContent != "" {
				err := os.WriteFile(valuesFile, []byte(tt.valuesContent), 0o600)
				require.NoError(t, err)
			}

			d := &helmUpdateImageDirective{}
			err := d.updateValuesFile(workDir, path.Base(valuesFile), tt.changes)
			tt.assertions(t, valuesFile, err)
		})
	}
}
