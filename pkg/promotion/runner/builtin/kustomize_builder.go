package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	securejoin "github.com/cyphar/filepath-securejoin"
	securefs "github.com/fluxcd/pkg/kustomize/filesys"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/io/fs"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindKustomizeBuild = "kustomize-build"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindKustomizeBuild,
			Value: newKustomizeBuilder,
		},
	)
}

// kustomizeRenderMutex is a mutex that ensures only one kustomize build is
// running at a time. Required because of an ancient bug in Kustomize that
// causes it to concurrently read and write to the same map, causing a panic.
// xref: https://github.com/kubernetes-sigs/kustomize/issues/3659
var kustomizeRenderMutex sync.Mutex

// kustomizeBuilder is an implementation of the promotion.StepRunner interface
// that builds a set of Kubernetes manifests using Kustomize.
type kustomizeBuilder struct {
	schemaLoader gojsonschema.JSONLoader
}

// newKustomizeBuilder returns an implementation of the
// promotion.StepRunner interface that builds a set of Kubernetes manifests using
// Kustomize.
func newKustomizeBuilder(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &kustomizeBuilder{
		schemaLoader: getConfigSchemaLoader(stepKindKustomizeBuild),
	}
}

// Run implements the promotion.StepRunner interface.
func (k *kustomizeBuilder) Run(
	_ context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := k.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return k.run(stepCtx, cfg)
}

// convert validates kustomizeBuilder configuration against a JSON schema and
// converts it into a builtin.KustomizeBuildConfig struct.
func (k *kustomizeBuilder) convert(cfg promotion.Config) (builtin.KustomizeBuildConfig, error) {
	return validateAndConvert[builtin.KustomizeBuildConfig](k.schemaLoader, cfg, stepKindKustomizeBuild)
}

func (k *kustomizeBuilder) run(
	stepCtx *promotion.StepContext,
	cfg builtin.KustomizeBuildConfig,
) (promotion.StepResult, error) {
	// Create a "chrooted" filesystem for the kustomize build.
	diskFS, err := securefs.MakeFsOnDiskSecureBuild(stepCtx.WorkDir)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Build the manifests.
	rm, err := kustomizeBuild(diskFS, filepath.Join(stepCtx.WorkDir, cfg.Path), cfg.Plugin)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Prepare the output path.
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Write the built manifests to the output path.
	if err := k.writeResult(rm, outPath); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"failed to write built manifests to %q: %w", cfg.OutPath,
			fs.SanitizePathError(err, stepCtx.WorkDir),
		)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func (k *kustomizeBuilder) writeResult(rm resmap.ResMap, outPath string) error {
	if ext := filepath.Ext(outPath); ext == ".yaml" || ext == ".yml" {
		if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			return err
		}
		b, err := rm.AsYaml()
		if err != nil {
			return err
		}
		return os.WriteFile(outPath, b, 0o600)
	}

	// If the output path is a directory, write each manifest to a separate file.
	if err := os.MkdirAll(outPath, 0o700); err != nil {
		return err
	}
	for _, r := range rm.Resources() {
		kind, namespace, name := r.GetKind(), r.GetNamespace(), r.GetName()
		if kind == "" || name == "" {
			return fmt.Errorf("resource kind and name of %q must be non-empty to write to a directory", r.CurId())
		}

		fileName := fmt.Sprintf("%s-%s", kind, name)
		if namespace != "" {
			fileName = fmt.Sprintf("%s-%s", namespace, fileName)
		}

		b, err := r.AsYAML()
		if err != nil {
			return fmt.Errorf("failed to convert %q to YAML: %w", r.CurId(), err)
		}

		path := filepath.Join(outPath, fmt.Sprintf("%s.yaml", strings.ToLower(fileName)))
		if err = os.WriteFile(path, b, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// kustomizeBuild builds the manifests in the given directory using Kustomize.
func kustomizeBuild(kusFS filesys.FileSystem, path string, pluginCfg *builtin.Plugin) (_ resmap.ResMap, err error) {
	kustomizeRenderMutex.Lock()
	defer kustomizeRenderMutex.Unlock()

	// Kustomize can panic in unpredicted ways due to (accidental)
	// invalid object data; recover when this happens to ensure
	// continuity of operations.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from kustomize build panic: %v", r)
		}
	}()

	// Disable plugins (i.e. "function based" plugins), but enable builtins
	// (e.g. transformers, generators).
	buildPluginCfg := kustypes.DisabledPluginConfig()
	// Helm plugin builtin requires explicit enabling. Kustomize itself ensures
	// the further Helm files (e.g. cache, data) are stored in a temporary
	// directory, AS LONG AS the global configuration is not set.
	buildPluginCfg.HelmConfig.Enabled = true
	buildPluginCfg.HelmConfig.Command = "helm"

	if pluginCfg != nil && pluginCfg.Helm != nil {
		buildPluginCfg.HelmConfig.ApiVersions = pluginCfg.Helm.APIVersions
		buildPluginCfg.HelmConfig.KubeVersion = pluginCfg.Helm.KubeVersion
	}

	buildOptions := &krusty.Options{
		// As we make use of a "chrooted" filesystem, we can safely allow
		// loading of files from anywhere.
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     buildPluginCfg,
	}

	k := krusty.MakeKustomizer(buildOptions)
	return k.Run(kusFS, path)
}
