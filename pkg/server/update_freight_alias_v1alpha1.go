package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
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
		client.MatchingLabels{kargoapi.LabelKeyAlias: newAlias},
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
				"alias %q already used by another Freight resource in namespace %q",
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
			kargoapi.LabelKeyAlias,
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

// @id PatchFreightAlias
// @Summary Patch a Freight resource's alias
// @Description Patch a Freight resource's human-friendly alias.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Param newAlias query string true "New alias"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias}/alias [patch]
func (s *server) patchFreightAliasHandler(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")
	newAlias := c.Query("newAlias")

	if newAlias == "" {
		_ = c.Error(libhttp.ErrorStr(
			"newAlias query parameter is required",
			http.StatusBadRequest,
		))
		return
	}

	freight := s.getFreightByNameOrAliasForGin(c, project, nameOrAlias)
	if freight == nil {
		return
	}

	// Make sure this alias isn't already used by some other piece of Freight
	freightList := kargoapi.FreightList{}
	if err := s.client.List(
		ctx,
		&freightList,
		client.InNamespace(project),
		client.MatchingLabels{kargoapi.LabelKeyAlias: newAlias},
	); err != nil {
		_ = c.Error(err)
		return
	}
	if len(freightList.Items) > 1 ||
		(len(freightList.Items) == 1 && freightList.Items[0].Name != freight.Name) {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"alias %q already used by another piece of Freight in namespace %q",
				newAlias,
				project,
			),
			http.StatusConflict,
		))
		return
	}

	// Proceed with the update using patch
	patchBytes := []byte(
		fmt.Sprintf(
			`{"metadata":{"labels":{%q:%q}},"alias":%q}`,
			kargoapi.LabelKeyAlias,
			newAlias,
			newAlias,
		),
	)
	patch := client.RawPatch(types.MergePatchType, patchBytes)
	if err := s.client.Patch(ctx, freight, patch); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
