package directives

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	securejoin "github.com/cyphar/filepath-securejoin"
	securefs "github.com/fluxcd/pkg/kustomize/filesys"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// kustomizeRenderMutex is a mutex that ensures only one kustomize build is
// running at a time. Required because of an ancient bug in Kustomize that
// causes it to concurrently read and write to the same map, causing a panic.
// xref: https://github.com/kubernetes-sigs/kustomize/issues/3659
var kustomizeRenderMutex sync.Mutex

func init() {
	// Register the kustomize-build directive with the builtins registry.
	builtins.RegisterDirective(newKustomizeBuildDirective(), nil)
}

// kustomizeBuildDirective is a directive that builds a set of Kubernetes
// manifests using Kustomize.
type kustomizeBuildDirective struct {
	schemaLoader gojsonschema.JSONLoader
}

// newKustomizeBuildDirective creates a new kustomize-build directive.
func newKustomizeBuildDirective() Directive {
	return &kustomizeBuildDirective{
		schemaLoader: getConfigSchemaLoader("kustomize-build"),
	}
}

// Name implements the Directive interface.
func (d *kustomizeBuildDirective) Name() string {
	return "kustomize-build"
}

// Run implements the Directive interface.
func (d *kustomizeBuildDirective) RunPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: PromotionStatusFailure}

	// Validate the configuration against the JSON Schema.
	if err := validate(d.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), d.Name()); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed object.
	cfg, err := configToStruct[KustomizeBuildConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", d.Name(), err)
	}

	return d.runPromotionStep(stepCtx, cfg)
}

// RunHealthCheckStep implements the Directive interface.
func (d *kustomizeBuildDirective) RunHealthCheckStep(
	context.Context,
	*HealthCheckStepContext,
) HealthCheckStepResult {
	return HealthCheckStepResult{Status: kargoapi.HealthStateNotApplicable}
}

func (d *kustomizeBuildDirective) runPromotionStep(
	stepCtx *PromotionStepContext,
	cfg KustomizeBuildConfig,
) (PromotionStepResult, error) {
	// Create a "chrooted" filesystem for the kustomize build.
	fs, err := securefs.MakeFsOnDiskSecureBuild(stepCtx.WorkDir)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}

	// Build the manifests.
	rm, err := kustomizeBuild(fs, filepath.Join(stepCtx.WorkDir, cfg.Path))
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}

	// Prepare the output path.
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	if err = os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}

	// Write the built manifests to the output path.
	b, err := rm.AsYaml()
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	if err = os.WriteFile(outPath, b, 0o600); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	return PromotionStepResult{Status: PromotionStatusSuccess}, nil
}

// kustomizeBuild builds the manifests in the given directory using Kustomize.
func kustomizeBuild(fs filesys.FileSystem, path string) (_ resmap.ResMap, err error) {
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

	buildOptions := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}

	k := krusty.MakeKustomizer(buildOptions)
	return k.Run(fs, path)
}
