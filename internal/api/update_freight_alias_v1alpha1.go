package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

// UpdateFreightAlias updates a piece of Freight's human-friendly alias.
func (s *server) UpdateFreightAlias(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateFreightAliasRequest],
) (*connect.Response[svcv1alpha1.UpdateFreightAliasResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("project should not be empty"),
		)
	}

	if req.Msg.GetFreight() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("freight should not be empty"),
		)
	}

	if req.Msg.GetAlias() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("alias should not be empty"),
		)
	}

	if err := s.validateProjectFn(ctx, req.Msg.GetProject()); err != nil {
		return nil, err // This already returns a connect.Error
	}

	freight, err := s.getFreightFn(
		ctx,
		s.client,
		types.NamespacedName{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetFreight(),
		},
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if freight == nil {
		return nil, connect.NewError(
			connect.CodeNotFound,
			errors.Errorf(
				"Freight %q not found in namespace %q",
				req.Msg.GetFreight(),
				req.Msg.GetProject(),
			),
		)
	}

	// Make sure this alias isn't already used by some other piece of Freight
	freightList := kargoapi.FreightList{}
	if err = s.listFreightFn(
		ctx,
		&freightList,
		client.InNamespace(req.Msg.GetProject()),
		client.MatchingLabels{kargoapi.AliasLabelKey: req.Msg.GetAlias()},
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(freightList.Items) > 1 ||
		(len(freightList.Items) == 1 && freightList.Items[0].Name != freight.Name) {
		return nil, connect.NewError(
			// TODO: This should probably be a 409 Conflict, but connect doesn't seem
			// to have that
			connect.CodeAlreadyExists,
			errors.Errorf(
				"alias %q already used by another piece of Freight in namespace %q",
				req.Msg.GetAlias(),
				req.Msg.GetProject(),
			),
		)
	}

	// Proceed with the update
	if err = s.patchFreightAliasFn(ctx, freight, req.Msg.GetAlias()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&svcv1alpha1.UpdateFreightAliasResponse{}), nil
}

func (s *server) patchFreightAlias(
	ctx context.Context,
	freight *kargoapi.Freight,
	alias string,
) error {
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"labels":{"%s":"%s"}}}`,
			kargoapi.AliasLabelKey,
			alias,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := s.client.Patch(ctx, freight, patch); err != nil {
		return errors.Wrap(err, "patch label")
	}
	return nil
}
