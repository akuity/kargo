package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	securefs "github.com/fluxcd/pkg/kustomize/filesys"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
	kusPath, err := promotion.ExternalPath(stepCtx, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	rm, err := kustomizeBuild(diskFS, stepCtx.Root, kusPath, cfg.Plugin)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Prepare the output path.
	outPath, err := promotion.WorkspacePath(cfg.OutPath)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	// Write the built manifests to the output path.
	if err = k.writeResult(stepCtx.Root, rm, outPath, cfg.OutputFormat); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"failed to write built manifests to %q: %w", cfg.OutPath, err,
		)
	}
	return promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, nil
}

func (k *kustomizeBuilder) writeResult(
	root *os.Root,
	rm resmap.ResMap,
	outPath string,
	outputFmt *builtin.OutputFormat,
) error {
	if ext := filepath.Ext(outPath); ext == ".yaml" || ext == ".yml" {
		if err := root.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			return err
		}
		b, err := rm.AsYaml()
		if err != nil {
			return err
		}
		return root.WriteFile(outPath, b, 0o600)
	}

	// If the output path is a directory, write each manifest to a separate file.
	if err := root.MkdirAll(outPath, 0o700); err != nil {
		return err
	}

	// Write to the directory based on the configured format.
	format := builtin.Kargo
	if outputFmt != nil {
		format = *outputFmt
	}
	switch format {
	case builtin.Kargo:
		return k.writeIndividualFiles(root, outPath, rm, k.fileNameKargo)
	case builtin.Kustomize:
		return k.writeIndividualFiles(root, outPath, rm, k.fileNameKustomize)
	default:
		return fmt.Errorf("unsupported output format: %v", format)
	}
}

// kustomizeBuild builds the manifests in the given directory using Kustomize.
// Kustomization files and Helm chart inflation settings read during the build
// are validated against root; see kustomizeValidatorFS.
func kustomizeBuild(
	kusFS filesys.FileSystem,
	root *os.Root,
	path string,
	pluginCfg *builtin.Plugin,
) (_ resmap.ResMap, err error) {
	kustomizeRenderMutex.Lock()
	defer kustomizeRenderMutex.Unlock()

	// Kustomize stores custom OpenAPI schemas in package-level state. Reset it
	// before and after every build to isolate independent Promotion steps.
	openapi.ResetOpenAPI()
	defer openapi.ResetOpenAPI()

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

	// Reject remote entries whose Git ref would be interpreted by git as an
	// option. See kustomizeValidatorFS.
	validatingFS, err := newKustomizeValidatorFS(kusFS, root)
	if err != nil {
		return nil, err
	}
	k := krusty.MakeKustomizer(buildOptions)
	rm, err := k.Run(validatingFS, path)
	if err != nil {
		// Kustomize obscures validation errors (see kustomizeValidatorFS), so
		// report them in preference to whatever error Kustomize returned.
		if validationErr := validatingFS.Err(); validationErr != nil {
			return nil, validationErr
		}
		return nil, err
	}
	return rm, nil
}

// fileNameFunc generates a filename for a resource.
type fileNameFunc func(res *resource.Resource, multipleNamespaces bool) string

// writeIndividualFiles writes each resource to a separate file using the provided
// filename generator function.
//
// The iteration pattern is borrowed from Kustomize to ensure consistent behavior
// with `kustomize build -o dir/`.
//
// nolint:lll
// xref: https://github.com/kubernetes-sigs/kustomize/blob/17a06a72be7fa8e3fd50b5536c2fb32f8a4126cf/kustomize/commands/build/writer.go
//
// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0
func (k *kustomizeBuilder) writeIndividualFiles(
	root *os.Root,
	dirPath string,
	m resmap.ResMap,
	fileNameFn fileNameFunc,
) error {
	byNamespace := m.GroupedByCurrentNamespace()
	multiNs := len(byNamespace) > 1

	for _, resList := range byNamespace {
		for _, res := range resList {
			if err := k.writeResource(root, dirPath, fileNameFn(res, multiNs), res); err != nil {
				return err
			}
		}
	}
	for _, res := range m.ClusterScoped() {
		if err := k.writeResource(root, dirPath, fileNameFn(res, false), res); err != nil {
			return err
		}
	}
	return nil
}

func (k *kustomizeBuilder) writeResource(root *os.Root, path, fName string, res *resource.Resource) error {
	// Filenames are derived from rendered resource metadata, which is untrusted
	// input. Reject anything that isn't a plain filename so a malicious or
	// malformed resource cannot write outside path.
	if err := validateGeneratedResourceFilename(fName); err != nil {
		return err
	}
	m, err := res.Map()
	if err != nil {
		return err
	}
	yml, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return root.WriteFile(filepath.Join(path, fName), yml, 0o600)
}

// fileNameKargo generates filenames in Kargo format: [namespace-]kind-name.yaml.
// Namespace is always included when present on the resource.
func (k *kustomizeBuilder) fileNameKargo(res *resource.Resource, _ bool) string {
	kind, namespace, name := res.GetKind(), res.GetNamespace(), res.GetName()
	if namespace != "" {
		return strings.ToLower(fmt.Sprintf("%s-%s-%s.yaml", namespace, kind, name))
	}
	return strings.ToLower(fmt.Sprintf("%s-%s.yaml", kind, name))
}

// fileNameKustomize generates filenames in Kustomize format:
// [namespace_]group_version_kind_name.yaml.
// Namespace is only included when multiple namespaces are present in the
// resource map.
func (k *kustomizeBuilder) fileNameKustomize(res *resource.Resource, multipleNamespaces bool) string {
	base := strings.ToLower(res.GetGvk().StringWoEmptyField()) + "_" + strings.ToLower(res.GetName()) + ".yaml"
	if multipleNamespaces {
		namespace := res.GetNamespace()
		if namespace != "" {
			return strings.ToLower(namespace) + "_" + base
		}
	}
	return base
}
