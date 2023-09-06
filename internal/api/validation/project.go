package validation

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

var (
	ErrProjectNotFound = errors.New("project not found")
)

func ValidateProject(ctx context.Context, kc client.Client, project string) error {
	var ns corev1.Namespace
	if err := kc.Get(ctx, client.ObjectKey{Name: project}, &ns); err != nil {
		if kubeerr.IsNotFound(err) {
			return ErrProjectNotFound
		}
		return errors.Wrap(err, "get project")
	}
	if ns.GetLabels()[kargoapi.LabelProjectKey] != kargoapi.LabelTrueValue {
		return field.Invalid(field.NewPath("metadata", "namespace"),
			project, fmt.Sprintf("namespace %q is not a project", project))
	}
	return nil
}
