package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id ListFreightByCurrentStage
// @Summary List Freight currently in a Stage
// @Description List the Freight whose status records it as currently in use by
// @Description the named Stage. Returns a FreightList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 {object} kargoapi.FreightList "FreightList custom resource"
// @Router /v1beta1/projects/{project}/stages/{stage}/freight [get]
func (s *server) listFreightByCurrentStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	freight, err := api.ListFreightByCurrentStage(ctx, s.client, project, stageName)
	if err != nil {
		_ = c.Error(fmt.Errorf("list Freight currently in Stage %q: %w", stageName, err))
		return
	}
	c.JSON(http.StatusOK, &kargoapi.FreightList{Items: freight})
}

// @id ListFreightReferencingStage
// @Summary List Freight referencing a Stage
// @Description List the Freight whose status records the named Stage in its
// @Description verifiedIn or approvedFor map. Returns a FreightList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 {object} kargoapi.FreightList "FreightList custom resource"
// @Router /v1beta1/projects/{project}/stages/{stage}/referencing-freight [get]
func (s *server) listFreightReferencingStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	stageName := c.Param("stage")

	freight, err := api.ListFreightReferencingStage(ctx, s.client, project, stageName)
	if err != nil {
		_ = c.Error(fmt.Errorf("list Freight referencing Stage %q: %w", stageName, err))
		return
	}
	c.JSON(http.StatusOK, &kargoapi.FreightList{Items: freight})
}

// @id ListFreightFromWarehouse
// @Summary List Freight from a Warehouse
// @Description List Freight that originated from the named Warehouse, optionally
// @Description filtered to that approved for or verified in given Stages, currently
// @Description in a Stage, and having soaked for a minimum duration. Returns a
// @Description FreightList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param warehouse path string true "Warehouse name"
// @Param approvedFor query string false "Include Freight approved for this Stage"
// @Param verifiedIn query []string false "Include Freight verified in these Stages" collectionFormat(multi)
// @Param currentlyIn query string false "Only Freight currently in this Stage"
// @Param availabilityStrategy query string false "How verifiedIn Stages are combined (OneOf or All)"
// @Param requiredSoakTime query string false "Minimum soak duration, e.g. 1h"
// @Success 200 {object} kargoapi.FreightList "FreightList custom resource"
// @Router /v1beta1/projects/{project}/warehouses/{warehouse}/freight [get]
func (s *server) listFreightFromWarehouse(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	warehouseName := c.Param("warehouse")

	opts, err := warehouseFreightOptionsFromQuery(c)
	if err != nil {
		_ = c.Error(libhttp.ErrorStr(err.Error(), http.StatusBadRequest))
		return
	}

	freight, err := api.ListFreightFromWarehouse(ctx, s.client, project, warehouseName, opts)
	if err != nil {
		_ = c.Error(fmt.Errorf("list Freight from Warehouse %q: %w", warehouseName, err))
		return
	}
	c.JSON(http.StatusOK, &kargoapi.FreightList{Items: freight})
}

// warehouseFreightOptionsFromQuery builds ListWarehouseFreightOptions from the
// request's query parameters, returning nil options when none are set.
func warehouseFreightOptionsFromQuery(c *gin.Context) (*freightclient.ListWarehouseFreightOptions, error) {
	opts := &freightclient.ListWarehouseFreightOptions{
		ApprovedFor:          c.Query("approvedFor"),
		VerifiedIn:           c.QueryArray("verifiedIn"),
		CurrentlyIn:          c.Query("currentlyIn"),
		AvailabilityStrategy: kargoapi.FreightAvailabilityStrategy(c.Query("availabilityStrategy")),
	}
	if soak := c.Query("requiredSoakTime"); soak != "" {
		d, err := time.ParseDuration(soak)
		if err != nil {
			return nil, fmt.Errorf("invalid requiredSoakTime %q: %w", soak, err)
		}
		opts.RequiredSoakTime = &metav1.Duration{Duration: d}
	}
	return opts, nil
}
