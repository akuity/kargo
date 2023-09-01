package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/validation"
)

func (s *server) validateProject(ctx context.Context, project string) error {
	if err := validation.ValidateProject(ctx, s.client, project); err != nil {
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

func validateProjectAndStageNonEmpty(project string, stage string) error {
	if project == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if stage == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("stage should not be empty"))
	}
	return nil
}

// validateFreightExists returns the freight with the given ID in the list of freights, otherwise
// return an error if it doesn't exist
func validateFreightExists(freight string, freights kubev1alpha1.StageStateStack) (*kubev1alpha1.StageState, error) {
	if freight == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("freight should not be empty"))
	}
	for _, f := range freights {
		if freight == f.ID {
			return &f, nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("freight %q not found in Stage", freight))
}
