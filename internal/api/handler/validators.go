package handler

import (
	"context"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
)

type creationValidatorFunc func(ctx context.Context, object metav1.Object) error

func newCreationValidator(kc client.Client) creationValidatorFunc {
	return func(ctx context.Context, object metav1.Object) error {
		var ns corev1.Namespace
		if err := kc.Get(ctx, client.ObjectKey{Name: object.GetNamespace()}, &ns); err != nil {
			if kubeerr.IsNotFound(err) {
				return connect.NewError(connect.CodeNotFound,
					errors.Errorf("project %q not found", object.GetNamespace()))
			}
			return connect.NewError(connect.CodeInternal, errors.Wrap(err, "get project"))
		}
		if ns.GetLabels()[v1alpha1.LabelProjectKey] != v1alpha1.LabelTrueValue {
			return connect.NewError(connect.CodeFailedPrecondition,
				errors.Errorf("namespace %q is not a project", object.GetNamespace()))
		}
		return nil
	}
}

type projectExistenceValidatorFunc func(ctx context.Context, project string) error

func newProjectExistenceValidator(kc client.Client) projectExistenceValidatorFunc {
	return func(ctx context.Context, project string) error {
		if err := kc.Get(ctx, client.ObjectKey{Name: project}, &corev1.Namespace{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return connect.NewError(connect.CodeNotFound,
					fmt.Errorf("project %q not found", project))
			}
			return connect.NewError(connect.CodeInternal, err)
		}
		return nil
	}
}
