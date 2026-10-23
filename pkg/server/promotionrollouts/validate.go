package promotionrollouts

import (
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/request"
)

// bindPromotionRollout reads a PromotionRollout from the request body and
// checks it the way admission would have checked a PromotionRequest custom
// resource. Its namespace, if given, must be the project. It reports false
// after recording the error, in which case the handler must return.
func bindPromotionRollout(c *gin.Context) (*kargoapi.PromotionRollout, bool) {
	project := c.Param(paramProject)
	rollout := &kargoapi.PromotionRollout{}
	if !request.BindJSON(c, rollout) {
		return nil, false
	}
	if errs := validatePromotionRollout(rollout, project); len(errs) > 0 {
		_ = c.Error(apierrors.NewInvalid(
			schema.GroupKind{Group: kargoapi.GroupVersion.Group, Kind: "PromotionRollout"},
			rollout.Name,
			errs,
		))
		return nil, false
	}
	rollout.Namespace = project
	return rollout, true
}

// validatePromotionRollout checks what the PromotionRequest custom resource's
// schema and ObjectMeta validation would have checked. Everything the server
// assigns -- the name, the Targets, the number, the status and the
// server-set fields of metadata -- is refused if given rather than silently
// dropped, so that a client never believes it chose something it did not.
func validatePromotionRollout(rollout *kargoapi.PromotionRollout, project string) field.ErrorList {
	var errs field.ErrorList
	meta := field.NewPath("metadata")
	if rollout.Name != "" {
		errs = append(errs, field.Forbidden(meta.Child("name"), "the name of a PromotionRollout is generated"))
	}
	if rollout.Namespace != "" && rollout.Namespace != project {
		errs = append(errs, field.Invalid(
			meta.Child("namespace"), rollout.Namespace, "namespace in body does not match project name in URL",
		))
	}
	if len(rollout.Labels) > 0 {
		errs = append(errs, field.Forbidden(meta.Child("labels"), "PromotionRollouts do not carry labels"))
	}
	if len(rollout.Annotations) > 0 {
		errs = append(errs, field.Forbidden(meta.Child("annotations"), "PromotionRollouts do not carry annotations"))
	}
	if len(rollout.Finalizers) > 0 {
		errs = append(errs, field.Forbidden(meta.Child("finalizers"), "PromotionRollouts do not carry finalizers"))
	}
	if len(rollout.OwnerReferences) > 0 {
		errs = append(errs, field.Forbidden(
			meta.Child("ownerReferences"), "PromotionRollouts do not carry owner references",
		))
	}
	if rollout.Number != 0 {
		errs = append(errs, field.Forbidden(field.NewPath("number"), "the number of a PromotionRollout is assigned"))
	}

	spec := field.NewPath("spec")
	for _, name := range []struct {
		path  *field.Path
		value string
	}{
		{spec.Child("stage"), rollout.Spec.Stage},
		{spec.Child("freight"), rollout.Spec.Freight},
	} {
		if name.value == "" {
			errs = append(errs, field.Required(name.path, "a name is required"))
			continue
		}
		for _, msg := range validation.IsDNS1123Subdomain(name.value) {
			errs = append(errs, field.Invalid(name.path, name.value, msg))
		}
	}
	if len(rollout.Spec.Targets) > 0 {
		errs = append(errs, field.Forbidden(
			spec.Child("targets"), "the Targets of a PromotionRollout are resolved from its Stage's selectors",
		))
	}
	if rollout.Spec.UpdateStrategy != (kargoapi.PromotionRolloutUpdateStrategy{}) {
		errs = append(errs, field.Forbidden(
			spec.Child("updateStrategy"), "the update strategy of a PromotionRollout is taken from its Stage",
		))
	}
	return errs
}
