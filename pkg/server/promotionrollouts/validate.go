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

// createPromotionRolloutRequest is the body of a request to create a
// PromotionRollout. The Stage is named by the route, and everything else
// about the rollout is assigned by the server, so the Freight is all there
// is to give, and which of the Stage's Targets to promote to, if not all.
type createPromotionRolloutRequest struct {
	// Freight is the name of the Freight to promote.
	Freight string `json:"freight"`
	// Targets names the Targets to promote to, each of which the Stage must
	// govern. Omitted, the rollout promotes to every Target the Stage
	// governs. Naming a few is how a canary is run or a failed rollout is
	// retried for only the Targets that failed.
	Targets []string `json:"targets,omitempty"`
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

// validateCreateRequest checks that the request names Freight, and that any
// Targets it names are names and named once, as the PromotionRequest custom
// resource's schema would have.
func validateCreateRequest(req createPromotionRolloutRequest) field.ErrorList {
	var errs field.ErrorList
	path := field.NewPath("freight")
	if req.Freight == "" {
		errs = append(errs, field.Required(path, "a name is required"))
	}
	for _, msg := range validation.IsDNS1123Subdomain(req.Freight) {
		if req.Freight != "" {
			errs = append(errs, field.Invalid(path, req.Freight, msg))
		}
	}
	seen := make(map[string]struct{}, len(req.Targets))
	for i, name := range req.Targets {
		path := field.NewPath("targets").Index(i)
		for _, msg := range validation.IsDNS1123Subdomain(name) {
			errs = append(errs, field.Invalid(path, name, msg))
		}
		if _, dup := seen[name]; dup {
			errs = append(errs, field.Duplicate(path, name))
		}
		seen[name] = struct{}{}
	}
	return errs
}
