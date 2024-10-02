package directives

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
)

// kustomizeRenderMutex is a mutex that ensures only one kustomize build is
// running at a time. Required because of an ancient bug in Kustomize that
// causes it to concurrently read and write to the same map, causing a panic.
// xref: https://github.com/kubernetes-sigs/kustomize/issues/3659
var kustomizeRenderMutex sync.Mutex

func init() {
	builtins.RegisterPromotionStepRunner(newKustomizeBuilder(), nil)
}

// kustomizeBuilder is an implementation of the PromotionStepRunner interface
// that builds a set of Kubernetes manifests using Kustomize.
type kustomizeBuilder struct {
	schemaLoader gojsonschema.JSONLoader
}

// newKustomizeBuilder returns an implementation of the
// PromotionStepRunner interface that builds a set of Kubernetes manifests using
// Kustomize.
func newKustomizeBuilder() PromotionStepRunner {
	return &kustomizeBuilder{
		schemaLoader: getConfigSchemaLoader("kustomize-build"),
	}
}

// Name implements the PromotionStepRunner interface.
func (k *kustomizeBuilder) Name() string {
	return "kustomize-build"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (k *kustomizeBuilder) RunPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: PromotionStatusErrored}

	// Validate the configuration against the JSON Schema.
	if err := validate(k.schemaLoader, gojsonschema.NewGoLoader(stepCtx.Config), k.Name()); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed object.
	cfg, err := configToStruct[KustomizeBuildConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", k.Name(), err)
	}

	return k.runPromotionStep(stepCtx, cfg)
}

func (k *kustomizeBuilder) runPromotionStep(
	stepCtx *PromotionStepContext,
	cfg KustomizeBuildConfig,
) (PromotionStepResult, error) {
	// Create a "chrooted" filesystem for the kustomize build.
	fs, err := securefs.MakeFsOnDiskSecureBuild(stepCtx.WorkDir)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusErrored}, err
	}

	// Build the manifests.
	rm, err := kustomizeBuild(fs, filepath.Join(stepCtx.WorkDir, cfg.Path))
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusErrored}, err
	}

	// Prepare the output path.
	outPath, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.OutPath)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusErrored}, err
	}

	// Write the built manifests to the output path.
	if err := k.writeResult(rm, outPath); err != nil {
		return PromotionStepResult{Status: PromotionStatusErrored}, fmt.Errorf(
			"failed to write built manifests to %q: %w", cfg.OutPath,
			sanitizePathError(err, stepCtx.WorkDir),
		)
	}
	return PromotionStepResult{Status: PromotionStatusSuccess}, nil
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
