package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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

// validateFreightExists returns the Freight with the given ID in the list of Freight, otherwise
// return an error if it doesn't exist
func validateFreightExists(freight string, freightStack kargoapi.FreightStack) (*kargoapi.Freight, error) {
	if freight == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("freight should not be empty"))
	}
	for _, f := range freightStack {
		if freight == f.ID {
			return &f, nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("freight %q not found in Stage", freight))
}

func validateGroupByOrderBy(group string, groupBy string, orderBy string) error {
	if group != "" && groupBy == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("Cannot filter by group without group by"))
	}
	switch groupBy {
	case GroupByContainerRepository, GroupByGitRepository, GroupByHelmRepository, "":
	default:
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("Invalid group by: %s", groupBy))
	}
	switch orderBy {
	case OrderByTag:
		if groupBy != GroupByContainerRepository && groupBy != GroupByHelmRepository {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("Tag ordering only valid when grouping by: %s, %s",
					GroupByContainerRepository, GroupByHelmRepository))
		}
	case OrderByFirstSeen, "":
	default:
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("Invalid order by: %s", orderBy))
	}

	return nil
}
