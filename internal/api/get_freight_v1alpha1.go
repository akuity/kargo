package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetFreightRequest],
) (*connect.Response[svcv1alpha1.GetFreightResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	alias := req.Msg.GetAlias()
	if (name == "" && alias == "") || (name != "" && alias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of name or alias should not be empty"),
		)
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	freight, err := s.getFreightByNameOrAliasFn(
		ctx,
		s.client,
		project,
		name,
		alias,
	)
	if err != nil {
		return nil, errors.Wrap(err, "get freight")
	}
	if freight == nil {
		if name != "" {
			err = fmt.Errorf("freight %q not found in namespace %q", name, project)
		} else {
			err = fmt.Errorf("freight with alias %q not found in namespace %q", alias, project)
		}
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&svcv1alpha1.GetFreightResponse{
		Freight: typesv1alpha1.ToFreightProto(*freight),
	}), nil
}
