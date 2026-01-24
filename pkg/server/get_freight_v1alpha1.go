package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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

	// Get the Freight from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "Freight",
		},
	}

	switch {
	case alias != "":
		ul := unstructured.UnstructuredList{
			Object: map[string]any{
				"apiVersion": kargoapi.GroupVersion.String(),
				"kind":       "FreightList",
			},
		}
		if err := s.client.List(
			ctx,
			&ul,
			client.InNamespace(project),
			client.MatchingLabels{
				kargoapi.LabelKeyAlias: alias,
			},
		); err != nil {
			return nil, err
		}
		if len(ul.Items) == 0 {
			// nolint:staticcheck
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf("Freight with alias %q not found in namespace %q", alias, project),
			)
		}
		u = &ul.Items[0]
	default:
		if err := s.client.Get(
			ctx, client.ObjectKey{Namespace: project, Name: name}, u,
		); err != nil {
			if client.IgnoreNotFound(err) == nil {
				// nolint:staticcheck
				err = fmt.Errorf("Freight %q not found in namespace %q", name, project)
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, err
		}
	}

	f, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.Freight{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetFreightResponse{
			Result: &svcv1alpha1.GetFreightResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetFreightResponse{
		Result: &svcv1alpha1.GetFreightResponse_Freight{Freight: f},
	}), nil
}

// @id GetFreight
// @Summary Retrieve a Freight resource
// @Description Retrieve a Freight resource from a project's namespace by name
// @Description or alias.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param freight-name-or-alias path string true "Freight name or alias"
// @Success 200 {object} object "Freight custom resource (github.com/akuity/kargo/api/v1alpha1.Freight)"
// @Router /v1beta1/projects/{project}/freight/{freight-name-or-alias} [get]
func (s *server) getFreight(c *gin.Context) {
	project := c.Param("project")
	nameOrAlias := c.Param("freight-name-or-alias")

	freight := s.getFreightByNameOrAliasForGin(c, project, nameOrAlias)
	if freight == nil {
		return
	}

	c.JSON(http.StatusOK, freight)
}
