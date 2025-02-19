package rollouts

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/expressions"
)

// controllerInstanceIDLabelKey is the key for the Argo Rollouts controller
// instance ID label. It can be used to assign an AnalysisRun to a specific
// controller instance.
const controllerInstanceIDLabelKey = "argo-rollouts.argoproj.io/controller-instance-id"

// Config holds the configuration for the AnalysisRunBuilder.
type Config struct {
	// ControllerInstanceID is the unique identifier for the Argo Rollouts
	// controller instance. If set, any AnalysisRun created by the builder
	// will have this value set as a label.
	ControllerInstanceID string
}

// AnalysisRunBuilder constructs AnalysisRun objects with consistent configuration.
type AnalysisRunBuilder struct {
	client client.Client
	cfg    Config
}

// NewAnalysisRunBuilder creates a new AnalysisRunBuilder with the provided
// client and configuration.
func NewAnalysisRunBuilder(c client.Client, cfg Config) *AnalysisRunBuilder {
	return &AnalysisRunBuilder{
		client: c,
		cfg:    cfg,
	}
}

// Build creates a new AnalysisRun from the provided verification and options.
func (b *AnalysisRunBuilder) Build(
	ctx context.Context,
	namespace string,
	cfg *kargoapi.Verification,
	opt ...AnalysisRunOption,
) (*rolloutsapi.AnalysisRun, error) {
	opts := &AnalysisRunOptions{}
	opts.Apply(opt...)

	if cfg == nil {
		return nil, errors.New("missing verification configuration")
	}

	metadata := b.buildMetadata(
		namespace,
		b.generateName(opts.NamePrefix, opts.NameSuffix),
		cfg.AnalysisRunMetadata,
		opts.ExtraLabels,
		opts.ExtraAnnotations,
	)

	templates, err := b.getAnalysisTemplates(
		ctx,
		namespace,
		cfg.AnalysisTemplates,
	)
	if err != nil {
		return nil, fmt.Errorf("get analysis templates: %w", err)
	}

	spec, err := b.buildSpec(templates, cfg.Args, opts.ExpressionConfig)
	if err != nil {
		return nil, fmt.Errorf("build spec: %w", err)
	}

	ownerRefs, err := b.buildOwnerReferences(ctx, opts.Owners)
	if err != nil {
		return nil, fmt.Errorf("build owner references: %w", err)
	}

	obj := &rolloutsapi.AnalysisRun{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	obj.SetOwnerReferences(ownerRefs)

	return obj, nil
}

// generateName creates a unique name for an AnalysisRun by combining the prefix,
// a ULID, and an optional suffix.
func (b *AnalysisRunBuilder) generateName(prefix, suffix string) string {
	var parts []string

	if prefix != "" {
		parts = append(parts, prefix)
	}

	parts = append(parts, ulid.Make().String())

	if suffix != "" {
		parts = append(parts, suffix)
	}

	return strings.ToLower(strings.Join(parts, "."))
}

// buildMetadata creates an ObjectMeta for an AnalysisRun, combining metadata
// from multiple sources.
func (b *AnalysisRunBuilder) buildMetadata(
	namespace, name string,
	metadata *kargoapi.AnalysisRunMetadata,
	extraLabels map[string]string,
	extraAnnotations map[string]string,
) metav1.ObjectMeta {
	annotations := make(map[string]string)
	labels := make(map[string]string)

	if metadata != nil {
		if metadata.Annotations != nil {
			maps.Copy(annotations, metadata.Annotations)
		}
		if metadata.Labels != nil {
			maps.Copy(labels, metadata.Labels)
		}
	}

	if extraAnnotations != nil {
		maps.Copy(annotations, extraAnnotations)
	}

	if extraLabels != nil {
		maps.Copy(labels, extraLabels)
	}

	if id := b.cfg.ControllerInstanceID; id != "" {
		labels[controllerInstanceIDLabelKey] = id
	}

	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
	}
}

// buildSpec constructs an AnalysisRunSpec from the provided templates and
// arguments.
func (b *AnalysisRunBuilder) buildSpec(
	templates []*rolloutsapi.AnalysisTemplate,
	args []kargoapi.AnalysisRunArgument,
	exprCfg *ArgumentEvaluationConfig,
) (rolloutsapi.AnalysisRunSpec, error) {
	template, err := flattenTemplates(templates)
	if err != nil {
		return rolloutsapi.AnalysisRunSpec{}, fmt.Errorf("flatten templates: %w", err)
	}

	finalArgs, err := b.buildArgs(template, args, exprCfg)
	if err != nil {
		return rolloutsapi.AnalysisRunSpec{}, fmt.Errorf("build arguments: %w", err)
	}

	return rolloutsapi.AnalysisRunSpec{
		Metrics:              template.Spec.Metrics,
		DryRun:               template.Spec.DryRun,
		MeasurementRetention: template.Spec.MeasurementRetention,
		Args:                 finalArgs,
	}, nil
}

// buildArgs converts analysis run arguments to rollouts arguments and merges them
// with template arguments.
func (b *AnalysisRunBuilder) buildArgs(
	template *rolloutsapi.AnalysisTemplate,
	args []kargoapi.AnalysisRunArgument,
	exprCfg *ArgumentEvaluationConfig,
) ([]rolloutsapi.Argument, error) {
	if exprCfg == nil {
		exprCfg = &ArgumentEvaluationConfig{}
	}

	rolloutsArgs := make([]rolloutsapi.Argument, len(args))
	for i, arg := range args {
		rolloutsArgs[i] = rolloutsapi.Argument{
			Name: arg.Name,
		}
		if arg.Value != "" {
			value, err := expressions.EvaluateTemplate(arg.Value, exprCfg.Env, exprCfg.Options...)
			if err != nil {
				return nil, fmt.Errorf("evaluate argument %q: %w", arg.Name, err)
			}
			strValue, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("evaluated argument %q value is not a string but %T", arg.Name, value)
			}
			rolloutsArgs[i].Value = &strValue
		}
	}

	mergedArgs, err := mergeArgs(rolloutsArgs, template.Spec.Args)
	if err != nil {
		return nil, fmt.Errorf("merge arguments: %w", err)
	}

	return mergedArgs, nil
}

// buildOwnerReferences creates owner references for the specified owners by
// fetching their current state from the cluster.
func (b *AnalysisRunBuilder) buildOwnerReferences(
	ctx context.Context,
	owners []Owner,
) ([]metav1.OwnerReference, error) {
	refs := make([]metav1.OwnerReference, 0, len(owners))

	for _, owner := range owners {
		obj := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": owner.APIVersion,
				"kind":       owner.Kind,
			},
		}

		if err := b.client.Get(ctx, owner.Reference, &obj); err != nil {
			return nil, fmt.Errorf(
				"get %s %q in namespace %q: %w",
				owner.Kind,
				owner.Reference.Name,
				owner.Reference.Namespace,
				err,
			)
		}

		refs = append(refs, metav1.OwnerReference{
			APIVersion:         obj.GetAPIVersion(),
			Kind:               obj.GetKind(),
			Name:               obj.GetName(),
			UID:                obj.GetUID(),
			BlockOwnerDeletion: ptr.To(owner.BlockDeletion),
		})
	}

	return refs, nil
}

// getAnalysisTemplates retrieves all referenced analysis templates from the
// cluster.
func (b *AnalysisRunBuilder) getAnalysisTemplates(
	ctx context.Context,
	namespace string,
	references []kargoapi.AnalysisTemplateReference,
) ([]*rolloutsapi.AnalysisTemplate, error) {
	templates := make([]*rolloutsapi.AnalysisTemplate, len(references))

	for i, ref := range references {
		template := &rolloutsapi.AnalysisTemplate{}
		if err := b.client.Get(ctx, types.NamespacedName{
			Namespace: namespace,
			Name:      ref.Name,
		}, template); err != nil {
			return nil, fmt.Errorf(
				"get AnalysisRun %q in namespace %q: %w",
				ref.Name,
				namespace,
				err,
			)
		}
		templates[i] = template
	}

	return templates, nil
}
