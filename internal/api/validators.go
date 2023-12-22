package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/akuity/kargo/internal/api/validation"
)

func (s *server) validateProject(ctx context.Context, project string) error {
	if err := s.externalValidateProjectFn(ctx, s.client, project); err != nil {
		if errors.Is(err, validation.ErrProjectNotFound) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); ok {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
		return errors.Wrap(err, "validate project")
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

func validateProjectAndWarehouseName(project, name string) error {
	if project == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if name == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
	}
	return nil
}

func validateProjectAndFreightName(project, name string) error {
	if project == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if name == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("freight should not be empty"))
	}
	return nil
}

func validateGroupByOrderBy(group string, groupBy string, orderBy string) error {
	if group != "" && groupBy == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.Errorf("Cannot filter by group without group by"),
		)
	}
	switch groupBy {
	case GroupByImageRepository, GroupByGitRepository, GroupByChartRepository, "":
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.Errorf("Invalid group by: %s", groupBy),
		)
	}
	switch orderBy {
	case OrderByTag:
		if groupBy != GroupByImageRepository && groupBy != GroupByChartRepository {
			return connect.NewError(connect.CodeInvalidArgument,
				errors.Errorf("Tag ordering only valid when grouping by: %s, %s",
					GroupByImageRepository, GroupByChartRepository))
		}
	case OrderByFirstSeen, "":
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.Errorf("Invalid order by: %s", orderBy),
		)
	}

	return nil
}
