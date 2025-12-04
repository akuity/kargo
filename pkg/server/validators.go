package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/validation"
)

func validateFieldNotEmpty(fieldName string, fieldValue string) error {
	if fieldValue == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("%s should not be empty", fieldName),
		)
	}
	return nil
}

func (s *server) validateSystemLevelOrProject(
	systemLevel bool,
	project string,
) error {
	if !systemLevel && project == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("project must be specified when system level is false"),
		)
	}
	return nil
}

func (s *server) validateProjectExists(ctx context.Context, project string) error {
	var cl client.Client = s.client
	if s.client != nil && s.client.InternalClient() != nil {
		cl = s.client.InternalClient()
	}
	if err := s.externalValidateProjectFn(ctx, cl, project); err != nil {
		if errors.Is(err, validation.ErrProjectNotFound) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); ok {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
		return fmt.Errorf("validate project: %w", err)
	}
	return nil
}

func validateGroupByOrderBy(group string, groupBy string, orderBy string) error {
	if group != "" && groupBy == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("cannot filter by group without group by"),
		)
	}
	switch groupBy {
	case GroupByImageRepository, GroupByGitRepository, GroupByChartRepository, "":
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("invalid group by: %s", groupBy),
		)
	}
	switch orderBy {
	case OrderByTag:
		if groupBy != GroupByImageRepository && groupBy != GroupByChartRepository {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("tag ordering only valid when grouping by: %s, %s",
					GroupByImageRepository, GroupByChartRepository))
		}
	case OrderByFirstSeen, "":
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("invalid order by: %s", orderBy),
		)
	}

	return nil
}
