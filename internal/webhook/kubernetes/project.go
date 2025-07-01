package webhook

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/server/validation"
)

func ValidateProject(
	ctx context.Context,
	kubeClient client.Client,
	gk schema.GroupKind,
	obj client.Object,
) error {
	if err := validation.ValidateProject(
		ctx,
		kubeClient,
		obj.GetNamespace(),
	); err != nil {
		if errors.Is(err, validation.ErrProjectNotFound) {
			return apierrors.NewNotFound(
				schema.GroupResource{
					Group:    corev1.SchemeGroupVersion.Group,
					Resource: "Namespace",
				},
				obj.GetNamespace(),
			)
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); ok {
			return apierrors.NewInvalid(
				gk,
				obj.GetName(),
				field.ErrorList{fieldErr},
			)
		}
		return apierrors.NewInternalError(err)
	}
	return nil
}
