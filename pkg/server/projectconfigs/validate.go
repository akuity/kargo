package projectconfigs

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/pattern"
	"github.com/akuity/kargo/pkg/webhook/kubernetes/external"
)

// The constants below mirror constraints declared on the ProjectConfig CRD
// schema (kubebuilder markers in api/v1alpha1). They are hand-written here
// because ProjectConfigs are validated by the API server instead of by
// Kubernetes. TestValidateParity checks them against the CRD.
const (
	// kubernetesNamePattern is the pattern the CRD applies to Stage names,
	// webhook receiver names, and promotion window names.
	kubernetesNamePattern = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`

	maxNameLength              = 253
	maxWindowDescriptionLength = 1024
	maxAutoRollbackPhases      = 2

	policyOneOfMessage = "PromotionPolicy must have exactly one of stage or " +
		"stageSelector set"
)

var kubernetesNameRegex = regexp.MustCompile(kubernetesNamePattern)

// Validate returns field errors at the same paths the admission webhook and
// the CRD schema report today. create is true on create; the name and
// namespace rules apply only then, as they do today. Labels and annotations
// are checked on every write, as the Kubernetes API server checks them. Spec
// validators registered in
// DefaultSpecValidatorRegistry run after the built-in rules, and only when
// those rules pass.
func Validate(
	ctx context.Context,
	project string,
	cfg *kargoapi.ProjectConfig,
	create bool,
) field.ErrorList {
	return validate(ctx, project, cfg, create, DefaultSpecValidatorRegistry)
}

func validate(
	ctx context.Context,
	project string,
	cfg *kargoapi.ProjectConfig,
	create bool,
	registry SpecValidatorRegistry,
) field.ErrorList {
	var errs field.ErrorList
	meta := field.NewPath("metadata")
	if create {
		errs = append(errs, validateObjectMeta(meta, project, cfg)...)
	}
	errs = append(errs, validateLabelsAndAnnotations(meta, cfg)...)
	errs = append(errs, validateSpec(field.NewPath("spec"), &cfg.Spec)...)
	if len(errs) > 0 {
		// Registered validators may rely on the built-in rules having passed,
		// just as webhooks today only see objects the CRD schema accepted.
		return errs
	}
	for _, reg := range registry.List() {
		errs = append(errs, reg.Value(ctx, project, &cfg.Spec)...)
	}
	return errs
}

// validateObjectMeta ports the admission webhook's name rule and models the
// Kubernetes API server's check that the body's namespace matches the request.
func validateObjectMeta(
	f *field.Path,
	project string,
	cfg *kargoapi.ProjectConfig,
) field.ErrorList {
	var errs field.ErrorList
	if cfg.Name != project {
		errs = append(errs, field.Invalid(
			f.Child("name"),
			cfg.Name,
			fmt.Sprintf("name %q must match project name %q", cfg.Name, project),
		))
	}
	if cfg.Namespace != "" && cfg.Namespace != project {
		errs = append(errs, field.Invalid(
			f.Child("namespace"),
			cfg.Namespace,
			fmt.Sprintf(
				"namespace %q must be empty or match project name %q",
				cfg.Namespace,
				project,
			),
		))
	}
	return errs
}

// validateLabelsAndAnnotations ports the Kubernetes API server's checks of
// label keys and values, annotation keys, and the total size of annotations.
// It checks the annotations as sent, including any refresh annotation that
// is dropped before the ProjectConfig is stored.
func validateLabelsAndAnnotations(
	f *field.Path,
	cfg *kargoapi.ProjectConfig,
) field.ErrorList {
	errs := metav1validation.ValidateLabels(cfg.Labels, f.Child("labels"))
	return append(
		errs,
		apimachineryvalidation.ValidateAnnotations(
			cfg.Annotations,
			f.Child("annotations"),
		)...,
	)
}

func validateSpec(
	f *field.Path,
	spec *kargoapi.ProjectConfigSpec,
) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validatePromotionPolicies(f.Child("promotionPolicies"), spec.PromotionPolicies)...)
	errs = append(errs, validateWebhookReceivers(f.Child("webhookReceivers"), spec.WebhookReceivers)...)
	errs = append(errs, validateDeepLinks(f.Child("freightLinks"), spec.FreightLinks)...)
	errs = append(errs, validateDeepLinks(f.Child("stageLinks"), spec.StageLinks)...)
	errs = append(errs, validatePromotionWindows(f.Child("promotionWindows"), spec.PromotionWindows)...)
	return errs
}

// validatePromotionPolicies applies the CRD constraints on each policy, then
// the admission webhook's rules (ported unchanged).
func validatePromotionPolicies(
	f *field.Path,
	policies []kargoapi.PromotionPolicy,
) field.ErrorList {
	var errs field.ErrorList
	for i, policy := range policies {
		errs = append(errs, validatePromotionPolicySchema(f.Index(i), policy)...)
	}
	return append(errs, validatePromotionPolicyStages(f, policies)...)
}

func validatePromotionPolicySchema(
	f *field.Path,
	policy kargoapi.PromotionPolicy,
) field.ErrorList {
	var errs field.ErrorList
	stage := policy.Stage // nolint:staticcheck
	if (stage != "") == (policy.StageSelector != nil) {
		errs = append(errs, field.Invalid(f, field.OmitValueType{}, policyOneOfMessage))
	}
	if stage != "" {
		errs = append(errs, validatePattern(f.Child("stage"), stage)...)
	}
	if ar := policy.AutoRollback; ar != nil {
		arPath := f.Child("autoRollback")
		errs = append(errs, validateAutoRollbackPhases(
			arPath.Child("onPromotion"),
			"onPromotion",
			phasesToStrings(ar.OnPromotion),
			string(kargoapi.PromotionPhaseFailed),
			string(kargoapi.PromotionPhaseErrored),
		)...)
		errs = append(errs, validateAutoRollbackPhases(
			arPath.Child("onVerification"),
			"onVerification",
			phasesToStrings(ar.OnVerification),
			string(kargoapi.VerificationPhaseFailed),
			string(kargoapi.VerificationPhaseError),
		)...)
	}
	return errs
}

// validateAutoRollbackPhases mirrors the CRD's MaxItems=2, listType=set, and
// CEL rules on autoRollback.onPromotion and autoRollback.onVerification. The
// CEL rules check only the first two elements, which MaxItems makes the whole
// list.
func validateAutoRollbackPhases(
	f *field.Path,
	name string,
	phases []string,
	allowed ...string,
) field.ErrorList {
	var errs field.ErrorList
	if len(phases) > maxAutoRollbackPhases {
		errs = append(errs, field.TooMany(f, len(phases), maxAutoRollbackPhases))
	}
	for i := 0; i < len(phases) && i < maxAutoRollbackPhases; i++ {
		if !slices.Contains(allowed, phases[i]) {
			errs = append(errs, field.Invalid(
				f,
				field.OmitValueType{},
				fmt.Sprintf("%s[%d] must be %s or %s", name, i, allowed[0], allowed[1]),
			))
		}
	}
	return append(errs, validateUniqueValues(f, phases)...)
}

// validatePromotionPolicyStages is the admission webhook's promotion policy
// validation, copied unchanged: stage selector patterns must parse and exact
// Stage names must not repeat.
func validatePromotionPolicyStages(
	f *field.Path,
	promotionPolicies []kargoapi.PromotionPolicy,
) field.ErrorList {
	var errs field.ErrorList

	// Map of stage names to their indices in the promotionPolicies slice
	stageNames := make(map[string][]int)
	// Stage names in order of first appearance, so errors are deterministic
	var orderedStages []string

	for i, policy := range promotionPolicies {
		stage := policy.Stage // nolint:staticcheck
		if policy.StageSelector != nil {
			stage = policy.StageSelector.Name
		}

		// Skip empty stage names
		if stage == "" {
			continue
		}

		// Handle patterns (a name containing an identifier, e.g. "glob:")
		stageParts := strings.SplitN(stage, ":", 2)
		if len(stageParts) > 1 {
			m, err := pattern.ParseNamePattern(stage)
			if err != nil {
				errs = append(errs, field.Invalid(
					// NB: Our validation rule only allows patterns to be set
					// on the stageSelector's name field. The deprecated stage
					// field has to adhere to Kubernetes naming rules.
					f.Index(i).Child("stageSelector").Child("name"),
					stage,
					err.Error(),
				))
				continue
			}

			// The behavior of the pattern parser is to fall back to an exact
			// matcher. However, if a ":" is present we do not expect this
			// fallback to ever happen.
			if _, isExact := m.(*pattern.ExactMatcher); isExact {
				errs = append(errs, field.Invalid(
					f.Index(i).Child("stageSelector").Child("name"),
					stage,
					fmt.Sprintf(
						`invalid pattern identifier %q: must be "regex", "regexp" or "glob"`,
						stageParts[0],
					),
				))
			}

			continue
		}

		// Track stages by name and their positions
		if _, seen := stageNames[stage]; !seen {
			orderedStages = append(orderedStages, stage)
		}
		stageNames[stage] = append(stageNames[stage], i)
	}

	// Generate an error for each duplicate field
	for _, stage := range orderedStages {
		indices := stageNames[stage]
		// Skip the first occurrence (it's not a duplicate)
		for _, i := range indices[1:] {
			errs = append(errs, field.Invalid(
				f.Index(i),
				stage,
				fmt.Sprintf("stage name already defined at %s", f.Index(indices[0])),
			))
		}
	}

	return errs
}

// validateWebhookReceivers applies the CRD constraints on each receiver, then
// the rules the admission webhook shares with ClusterConfig.
func validateWebhookReceivers(
	f *field.Path,
	receivers []kargoapi.WebhookReceiverConfig,
) field.ErrorList {
	var errs field.ErrorList
	for i, r := range receivers {
		rPath := f.Index(i)
		errs = append(errs, validateName(rPath.Child("name"), r.Name)...)
		if r.Generic != nil {
			errs = append(errs, validateGenericReceiverSchema(rPath.Child("generic"), r.Generic)...)
		}
	}
	return append(errs, external.ValidateWebhookReceivers(f, receivers)...)
}

func validateGenericReceiverSchema(
	f *field.Path,
	cfg *kargoapi.GenericWebhookReceiverConfig,
) field.ErrorList {
	var errs field.ErrorList
	actionsPath := f.Child("actions")
	errs = append(errs, validateMinItems(actionsPath, cfg.Actions)...)
	for i, action := range cfg.Actions {
		aPath := actionsPath.Index(i)
		errs = append(errs, validateEnum(
			aPath.Child("action"),
			string(action.ActionType),
			string(kargoapi.GenericWebhookActionTypeRefresh),
		)...)
		targetsPath := aPath.Child("targetSelectionCriteria")
		errs = append(errs, validateMinItems(targetsPath, action.TargetSelectionCriteria)...)
		for j, target := range action.TargetSelectionCriteria {
			tPath := targetsPath.Index(j)
			errs = append(errs, validateEnum(
				tPath.Child("kind"),
				string(target.Kind),
				string(kargoapi.GenericWebhookTargetKindWarehouse),
			)...)
			indicesPath := tPath.Child("indexSelector", "matchIndices")
			errs = append(errs, validateMinItems(indicesPath, target.IndexSelector.MatchIndices)...)
			for k, req := range target.IndexSelector.MatchIndices {
				errs = append(errs, validateEnum(
					indicesPath.Index(k).Child("key"),
					req.Key,
					"subscribedURLs",
					"receiverPaths",
				)...)
			}
		}
	}
	return errs
}

// validateDeepLinks mirrors the CRD's required, MinLength=1 title and url.
func validateDeepLinks(f *field.Path, links []kargoapi.DeepLink) field.ErrorList {
	var errs field.ErrorList
	for i, link := range links {
		if link.Title == "" {
			errs = append(errs, field.Required(f.Index(i).Child("title"), ""))
		}
		if link.URL == "" {
			errs = append(errs, field.Required(f.Index(i).Child("url"), ""))
		}
	}
	return errs
}

// validatePromotionWindows mirrors the CRD constraints on promotion windows,
// including the unique names implied by listType=map with listMapKey=name.
// Rules beyond the CRD (rrule, dtstart, dtend, and so on) belong to registered
// spec validators.
func validatePromotionWindows(
	f *field.Path,
	windows []kargoapi.PromotionWindow,
) field.ErrorList {
	var errs field.ErrorList
	for i, w := range windows {
		wPath := f.Index(i)
		errs = append(errs, validateName(wPath.Child("name"), w.Name)...)
		if utf8.RuneCountInString(w.Description) > maxWindowDescriptionLength {
			errs = append(errs, field.TooLong(
				wPath.Child("description"),
				w.Description,
				maxWindowDescriptionLength,
			))
		}
		errs = append(errs, validateEnum(
			wPath.Child("kind"),
			string(w.Kind),
			string(kargoapi.PromotionWindowKindAllow),
			string(kargoapi.PromotionWindowKindDeny),
		)...)
	}
	seen := make(map[string]struct{}, len(windows))
	for i, w := range windows {
		if w.Name == "" {
			continue
		}
		if _, dup := seen[w.Name]; dup {
			errs = append(errs, field.Duplicate(f.Index(i), map[string]string{"name": w.Name}))
			continue
		}
		seen[w.Name] = struct{}{}
	}
	return errs
}

// validateName mirrors the CRD's required, MinLength=1, MaxLength=253, and
// Kubernetes name pattern constraints.
func validateName(f *field.Path, name string) field.ErrorList {
	if name == "" {
		return field.ErrorList{field.Required(f, "")}
	}
	var errs field.ErrorList
	if utf8.RuneCountInString(name) > maxNameLength {
		errs = append(errs, field.TooLong(f, name, maxNameLength))
	}
	return append(errs, validatePattern(f, name)...)
}

func validatePattern(f *field.Path, value string) field.ErrorList {
	if kubernetesNameRegex.MatchString(value) {
		return nil
	}
	return field.ErrorList{field.Invalid(
		f,
		value,
		fmt.Sprintf("should match '%s'", kubernetesNamePattern),
	)}
}

// validateEnum mirrors a required field with an enum constraint.
func validateEnum(f *field.Path, value string, allowed ...string) field.ErrorList {
	if value == "" {
		return field.ErrorList{field.Required(f, "")}
	}
	if !slices.Contains(allowed, value) {
		return field.ErrorList{field.NotSupported(f, value, allowed)}
	}
	return nil
}

// validateMinItems mirrors MinItems=1 on an optional list. A list that is
// present but empty decodes to a non-nil, empty slice, which the CRD rejects;
// an absent list decodes to nil, which the CRD accepts.
// This relies on the nil vs empty slice distinction a direct decode (Decode)
// preserves.
func validateMinItems[T any](f *field.Path, items []T) field.ErrorList {
	if items != nil && len(items) == 0 {
		return field.ErrorList{field.Invalid(f, 0, "should have at least 1 items")}
	}
	return nil
}

// validateUniqueValues mirrors listType=set.
func validateUniqueValues(f *field.Path, values []string) field.ErrorList {
	var errs field.ErrorList
	seen := make(map[string]struct{}, len(values))
	for i, v := range values {
		if _, dup := seen[v]; dup {
			errs = append(errs, field.Duplicate(f.Index(i), v))
			continue
		}
		seen[v] = struct{}{}
	}
	return errs
}

func phasesToStrings[T ~string](phases []T) []string {
	out := make([]string, len(phases))
	for i, p := range phases {
		out[i] = string(p)
	}
	return out
}
