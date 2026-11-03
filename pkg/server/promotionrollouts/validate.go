package promotionrollouts

import (
	"fmt"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/request"
)

// createPromotionRolloutRequest is the body of a request to create a
// PromotionRollout. The Stage is named by the route, and everything else
// about the rollout is assigned by the server, so the Freight is all there
// is to give, which of the Stage's Targets to promote to, if not all, and
// how fast, if not as the Stage says.
type createPromotionRolloutRequest struct {
	// Freight is the name of the Freight to promote.
	Freight string `json:"freight"`
	// Targets names the Targets to promote to, each of which the Stage must
	// govern. Omitted, the rollout promotes to every Target the Stage
	// governs. Naming a few is how a canary is run or a failed rollout is
	// retried for only the Targets that failed.
	Targets []string `json:"targets,omitempty"`
	// UpdateStrategy paces this rollout in place of the Stage's update
	// strategy. Omitted, the Stage's applies. Either way, a percentage is
	// resolved against the Targets the rollout promotes to, and the numbers
	// that gives are recorded with the rollout for good.
	UpdateStrategy *kargoapi.TargetUpdateStrategy `json:"updateStrategy,omitempty"`
} // @name CreatePromotionRolloutRequest

// bindCreateRequest reads a request to create a PromotionRollout from the
// request body and checks it. It reports false after recording the error,
// in which case the handler must return.
func bindCreateRequest(c *gin.Context) (createPromotionRolloutRequest, bool) {
	var req createPromotionRolloutRequest
	if !request.BindJSON(c, &req) {
		return req, false
	}
	if errs := validateCreateRequest(req); len(errs) > 0 {
		_ = c.Error(apierrors.NewInvalid(
			schema.GroupKind{Group: kargoapi.GroupVersion.Group, Kind: "PromotionRollout"},
			"",
			errs,
		))
		return req, false
	}
	return req, true
}

// validateCreateRequest checks that the request names Freight, that any
// Targets it names are names and named once, and that any update strategy
// it gives is one a Stage could have, as the PromotionRequest and Stage
// custom resources' schemas would have.
func validateCreateRequest(req createPromotionRolloutRequest) field.ErrorList {
	var errs field.ErrorList
	path := field.NewPath("freight")
	if req.Freight == "" {
		errs = append(errs, field.Required(path, "a name is required"))
	} else {
		for _, msg := range validation.IsDNS1123Subdomain(req.Freight) {
			errs = append(errs, field.Invalid(path, req.Freight, msg))
		}
	}
	seen := make(map[string]struct{}, len(req.Targets))
	for i, name := range req.Targets {
		targetPath := field.NewPath("targets").Index(i)
		for _, msg := range validation.IsDNS1123Subdomain(name) {
			errs = append(errs, field.Invalid(targetPath, name, msg))
		}
		if _, dup := seen[name]; dup {
			errs = append(errs, field.Duplicate(targetPath, name))
		}
		seen[name] = struct{}{}
	}
	if req.UpdateStrategy != nil {
		errs = append(errs, validateUpdateStrategy(field.NewPath("updateStrategy"), *req.UpdateStrategy)...)
	}
	return errs
}

// validateUpdateStrategy checks an update strategy as the Stage webhook
// checks a Stage's: each value given is a number or a percentage, with
// maxConcurrent at least 1 and maxFailures at least 0.
func validateUpdateStrategy(
	path *field.Path,
	strategy kargoapi.TargetUpdateStrategy,
) field.ErrorList {
	var errs field.ErrorList
	if strategy.MaxConcurrent != nil {
		if err := validateUpdateStrategyValue(path.Child("maxConcurrent"), strategy.MaxConcurrent, 1); err != nil {
			errs = append(errs, err)
		}
	}
	if strategy.MaxFailures != nil {
		if err := validateUpdateStrategyValue(path.Child("maxFailures"), strategy.MaxFailures, 0); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateUpdateStrategyValue(
	path *field.Path,
	value *intstr.IntOrString,
	floor int,
) *field.Error {
	// Scaled against 100 so that a percentage is checked as a percentage: 1%
	// is 1, 0% is 0.
	n, err := intstr.GetScaledValueFromIntOrPercent(value, 100, true)
	if err != nil {
		return field.Invalid(path, value, err.Error())
	}
	if n < floor {
		return field.Invalid(path, value, fmt.Sprintf("value cannot be below %d", floor))
	}
	return nil
}
