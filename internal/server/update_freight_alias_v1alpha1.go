package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
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
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	oldAlias := req.Msg.GetOldAlias()
	if (name == "" && oldAlias == "") || (name != "" && oldAlias != "") {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("exactly one of name or oldAlias should not be empty"),
		)
	}

	newAlias := req.Msg.GetNewAlias()
	if err := validateFieldNotEmpty("newAlias", newAlias); err != nil {
		return nil, err
	}

	if err := s.validateProjectExistsFn(ctx, project); err != nil {
		return nil, err // This already returns a connect.Error
	}

	freight, err := s.getFreightByNameOrAliasFn(
		ctx,
		s.client,
		project,
		name,
		oldAlias,
	)
	if err != nil {
		return nil, fmt.Errorf("get freight: %w", err)
	}
	if freight == nil {
		if name != "" {
			err = fmt.Errorf("freight %q not found in namespace %q", name, project)
		} else {
			err = fmt.Errorf("freight with alias %q not found in namespace %q", oldAlias, project)
		}
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Make sure this alias isn't already used by some other piece of Freight
	freightList := kargoapi.FreightList{}
	if err = s.listFreightFn(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{kargoapi.AliasLabelKey: newAlias},
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(freightList.Items) > 1 ||
		(len(freightList.Items) == 1 && freightList.Items[0].Name != freight.Name) {
		return nil, connect.NewError(
			// TODO: This should probably be a 409 Conflict, but connect doesn't seem
			// to have that
			connect.CodeAlreadyExists,
			fmt.Errorf(
				"alias %q already used by another piece of Freight in namespace %q",
				newAlias,
				project,
			),
		)
	}

	// Proceed with the update
	if err = s.patchFreightAliasFn(ctx, freight, newAlias); err != nil {
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
			`{"metadata":{"labels":{%q:%q}},"alias":%q}`,
			kargoapi.AliasLabelKey,
			alias,
			alias,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := s.client.Patch(ctx, freight, patch); err != nil {
		return fmt.Errorf("patch label: %w", err)
	}
	return nil
}
