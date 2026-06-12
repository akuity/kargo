package server

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

type freightGroupList struct {
	Items []*kargoapi.Freight `json:"items"`
}

type queryFreightsResponse struct {
	Groups map[string]freightGroupList `json:"groups"`
	// ResourceVersion is the Kubernetes list resourceVersion clients use to
	// seed a follow-up Freight watch so the API server does not replay every
	// existing Freight as an ADDED event. It is empty for the stage-scoped
	// query, whose result is assembled from multiple sources rather than a
	// single watchable namespace list.
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

const (
	GroupByImageRepository = "image_repo"
	GroupByGitRepository   = "git_repo"
	GroupByChartRepository = "chart_repo"

	OrderByFirstSeen = "first_seen"
	OrderByTag       = "tag"
	// TODO: KR: Maybe we should add OrderBySemVer since charts are always
	// semantically versioned and images sometimes are.
)

// QueryFreight retrieves and tabulates Freight according to the criteria
// specified in the request.
func (s *server) QueryFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.QueryFreightRequest],
) (*connect.Response[svcv1alpha1.QueryFreightResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	group := req.Msg.GetGroup()
	groupBy := req.Msg.GetGroupBy()
	orderBy := req.Msg.GetOrderBy()
	if err := validateGroupByOrderBy(group, groupBy, orderBy); err != nil {
		return nil, err // This already returns a connect.Error
	}

	if err := s.validateProjectExistsFn(ctx, project); err != nil {
		return nil, err // This already returns a connect.Error
	}

	stageName := req.Msg.GetStage()
	origins := req.Msg.GetOrigins()
	reverse := req.Msg.GetReverse()

	var freight []kargoapi.Freight
	switch {
	case stageName != "":
		stage, err := s.getStageFn(
			ctx,
			s.client,
			types.NamespacedName{
				Namespace: project,
				Name:      stageName,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("get stage: %w", err)
		}
		if stage == nil {
			// nolint:staticcheck
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf(
					"Stage %q not found in namespace %q",
					stageName,
					project,
				),
			)
		}
		freight, err = s.getAvailableFreightForStageFn(ctx, stage)
		if err != nil {
			return nil, fmt.Errorf("get available freight for stage: %w", err)
		}
	case len(origins) > 0:
		var err error
		freight, err = s.getFreightFromWarehousesFn(ctx, project, origins)
		if err != nil {
			return nil, fmt.Errorf("get freight from warehouse: %w", err)
		}
	default:
		freightList := &kargoapi.FreightList{}
		// Get ALL Freight in the project/namespace
		if err := s.listFreightFn(
			ctx,
			freightList,
			client.InNamespace(project),
		); err != nil {
			return nil, fmt.Errorf("list freight: %w", err)
		}
		freight = freightList.Items
	}

	// Split the Freight into groups
	var freightGroups map[string]*svcv1alpha1.FreightList
	switch groupBy {
	case GroupByImageRepository:
		freightGroups = groupByImageRepo(freight, group)
	case GroupByGitRepository:
		freightGroups = groupByGitRepo(freight, group)
	case GroupByChartRepository:
		freightGroups = groupByChart(freight, group)
	default:
		freightGroups = noGroupBy(freight)
	}

	sortFreightGroups(orderBy, reverse, freightGroups)

	return connect.NewResponse(&svcv1alpha1.QueryFreightResponse{
		Groups: freightGroups,
	}), nil
}

func (s *server) getAvailableFreightForStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) ([]kargoapi.Freight, error) {
	return api.ListFreightAvailableToStage(ctx, s.client, stage)
}

func (s *server) getFreightFromWarehouses(
	ctx context.Context,
	project string,
	warehouses []string,
) ([]kargoapi.Freight, error) {
	var allFreight []kargoapi.Freight
	for _, warehouse := range warehouses {
		var freight kargoapi.FreightList
		if err := s.listFreightFn(
			ctx,
			&freight,
			&client.ListOptions{
				Namespace: project,
				FieldSelector: fields.OneTermEqualSelector(
					indexer.FreightByWarehouseField,
					warehouse,
				),
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight for Warehouse %q in namespace %q: %w",
				warehouse,
				project,
				err,
			)
		}
		allFreight = append(allFreight, freight.Items...)
	}
	return allFreight, nil
}

func (s *server) getVerifiedFreight(
	ctx context.Context,
	project string,
	upstreams []string,
) ([]kargoapi.Freight, error) {
	var verifiedFreight []kargoapi.Freight
	for _, upstream := range upstreams {
		var freight kargoapi.FreightList
		if err := s.listFreightFn(
			ctx,
			&freight,
			&client.ListOptions{
				Namespace: project,
				FieldSelector: fields.OneTermEqualSelector(
					indexer.FreightByVerifiedStagesField,
					upstream,
				),
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight verified in Stage %q in namespace %q: %w",
				upstream,
				project,
				err,
			)
		}
		verifiedFreight = append(verifiedFreight, freight.Items...)
	}
	if len(verifiedFreight) == 0 {
		return nil, nil
	}
	// De-dupe the verified Freight
	slices.SortFunc(verifiedFreight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	verifiedFreight = slices.CompactFunc(verifiedFreight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	return verifiedFreight, nil
}

// groupFreightByImageRepo groups freight by image repository URL.
func groupFreightByImageRepo(
	freight []kargoapi.Freight,
	group string,
) map[string][]*kargoapi.Freight {
	groups := make(map[string][]*kargoapi.Freight)
	for _, f := range freight {
		for _, i := range f.Images {
			if group == "" || i.RepoURL == group {
				fCopy := f
				groups[i.RepoURL] = append(groups[i.RepoURL], &fCopy)
			}
		}
	}
	return groups
}

// groupFreightByGitRepo groups freight by git repository URL.
func groupFreightByGitRepo(
	freight []kargoapi.Freight,
	group string,
) map[string][]*kargoapi.Freight {
	groups := make(map[string][]*kargoapi.Freight)
	for _, f := range freight {
		for _, c := range f.Commits {
			if group == "" || c.RepoURL == group {
				fCopy := f
				groups[c.RepoURL] = append(groups[c.RepoURL], &fCopy)
			}
		}
	}
	return groups
}

// groupFreightByChart groups freight by chart repository URL and name.
func groupFreightByChart(
	freight []kargoapi.Freight,
	group string,
) map[string][]*kargoapi.Freight {
	groups := make(map[string][]*kargoapi.Freight)
	for _, f := range freight {
		for _, c := range f.Charts {
			// path.Join accounts for the possibility that chart.Name is empty
			key := path.Join(c.RepoURL, c.Name)
			if group == "" || key == group {
				fCopy := f
				groups[key] = append(groups[key], &fCopy)
			}
		}
	}
	return groups
}

// groupFreight groups freight into a single group with empty key (no grouping).
func groupFreight(freight []kargoapi.Freight) map[string][]*kargoapi.Freight {
	freightPtrs := make([]*kargoapi.Freight, len(freight))
	for i := range freight {
		freightPtrs[i] = &freight[i]
	}
	return map[string][]*kargoapi.Freight{
		"": freightPtrs,
	}
}

// sortFreightSlice sorts a slice of freight by the specified order.
//
// NOTE: sorting by tag will sort by the first container image we found
// or the first helm chart we found in the freight.
//
// TODO: KR: We might want to think about whether the current sorting behavior
// is useful at all, given the limitations noted above.
func sortFreightSlice(orderBy string, reverse bool, freight []*kargoapi.Freight) {
	slices.SortFunc(freight, func(lhs, rhs *kargoapi.Freight) int {
		if orderBy == OrderByTag {
			lhsRepo, lhsTag, lhsVer := getRepoAndTag(lhs)
			rhsRepo, rhsTag, rhsVer := getRepoAndTag(rhs)
			// Only compare by tag if the repos are the same
			if lhsRepo == rhsRepo {
				if lhsVer != nil && rhsVer != nil {
					return lhsVer.Compare(rhsVer)
				}
				return strings.Compare(lhsTag, rhsTag)
			}
		}
		return lhs.CreationTimestamp.Compare(rhs.CreationTimestamp.Time)
	})
	if reverse {
		slices.Reverse(freight)
	}
}

// sortFreightGroupsGeneric sorts all groups in the map.
func sortFreightGroupsGeneric(orderBy string, reverse bool, groups map[string][]*kargoapi.Freight) {
	for k := range groups {
		sortFreightSlice(orderBy, reverse, groups[k])
	}
}

// Legacy helper functions for Connect RPC endpoint compatibility.
// These wrap the generic functions and convert to svcv1alpha1.FreightList.

func groupByImageRepo(freight []kargoapi.Freight, group string) map[string]*svcv1alpha1.FreightList {
	return toSvcFreightListMap(groupFreightByImageRepo(freight, group))
}

func groupByGitRepo(freight []kargoapi.Freight, group string) map[string]*svcv1alpha1.FreightList {
	return toSvcFreightListMap(groupFreightByGitRepo(freight, group))
}

func groupByChart(freight []kargoapi.Freight, group string) map[string]*svcv1alpha1.FreightList {
	return toSvcFreightListMap(groupFreightByChart(freight, group))
}

func noGroupBy(freight []kargoapi.Freight) map[string]*svcv1alpha1.FreightList {
	return toSvcFreightListMap(groupFreight(freight))
}

func sortFreightGroups(orderBy string, reverse bool, groups map[string]*svcv1alpha1.FreightList) {
	for k := range groups {
		sortFreightSlice(orderBy, reverse, groups[k].Freight)
	}
}

// toSvcFreightListMap converts the generic freight groups to svcv1alpha1.FreightList format.
func toSvcFreightListMap(groups map[string][]*kargoapi.Freight) map[string]*svcv1alpha1.FreightList {
	result := make(map[string]*svcv1alpha1.FreightList, len(groups))
	for k, v := range groups {
		result[k] = &svcv1alpha1.FreightList{Freight: v}
	}
	return result
}

func getRepoAndTag(s *kargoapi.Freight) (string, string, *semver.Version) {
	var repo, tag string
	if len(s.Images) > 0 {
		repo = s.Images[0].RepoURL
		tag = s.Images[0].Tag
	} else if len(s.Charts) > 0 {
		// path.Join accounts for the possibility that chart.Name is empty
		repo = path.Join(s.Charts[0].RepoURL, s.Charts[0].Name)
		tag = s.Charts[0].Version
	} else {
		return "", "", nil
	}
	v, _ := semver.NewVersion(tag)
	return repo, tag, v
}

// @id QueryFreightsRest
// @Summary Query Freight
// @Description Query and filter Freight resources from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage query string false "Stage name to get available freight for"
// @Param origins query []string false "Warehouse names to get freight from"
// @Param group query string false "Group filter"
// @Param groupBy query string false "Group by (image_repo, git_repo, chart_repo)"
// @Param orderBy query string false "Order by (first_seen, tag)"
// @Param reverse query bool false "Reverse order"
// @Success 200 {object} queryFreightsResponse
// @Router /v1beta1/projects/{project}/freight [get]
func (s *server) queryFreight(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	stageName := c.Query("stage")
	group := c.Query("group")
	groupBy := c.Query("groupBy")
	orderBy := c.Query("orderBy")
	reverse := c.Query("reverse") == "true"

	// Get origins from query parameters (can be multiple)
	origins := c.QueryArray("origins")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchFreight(c, project, origins, c.Query("resourceVersion"))
		return
	}

	// Validate groupBy and orderBy
	if err := validateGroupByOrderBy(group, groupBy, orderBy); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return
	}

	var freight []kargoapi.Freight
	var resourceVersion string
	var err error

	switch {
	case stageName != "":
		// Get freight available to a specific stage
		stage := &kargoapi.Stage{}
		if err = s.client.Get(ctx, client.ObjectKey{Namespace: project, Name: stageName}, stage); err != nil {
			if apierrors.IsNotFound(err) {
				_ = c.Error(libhttp.ErrorStr(
					fmt.Sprintf("Stage %q not found in project %q", stageName, project),
					http.StatusNotFound,
				))
				return
			}
			_ = c.Error(err)
			return
		}
		freight, err = api.ListFreightAvailableToStage(ctx, s.client, stage)
		if err != nil {
			_ = c.Error(fmt.Errorf("get available freight for stage: %w", err))
			return
		}
		// The stage-availability list is assembled from multiple sources rather
		// than a single watchable namespace list, so ResourceVersion is left
		// empty and clients fall back to an unseeded watch for this view.

	case len(origins) > 0:
		// Get freight from specific warehouses. The seed list goes through the
		// uncached reader for a watchable ResourceVersion, then origins are
		// filtered in-process (see filterFreightByOrigins) to match the
		// unfiltered follow-up watch.
		freightList := &kargoapi.FreightList{}
		if err = s.listForWatchSeed(ctx, "freights", freightList, client.InNamespace(project)); err != nil {
			_ = c.Error(fmt.Errorf("list freight: %w", err))
			return
		}
		freight = filterFreightByOrigins(freightList.Items, origins)
		resourceVersion = resourceVersionForFreightList(freightList)

	default:
		// Get all freight in the project
		freightList := &kargoapi.FreightList{}
		if err = s.listForWatchSeed(ctx, "freights", freightList, client.InNamespace(project)); err != nil {
			_ = c.Error(fmt.Errorf("list freight: %w", err))
			return
		}
		freight = freightList.Items
		resourceVersion = resourceVersionForFreightList(freightList)
	}

	// Split the Freight into groups using the generic functions
	var freightGroups map[string][]*kargoapi.Freight
	switch groupBy {
	case GroupByImageRepository:
		freightGroups = groupFreightByImageRepo(freight, group)
	case GroupByGitRepository:
		freightGroups = groupFreightByGitRepo(freight, group)
	case GroupByChartRepository:
		freightGroups = groupFreightByChart(freight, group)
	default:
		freightGroups = groupFreight(freight)
	}

	sortFreightGroupsGeneric(orderBy, reverse, freightGroups)

	result := make(map[string]freightGroupList, len(freightGroups))
	for k, v := range freightGroups {
		result[k] = freightGroupList{Items: v}
	}

	c.JSON(http.StatusOK, queryFreightsResponse{
		Groups:          result,
		ResourceVersion: resourceVersion,
	})
}

// watchFreight streams Freight changes through the REST SSE endpoint, seeding
// the watch from the provided resourceVersion so the API server does not replay
// every existing Freight as an ADDED event.
func (s *server) watchFreight(c *gin.Context, project string, origins []string, resourceVersion string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	w, err := s.client.Watch(
		ctx,
		&kargoapi.FreightList{},
		buildWatchListOptions(project, resourceVersion)...,
	)
	if err != nil {
		if sendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch freight: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	setSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !writeSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}
			if watchErr := errorFromWatchEvent(e); watchErr != nil {
				sendSSEWatchError(c, watchErr)
				return
			}

			freight, ok := convertWatchEventObject(c, e, (*kargoapi.Freight)(nil))
			if !ok {
				continue
			}

			eventType := e.Type
			if len(origins) > 0 {
				var send bool
				eventType, send = filteredWatchEventType(
					e.Type,
					slices.Contains(origins, freight.Origin.Name),
				)
				if !send {
					continue
				}
			}

			if !sendSSEWatchEvent(c, eventType, freight) {
				return
			}
		}
	}
}

// filterFreightByOrigins returns Freight whose origin Warehouse is one of the
// provided origins.
//
// Origin filtering is done in-process rather than via the FreightByWarehouse
// field index because the watch-seed list goes through listForWatchSeed's
// uncached reader, which cannot serve controller-runtime field indexes. This
// over-fetches the whole namespace, but it matches the (also unfiltered)
// follow-up watch and keeps the returned ResourceVersion watchable.
func filterFreightByOrigins(freight []kargoapi.Freight, origins []string) []kargoapi.Freight {
	filtered := make([]kargoapi.Freight, 0, len(freight))
	for _, f := range freight {
		if slices.Contains(origins, f.Origin.Name) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// resourceVersionForFreightList returns the list ResourceVersion when useful,
// otherwise it falls back to the maximum Freight item ResourceVersion.
func resourceVersionForFreightList(list *kargoapi.FreightList) string {
	return effectiveResourceVersionForList(
		list.ResourceVersion,
		list.Items,
		func(f kargoapi.Freight) string { return f.ResourceVersion },
	)
}
