package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/internal/api/validation"
)

type projectValidatorFunc func(ctx context.Context, project string) error

func newProjectValidator(kc client.Client) projectValidatorFunc {
	return func(ctx context.Context, project string) error {
		if err := validation.ValidateProject(ctx, kc, project); err != nil {
			if errors.Is(err, validation.ErrProjectNotFound) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			var fieldErr *field.Error
			if ok := errors.As(err, &fieldErr); ok {
				return connect.NewError(connect.CodeInvalidArgument, err)
			}
			return connect.NewError(connect.CodeInternal, err)
		}
		return nil
	}
}
