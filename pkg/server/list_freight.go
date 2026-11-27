package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/dataplane/freightclient"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id ListFreightFromOrigin
// @Summary List Freight from an Origin
// @Description List Freight that originated from the named Origin, optionally
// @Description filtered to that approved for or verified in given Stages, currently
// @Description in a Stage, and having soaked for a minimum duration. Returns a
// @Description FreightList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param origin-kind path string true "Origin kind"
// @Param origin-name path string true "Origin name"
// @Param approvedFor query string false "Include Freight approved for this Stage"
// @Param verifiedIn query []string false "Include Freight verified in these Stages" collectionFormat(multi)
// @Param currentlyIn query string false "Only Freight currently in this Stage"
// @Param availabilityStrategy query string false "How verifiedIn Stages are combined (OneOf or All)"
// @Param requiredSoakTime query string false "Minimum soak duration, e.g. 1h"
// @Success 200 {object} kargoapi.FreightList "FreightList custom resource"
// @Router /v1beta1/projects/{project}/internal/origin/{origin-kind}/{origin-name}/freight [get]
func (s *server) listFreightFromOrigin(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	originKind := c.Param("origin-kind")
	originName := c.Param("origin-name")

	opts, err := originFreightOptionsFromQuery(c)
	if err != nil {
		_ = c.Error(libhttp.ErrorStr(err.Error(), http.StatusBadRequest))
		return
	}

	kind, err := kargoapi.ParseFreightOriginKind(originKind)
	if err != nil {
		_ = c.Error(libhttp.ErrorStr(err.Error(), http.StatusBadRequest))
		return
	}

	origin := kargoapi.FreightOrigin{
		Kind: kind,
		Name: originName,
	}

	freight, err := s.freightClientFn().ListFreightFromOrigin(ctx, project, origin, opts)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported AvailabilityStrategy") {
			err = libhttp.Error(err, http.StatusBadRequest)
		}
		_ = c.Error(fmt.Errorf("list Freight from Origin %v: %w", origin, err))
		return
	}
	c.JSON(http.StatusOK, &kargoapi.FreightList{Items: freight})
}

// originFreightOptionsFromQuery builds ListFreightFromOriginOptions from the
// request's query parameters, returning nil options when none are set.
func originFreightOptionsFromQuery(c *gin.Context) (*freightclient.ListFreightFromOriginOptions, error) {
	opts := &freightclient.ListFreightFromOriginOptions{
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
