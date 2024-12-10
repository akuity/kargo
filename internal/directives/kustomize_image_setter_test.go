package directives

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	kustypes "sigs.k8s.io/kustomize/api/types"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_kustomizeImageSetter_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           Config
		expectedProblems []string
	}{
		{
			name:   "path is not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): path is required",
			},
		},
		{
			name: "path is empty",
			config: Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "image not specified",
			config: Config{
				"images": []Config{{}},
			},
			expectedProblems: []string{
				"images.0: image is required",
			},
		},
		{
			name: "image is empty",
			config: Config{
				"images": []Config{{
					"image": "",
				}},
			},
			expectedProblems: []string{
				"images.0.image: String length must be greater than or equal to 1",
			},
		},
		{
			name: "digest and tag are both specified",
			// These should be mutually exclusive.
			config: Config{
				"images": []Config{{
					"digest": "fake-digest",
					"tag":    "fake-tag",
				}},
			},
			expectedProblems: []string{
				"images.0: Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "digest and useDigest are both specified",
			// These should be mutually exclusive.
			config: Config{
				"images": []Config{{
					"digest":    "fake-digest",
					"useDigest": true,
				}},
			},
			expectedProblems: []string{
				"images.0: Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "tag and useDigest are both specified",
			// These should be mutually exclusive.
			config: Config{
				"images": []Config{{
					"tag":       "fake-tag",
					"useDigest": true,
				}},
			},
			expectedProblems: []string{
				"images.0: Must validate one and only one schema (oneOf)",
			},
		},
		{
			name: "valid kitchen sink",
			config: Config{
				"path": "fake-path",
				"images": []Config{
					{
						"image": "fake-image-0",
					},
					{
						"image":     "fake-image-1",
						"digest":    "",
						"tag":       "",
						"useDigest": false,
					},
					{
						"image":  "fake-image-2",
						"digest": "fake-digest",
					},
					{
						"image":     "fake-image-3",
						"digest":    "fake-digest",
						"tag":       "",
						"useDigest": false,
					},
					{
						"image": "fake-image-4",
						"tag":   "fake-tag",
					},
					{
						"image":     "fake-image-5",
						"digest":    "",
						"tag":       "fake-tag",
						"useDigest": false,
					},
					{
						"image":     "fake-image-6",
						"useDigest": true,
					},
					{
						"image":     "fake-image-7",
						"digest":    "",
						"tag":       "",
						"useDigest": true,
					},
					{
						"image":     "fake-image-8",
						"useDigest": true,
						"fromOrigin": Config{
							"kind": Warehouse,
							"name": "fake-warehouse",
						},
					},
				},
			},
		},
	}

	r := newKustomizeImageSetter()
	runner, ok := r.(*kustomizeImageSetter)
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

func Test_kustomizeImageSetter_runPromotionStep(t *testing.T) {
	const testNamespace = "test-project-run"

	tests := []struct {
		name         string
		setupFiles   func(t *testing.T) string
		cfg          KustomizeSetImageConfig
		setupStepCtx func(t *testing.T, workDir string) *PromotionStepContext
		assertions   func(*testing.T, string, PromotionStepResult, error)
	}{
		{
			name: "successfully sets image",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()
				kustomizationContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`
				err := os.WriteFile(filepath.Join(tempDir, "kustomization.yaml"), []byte(kustomizationContent), 0o600)
				require.NoError(t, err)
				return tempDir
			},
			cfg: KustomizeSetImageConfig{
				Path: ".",
				Images: []KustomizeSetImageConfigImage{
					{Image: "nginx"},
				},
			},
			setupStepCtx: func(t *testing.T, workDir string) *PromotionStepContext {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
					mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
						Subscriptions: []kargoapi.RepoSubscription{
							{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
						},
					}),
				).Build()

				return &PromotionStepContext{
					WorkDir:     workDir,
					KargoClient: c,
					Project:     testNamespace,
					FreightRequests: []kargoapi.FreightRequest{
						{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: "Warehouse"}},
					},
					Freight: kargoapi.FreightCollection{
						Freight: map[string]kargoapi.FreightReference{
							"Warehouse/warehouse1": {
								Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse1"},
								Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:123"}},
							},
						},
					},
				}
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
					Output: map[string]any{
						"commitMessage": "Updated . to use new image\n\n- nginx:1.21.0",
					},
				}, result)

				b, err := os.ReadFile(filepath.Join(workDir, "kustomization.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "newTag: 1.21.0")
			},
		},
		{
			name: "automatically sets image",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()
				kustomizationContent := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`
				err := os.WriteFile(filepath.Join(tempDir, "kustomization.yaml"), []byte(kustomizationContent), 0o600)
				require.NoError(t, err)
				return tempDir
			},
			cfg: KustomizeSetImageConfig{
				Path:   ".",
				Images: nil, // Automatically set all images
			},
			setupStepCtx: func(_ *testing.T, workDir string) *PromotionStepContext {
				return &PromotionStepContext{
					WorkDir: workDir,
					Freight: kargoapi.FreightCollection{
						Freight: map[string]kargoapi.FreightReference{
							"Warehouse/warehouse1": {
								Images: []kargoapi.Image{{RepoURL: "nginx", Digest: "sha256:123"}},
							},
							"Warehouse/warehouse2": {
								Images: []kargoapi.Image{{RepoURL: "redis", Tag: "6.2.5"}},
							},
						},
					},
				}
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, PromotionStepResult{
					Status: kargoapi.PromotionPhaseSucceeded,
					Output: map[string]any{
						"commitMessage": "Updated . to use new images\n\n- nginx@sha256:123\n- redis:6.2.5",
					},
				}, result)

				b, err := os.ReadFile(filepath.Join(workDir, "kustomization.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(b), "newTag: 6.2.5")
				assert.Contains(t, string(b), "digest: sha256:123")
			},
		},
		{
			name: "Kustomization file not found",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: KustomizeSetImageConfig{
				Path: ".",
				Images: []KustomizeSetImageConfigImage{
					{Image: "nginx"},
				},
			},
			setupStepCtx: func(t *testing.T, workDir string) *PromotionStepContext {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				c := fake.NewClientBuilder().WithScheme(scheme).Build()

				return &PromotionStepContext{
					WorkDir:     workDir,
					KargoClient: c,
					Project:     testNamespace,
				}
			},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "could not discover kustomization file:")
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
			},
		},
		{
			name: "image origin not found",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()
				kustomizationContent := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
- name: nginx
  newTag: 1.19.0
`
				err := os.WriteFile(filepath.Join(tempDir, "kustomization.yaml"), []byte(kustomizationContent), 0o600)
				require.NoError(t, err)
				return tempDir
			},
			cfg: KustomizeSetImageConfig{
				Path: ".",
				Images: []KustomizeSetImageConfigImage{
					{Image: "nginx"},
				},
			},
			setupStepCtx: func(t *testing.T, workDir string) *PromotionStepContext {
				scheme := runtime.NewScheme()
				require.NoError(t, kargoapi.AddToScheme(scheme))
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

				return &PromotionStepContext{
					WorkDir:     workDir,
					KargoClient: fakeClient,
					Project:     testNamespace,
					FreightRequests: []kargoapi.FreightRequest{
						{Origin: kargoapi.FreightOrigin{Name: "non-existent-warehouse", Kind: "Warehouse"}},
					},
				}
			},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				require.ErrorContains(t, err, "unable to discover image")
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
			},
		},
	}

	runner := &kustomizeImageSetter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)
			stepCtx := tt.setupStepCtx(t, workDir)

			result, err := runner.runPromotionStep(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_kustomizeImageSetter_buildTargetImages(t *testing.T) {
	const testNamespace = "test-project"

	tests := []struct {
		name              string
		images            []KustomizeSetImageConfigImage
		freightRequests   []kargoapi.FreightRequest
		objects           []runtime.Object
		freightReferences map[string]kargoapi.FreightReference
		assertions        func(*testing.T, map[string]kustypes.Image, error)
	}{
		{
			name: "digest or tag specified",
			images: []KustomizeSetImageConfigImage{
				{
					Image: "nginx",
					Tag:   "fake-tag",
				},
				{
					Image:  "redis",
					Digest: "fake-digest",
				},
			},
			assertions: func(t *testing.T, result map[string]kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]kustypes.Image{
					"nginx": {Name: "nginx", NewTag: "fake-tag"},
					"redis": {Name: "redis", Digest: "fake-digest"},
				}, result)
			},
		},
		{
			name: "discovers origins and builds target images",
			images: []KustomizeSetImageConfigImage{
				{Image: "nginx"},
				{Image: "redis"},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: "Warehouse"}},
				{Origin: kargoapi.FreightOrigin{Name: "warehouse2", Kind: "Warehouse"}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
				mockWarehouse(testNamespace, "warehouse2", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "redis"}},
					},
				}),
			},
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse1"},
					Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:123"}},
				},
				"Warehouse/warehouse2": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse2"},
					Images: []kargoapi.Image{{RepoURL: "redis", Tag: "6.2.5", Digest: "sha256:456"}},
				},
			},
			assertions: func(t *testing.T, result map[string]kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]kustypes.Image{
					"nginx": {Name: "nginx", NewTag: "1.21.0"},
					"redis": {Name: "redis", NewTag: "6.2.5"},
				}, result)
			},
		},
		{
			name: "error when no origin found",
			images: []KustomizeSetImageConfigImage{
				{Image: "mysql"},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: "Warehouse"}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
			},
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse1"},
					Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:123"}},
				},
			},
			assertions: func(t *testing.T, _ map[string]kustypes.Image, err error) {
				require.ErrorContains(t, err, "not found in referenced Freight")
			},
		},
		{
			name: "uses provided origin",
			images: []KustomizeSetImageConfigImage{
				{Image: "nginx", FromOrigin: &ChartFromOrigin{Kind: "Warehouse", Name: "warehouse1"}},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: "Warehouse"}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
			},
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse1"},
					Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:123"}},
				},
			},
			assertions: func(t *testing.T, result map[string]kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]kustypes.Image{
					"nginx": {Name: "nginx", NewTag: "1.21.0"},
				}, result)
			},
		},
		{
			name: "uses custom name and digest",
			images: []KustomizeSetImageConfigImage{
				{
					Image:     "nginx",
					Name:      "custom-nginx",
					UseDigest: true,
					FromOrigin: &ChartFromOrigin{
						Kind: "Warehouse",
						Name: "warehouse1",
					},
				},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: "Warehouse"}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
			},
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse1"},
					Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:123"}},
				},
			},
			assertions: func(t *testing.T, result map[string]kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]kustypes.Image{
					"custom-nginx": {Name: "custom-nginx", NewTag: "1.21.0", Digest: "sha256:123"},
				}, result)
			},
		},
		{
			name: "error when multiple origins found",
			images: []KustomizeSetImageConfigImage{
				{Image: "nginx"},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: "Warehouse"}},
				{Origin: kargoapi.FreightOrigin{Name: "warehouse2", Kind: "Warehouse"}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
				mockWarehouse(testNamespace, "warehouse2", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
			},
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse1"},
					Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:123"}},
				},
				"Warehouse/warehouse2": {
					Origin: kargoapi.FreightOrigin{Kind: "Warehouse", Name: "warehouse2"},
					Images: []kargoapi.Image{{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:456"}},
				},
			},
			assertions: func(t *testing.T, _ map[string]kustypes.Image, err error) {
				require.ErrorContains(t, err, "multiple requested Freight could potentially provide a container image")
			},
		},
	}

	runner := &kustomizeImageSetter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			stepCtx := &PromotionStepContext{
				KargoClient:     fakeClient,
				Project:         testNamespace,
				FreightRequests: tt.freightRequests,
				Freight: kargoapi.FreightCollection{
					Freight: tt.freightReferences,
				},
			}

			result, err := runner.buildTargetImagesFromConfig(context.Background(), stepCtx, tt.images)
			tt.assertions(t, result, err)
		})
	}
}

func Test_kustomizeImageSetter_buildTargetImagesAutomatically(t *testing.T) {
	const testNamespace = "test-project"

	tests := []struct {
		name              string
		freightReferences map[string]kargoapi.FreightReference
		freightRequests   []kargoapi.FreightRequest
		objects           []runtime.Object
		assertions        func(*testing.T, map[string]kustypes.Image, error)
	}{
		{
			name: "successfully builds target images",
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Images: []kargoapi.Image{
						{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:abcdef1234567890"},
					},
				},
				"Warehouse/warehouse2": {
					Images: []kargoapi.Image{
						{RepoURL: "redis", Tag: "6.2.5"},
					},
				},
				"Warehouse/warehouse3": {
					Images: []kargoapi.Image{
						{RepoURL: "postgres", Digest: "sha256:abcdef1234567890"},
					},
				},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: kargoapi.FreightOriginKindWarehouse}},
				{Origin: kargoapi.FreightOrigin{Name: "warehouse2", Kind: kargoapi.FreightOriginKindWarehouse}},
				{Origin: kargoapi.FreightOrigin{Name: "warehouse3", Kind: kargoapi.FreightOriginKindWarehouse}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
				mockWarehouse(testNamespace, "warehouse2", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "redis"}},
					},
				}),
				mockWarehouse(testNamespace, "warehouse3", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "postgres"}},
					},
				}),
			},
			assertions: func(t *testing.T, result map[string]kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Equal(t, map[string]kustypes.Image{
					"nginx":    {Name: "nginx", NewTag: "1.21.0", Digest: "sha256:abcdef1234567890"},
					"redis":    {Name: "redis", NewTag: "6.2.5"},
					"postgres": {Name: "postgres", Digest: "sha256:abcdef1234567890"},
				}, result)
			},
		},
		{
			name: "error on ambiguous image match",
			freightReferences: map[string]kargoapi.FreightReference{
				"Warehouse/warehouse1": {
					Images: []kargoapi.Image{
						{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:abcdef1234567890"},
					},
				},
				"Warehouse/warehouse2": {
					Images: []kargoapi.Image{
						{RepoURL: "nginx", Tag: "1.21.0", Digest: "sha256:abcdef1234567890"},
					},
				},
			},
			freightRequests: []kargoapi.FreightRequest{
				{Origin: kargoapi.FreightOrigin{Name: "warehouse1", Kind: kargoapi.FreightOriginKindWarehouse}},
				{Origin: kargoapi.FreightOrigin{Name: "warehouse2", Kind: kargoapi.FreightOriginKindWarehouse}},
			},
			objects: []runtime.Object{
				mockWarehouse(testNamespace, "warehouse1", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
				mockWarehouse(testNamespace, "warehouse2", kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{Image: &kargoapi.ImageSubscription{RepoURL: "nginx"}},
					},
				}),
			},
			assertions: func(t *testing.T, _ map[string]kustypes.Image, err error) {
				require.ErrorContains(t, err, "manual configuration required due to ambiguous result")
			},
		},
	}

	runner := &kustomizeImageSetter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			stepCtx := &PromotionStepContext{
				KargoClient:     fakeClient,
				Project:         testNamespace,
				FreightRequests: tt.freightRequests,
				Freight: kargoapi.FreightCollection{
					Freight: tt.freightReferences,
				},
			}

			result, err := runner.buildTargetImagesAutomatically(context.Background(), stepCtx)
			tt.assertions(t, result, err)
		})
	}
}

func Test_kustomizeImageSetter_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		images     map[string]kustypes.Image
		assertions func(*testing.T, string)
	}{
		{
			name:   "empty images",
			path:   "path/to/kustomization",
			images: map[string]kustypes.Image{},
			assertions: func(t *testing.T, got string) {
				assert.Empty(t, got)
			},
		},
		{
			name: "single image update",
			path: "path/to/kustomization",
			images: map[string]kustypes.Image{
				"image1": {Name: "nginx", NewTag: "1.19"},
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated path/to/kustomization to use new image")
				assert.Contains(t, got, "- nginx:1.19")
				assert.Equal(t, 2, strings.Count(got, "\n"))
			},
		},
		{
			name: "multiple image updates",
			path: "path/to/kustomization",
			images: map[string]kustypes.Image{
				"image1": {Name: "nginx", NewTag: "1.19"},
				"image2": {Name: "redis", NewTag: "6.0"},
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated path/to/kustomization to use new images")
				assert.Contains(t, got, "- nginx:1.19")
				assert.Contains(t, got, "- redis:6.0")
				assert.Equal(t, 3, strings.Count(got, "\n"))
			},
		},
		{
			name: "image update with new name",
			path: "path/to/kustomization",
			images: map[string]kustypes.Image{
				"image1": {Name: "nginx", NewName: "custom-nginx", NewTag: "1.19"},
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated path/to/kustomization to use new image")
				assert.Contains(t, got, "- custom-nginx:1.19")
				assert.Equal(t, 2, strings.Count(got, "\n"))
			},
		},
		{
			name: "image update with digest",
			path: "path/to/kustomization",
			images: map[string]kustypes.Image{
				"image1": {Name: "nginx", Digest: "sha256:abcdef1234567890"},
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated path/to/kustomization to use new image")
				assert.Contains(t, got, "- nginx@sha256:abcdef1234567890")
				assert.Equal(t, 2, strings.Count(got, "\n"))
			},
		},
		{
			name: "mixed image updates",
			path: "path/to/kustomization",
			images: map[string]kustypes.Image{
				"image1": {Name: "nginx", NewTag: "1.19"},
				"image2": {Name: "redis", NewName: "custom-redis", NewTag: "6.0"},
				"image3": {Name: "postgres", Digest: "sha256:abcdef1234567890"},
			},
			assertions: func(t *testing.T, got string) {
				assert.Contains(t, got, "Updated path/to/kustomization to use new images")
				assert.Contains(t, got, "- nginx:1.19")
				assert.Contains(t, got, "- custom-redis:6.0")
				assert.Contains(t, got, "- postgres@sha256:abcdef1234567890")
				assert.Equal(t, 4, strings.Count(got, "\n"))
			},
		},
	}

	runner := &kustomizeImageSetter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runner.generateCommitMessage(tt.path, tt.images)
			tt.assertions(t, got)
		})
	}
}

func Test_updateKustomizationFile(t *testing.T) {
	tests := []struct {
		name         string
		initialYAML  string
		targetImages map[string]kustypes.Image
		assertions   func(*testing.T, string, error)
	}{
		{
			name: "update existing images",
			initialYAML: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
- name: nginx
  newTag: 1.19.0
`,
			targetImages: map[string]kustypes.Image{
				"nginx": {Name: "nginx", NewTag: "1.21.0"},
			},
			assertions: func(t *testing.T, kusPath string, err error) {
				require.NoError(t, err)

				b, readErr := os.ReadFile(kusPath)
				require.NoError(t, readErr)

				var node yaml.Node
				require.NoError(t, yaml.Unmarshal(b, &node))

				images, getErr := getCurrentImages(&node)
				require.NoError(t, getErr)

				assert.Len(t, images, 1)
				assert.Equal(t, "nginx", images[0].Name)
				assert.Equal(t, "1.21.0", images[0].NewTag)
			},
		},
		{
			name: "add new image",
			initialYAML: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`,
			targetImages: map[string]kustypes.Image{
				"nginx": {Name: "nginx", NewTag: "1.21.0"},
			},
			assertions: func(t *testing.T, kusPath string, err error) {
				assert.NoError(t, err)

				b, err := os.ReadFile(kusPath)
				require.NoError(t, err)

				var node yaml.Node
				require.NoError(t, yaml.Unmarshal(b, &node))

				images, getErr := getCurrentImages(&node)
				require.NoError(t, getErr)

				assert.Len(t, images, 1)
				assert.Equal(t, "nginx", images[0].Name)
				assert.Equal(t, "1.21.0", images[0].NewTag)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			kusPath := filepath.Join(tmpDir, "kustomization.yaml")
			err := os.WriteFile(kusPath, []byte(tt.initialYAML), 0o600)
			require.NoError(t, err)

			err = updateKustomizationFile(kusPath, tt.targetImages)
			tt.assertions(t, kusPath, err)
		})
	}
}

func Test_readKustomizationFile(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		assertions func(*testing.T, *yaml.Node, error)
	}{
		{
			name: "valid YAML",
			content: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
- name: nginx
  newTag: 1.21.0
`,
			assertions: func(t *testing.T, node *yaml.Node, err error) {
				require.NoError(t, err)
				assert.NotNil(t, node)
				assert.Equal(t, yaml.DocumentNode, node.Kind)
				assert.Len(t, node.Content, 1)
			},
		},
		{
			name: "invalid YAML",
			content: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
- name: nginx
  newTag: 1.21.0
  - invalid
`,
			assertions: func(t *testing.T, node *yaml.Node, err error) {
				require.Error(t, err)
				assert.Nil(t, node)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			kusPath := filepath.Join(tmpDir, "kustomization.yaml")
			require.NoError(t, os.WriteFile(kusPath, []byte(tt.content), 0o600))

			node, err := readKustomizationFile(kusPath)
			tt.assertions(t, node, err)
		})
	}
}

func Test_getCurrentImages(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		assertions func(*testing.T, []kustypes.Image, error)
	}{
		{
			name: "valid images field",
			yaml: `images:
- name: nginx
  newTag: 1.21.0
`,
			assertions: func(t *testing.T, images []kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Len(t, images, 1)
				assert.Equal(t, "nginx", images[0].Name)
				assert.Equal(t, "1.21.0", images[0].NewTag)
			},
		},
		{
			name: "no images field",
			yaml: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`,
			assertions: func(t *testing.T, images []kustypes.Image, err error) {
				require.NoError(t, err)
				assert.Empty(t, images)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			images, err := getCurrentImages(&node)
			tt.assertions(t, images, err)
		})
	}
}

func Test_mergeImages(t *testing.T) {
	tests := []struct {
		name          string
		currentImages []kustypes.Image
		targetImages  map[string]kustypes.Image
		assertions    func(*testing.T, []kustypes.Image)
	}{
		{
			name: "merge new and existing images",
			currentImages: []kustypes.Image{
				{Name: "nginx", NewTag: "1.19.0"},
			},
			targetImages: map[string]kustypes.Image{
				"nginx": {Name: "nginx", NewTag: "1.21.0"},
				"redis": {Name: "redis", NewTag: "6.2.5"},
			},
			assertions: func(t *testing.T, merged []kustypes.Image) {
				assert.Len(t, merged, 2)
				assert.Equal(t, []kustypes.Image{
					{Name: "nginx", NewTag: "1.21.0"},
					{Name: "redis", NewTag: "6.2.5"},
				}, merged)
			},
		},
		{
			name: "preserve existing images not in target",
			currentImages: []kustypes.Image{
				{Name: "nginx", NewTag: "1.19.0"},
				{Name: "mysql", NewTag: "8.0.0"},
			},
			targetImages: map[string]kustypes.Image{
				"nginx": {Name: "nginx", NewTag: "1.21.0"},
			},
			assertions: func(t *testing.T, merged []kustypes.Image) {
				assert.Len(t, merged, 2)
				assert.Equal(t, []kustypes.Image{
					{Name: "mysql", NewTag: "8.0.0"},
					{Name: "nginx", NewTag: "1.21.0"},
				}, merged)
			},
		},
		{
			name: "handle asterisk separator",
			currentImages: []kustypes.Image{
				{Name: "nginx", NewName: "custom-nginx", NewTag: "1.19.0"},
			},
			targetImages: map[string]kustypes.Image{
				"nginx": {Name: "nginx", NewName: preserveSeparator, NewTag: "1.21.0"},
			},
			assertions: func(t *testing.T, merged []kustypes.Image) {
				assert.Len(t, merged, 1)
				assert.Equal(t, []kustypes.Image{
					{Name: "nginx", NewName: "custom-nginx", NewTag: "1.21.0"},
				}, merged)
			},
		},
		{
			name: "sort images by name",
			currentImages: []kustypes.Image{
				{Name: "nginx", NewTag: "1.19.0"},
				{Name: "mysql", NewTag: "8.0.0"},
			},
			targetImages: map[string]kustypes.Image{
				"redis": {Name: "redis", NewTag: "6.2.5"},
			},
			assertions: func(t *testing.T, merged []kustypes.Image) {
				assert.Len(t, merged, 3)
				assert.Equal(t, "mysql", merged[0].Name)
				assert.Equal(t, "nginx", merged[1].Name)
				assert.Equal(t, "redis", merged[2].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := mergeImages(tt.currentImages, tt.targetImages)
			tt.assertions(t, merged)
		})
	}
}

func Test_writeKustomizationFile(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (string, *yaml.Node)
		assertions func(*testing.T, string, error)
	}{
		{
			name: "write valid Kustomization file",
			setup: func(t *testing.T) (string, *yaml.Node) {
				dir := t.TempDir()
				kusPath := filepath.Join(dir, "kustomization.yaml")
				node := &yaml.Node{
					Kind: yaml.DocumentNode,
					Content: []*yaml.Node{
						{
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Value: "apiVersion"},
								{Kind: yaml.ScalarNode, Value: "kustomize.config.k8s.io/v1beta1"},
								{Kind: yaml.ScalarNode, Value: "kind"},
								{Kind: yaml.ScalarNode, Value: "Kustomization"},
							},
						},
					},
				}
				return kusPath, node
			},
			assertions: func(t *testing.T, kusPath string, err error) {
				require.NoError(t, err)

				assert.FileExists(t, kusPath)

				b, _ := os.ReadFile(kusPath)
				assert.Contains(t, string(b), "apiVersion: kustomize.config.k8s.io/v1beta1")
				assert.Contains(t, string(b), "kind: Kustomization")
			},
		},
		{
			name: "write to non-existent directory",
			setup: func(t *testing.T) (string, *yaml.Node) {
				dir := t.TempDir()
				kusPath := filepath.Join(dir, "non-existent-dir", "kustomization.yaml")
				node := &yaml.Node{
					Kind: yaml.DocumentNode,
					Content: []*yaml.Node{
						{Kind: yaml.MappingNode, Content: []*yaml.Node{}},
					},
				}
				return kusPath, node
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "could not write updated Kustomization file")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kusPath, node := tt.setup(t)
			err := writeKustomizationFile(kusPath, node)
			tt.assertions(t, kusPath, err)
		})
	}
}

func Test_findKustomization(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (workDir string, cleanup func())
		path       string
		assertions func(*testing.T, string, error)
	}{
		{
			name: "single kustomization.yaml file",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				err := os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte{}, 0o600)
				require.NoError(t, err)
				return dir, func() {}
			},
			assertions: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "kustomization.yaml", filepath.Base(result))
			},
		},
		{
			name: "single Kustomization file",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				err := os.WriteFile(filepath.Join(dir, "Kustomization"), []byte{}, 0o600)
				require.NoError(t, err)
				return dir, func() {}
			},
			assertions: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "Kustomization", filepath.Base(result))
			},
		},
		{
			name: "multiple Kustomization files",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte{}, 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "Kustomization"), []byte{}, 0o600))
				return dir, func() {}
			},
			path: ".",
			assertions: func(t *testing.T, result string, err error) {
				require.ErrorContains(t, err, "ambiguous result")
				assert.Empty(t, result)
			},
		},
		{
			name: "no Kustomization files",
			setup: func(t *testing.T) (string, func()) {
				return t.TempDir(), func() {}
			},
			path: ".",
			assertions: func(t *testing.T, result string, err error) {
				require.ErrorContains(t, err, "could not find any Kustomization files")
				assert.Empty(t, result)
			},
		},
		{
			name: "Kustomization file in subdirectory",
			setup: func(t *testing.T) (string, func()) {
				dir := t.TempDir()
				subdir := filepath.Join(dir, "subdir")
				assert.NoError(t, os.Mkdir(subdir, 0755))
				assert.NoError(t, os.WriteFile(filepath.Join(subdir, "kustomization.yaml"), []byte{}, 0o600))
				return dir, func() {}
			},
			path: "subdir",
			assertions: func(t *testing.T, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "kustomization.yaml", filepath.Base(result))
				assert.Contains(t, result, "subdir")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir, cleanup := tt.setup(t)
			defer cleanup()

			result, err := findKustomization(workDir, tt.path)
			tt.assertions(t, result, err)
		})
	}
}

func mockWarehouse(namespace, name string, spec kargoapi.WarehouseSpec) *kargoapi.Warehouse {
	return &kargoapi.Warehouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Warehouse",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
	}
}
