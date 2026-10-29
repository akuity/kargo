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
// is to give.
type createPromotionRolloutRequest struct {
	// Freight is the name of the Freight to promote.
	Freight string `json:"freight"`
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

// validateCreateRequest checks that the request names Freight, as the
// PromotionRequest custom resource's schema would have.
func validateCreateRequest(req createPromotionRolloutRequest) field.ErrorList {
	path := field.NewPath("freight")
	if req.Freight == "" {
		return field.ErrorList{field.Required(path, "a name is required")}
	}
	var errs field.ErrorList
	for _, msg := range validation.IsDNS1123Subdomain(req.Freight) {
		errs = append(errs, field.Invalid(path, req.Freight, msg))
	}
	return errs
}
