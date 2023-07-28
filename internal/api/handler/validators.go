package handler

import (
	"context"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
)

type projectValidatorFunc func(ctx context.Context, project string) error

func newProjectValidator(kc client.Client) projectValidatorFunc {
	return func(ctx context.Context, project string) error {
		var ns corev1.Namespace
		if err := kc.Get(ctx, client.ObjectKey{Name: project}, &ns); err != nil {
			if kubeerr.IsNotFound(err) {
				return connect.NewError(connect.CodeNotFound,
					fmt.Errorf("project %q not found", project))
			}
			return connect.NewError(connect.CodeInternal, errors.Wrap(err, "get project"))
		}
		if ns.GetLabels()[v1alpha1.LabelProjectKey] != v1alpha1.LabelTrueValue {
			return connect.NewError(connect.CodeFailedPrecondition,
				errors.Errorf("namespace %q is not a project", project))
		}
		return nil
	}
}
