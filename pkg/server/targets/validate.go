package targets

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// bindTarget reads a Target from the request body and checks it the way
// admission would have checked a Target custom resource. The Target's name
// comes from the body, or from the route when it names one, in which case the
// two must agree. Its namespace, if given, must be the project. It reports
// false after responding with the error, as rest.Bind expects.
func bindTarget(c *gin.Context) (*kargoapi.Target, bool) {
	project, urlName := c.Param(paramProject), c.Param(paramName)
	target := &kargoapi.Target{}
	if err := c.ShouldBindJSON(target); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return nil, false
	}
	if urlName != "" {
		switch target.Name {
		case "":
			target.Name = urlName
		case urlName:
		default:
			_ = c.Error(libhttp.ErrorStr(
				"name in body does not match target name in URL",
				http.StatusBadRequest,
			))
			return nil, false
		}
	}
	if target.Namespace != "" && target.Namespace != project {
		_ = c.Error(libhttp.ErrorStr(
			"namespace in body does not match project name in URL",
			http.StatusBadRequest,
		))
		return nil, false
	}
	target.Namespace = project
	if errs := validateTarget(target); len(errs) > 0 {
		_ = c.Error(httpError(apierrors.NewInvalid(
			schema.GroupKind{Group: kargoapi.GroupVersion.Group, Kind: "Target"},
			target.Name,
			errs,
		)))
		return nil, false
	}
	return target, true
}

// validateTarget checks what the Target custom resource's schema and
// ObjectMeta validation would have checked. Metadata that only Kubernetes
// would have honored is refused rather than silently dropped; status and the
// server-set fields of metadata are ignored, as a status subresource and the
// API server would have ignored them.
func validateTarget(target *kargoapi.Target) field.ErrorList {
	var errs field.ErrorList
	if target.APIVersion != "" && target.APIVersion != kargoapi.GroupVersion.String() {
		errs = append(errs, field.NotSupported(
			field.NewPath("apiVersion"), target.APIVersion,
			[]string{kargoapi.GroupVersion.String()},
		))
	}
	if target.Kind != "" && target.Kind != "Target" {
		errs = append(errs, field.NotSupported(
			field.NewPath("kind"), target.Kind, []string{"Target"},
		))
	}
	meta := field.NewPath("metadata")
	if target.Name == "" {
		errs = append(errs, field.Required(meta.Child("name"), "name is required"))
	} else {
		for _, msg := range validation.IsDNS1123Subdomain(target.Name) {
			errs = append(errs, field.Invalid(meta.Child("name"), target.Name, msg))
		}
	}
	errs = append(errs, metav1validation.ValidateLabels(target.Labels, meta.Child("labels"))...)
	if len(target.Annotations) > 0 {
		errs = append(errs, field.Forbidden(meta.Child("annotations"), "Targets do not carry annotations"))
	}
	if len(target.Finalizers) > 0 {
		errs = append(errs, field.Forbidden(meta.Child("finalizers"), "Targets do not carry finalizers"))
	}
	if len(target.OwnerReferences) > 0 {
		errs = append(errs, field.Forbidden(meta.Child("ownerReferences"), "Targets do not carry owner references"))
	}
	return errs
}
