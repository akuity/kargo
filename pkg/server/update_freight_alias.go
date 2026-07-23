package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

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
