package api

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

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
			return nil, connect.NewError(
				connect.CodeNotFound,
				fmt.Errorf(
					"Stage %q not found in namespace %q",
					stageName,
					project,
				),
			)
		}
		freight, err = s.getAvailableFreightForStageFn(
			ctx,
			project,
			stageName,
			stage.Spec.RequestedFreight,
		)
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

// getAvailableFreightForStage gets all Freight available to the specified Stage
// for any reason. This includes:
//
// 1. Any Freight from a Warehouse that the Stage subscribes to directly
// 2. Any Freight that is verified in any upstream Stages
// 3. Any Freight that is approved for the Stage
func (s *server) getAvailableFreightForStage(
	ctx context.Context,
	project string,
	stage string,
	freightReqs []kargoapi.FreightRequest,
) ([]kargoapi.Freight, error) {
	// Find all Warehouses and upstream Stages we need to consider
	var warehouses []string
	var upstreams []string
	for _, req := range freightReqs {
		if req.Sources.Direct {
			warehouses = append(warehouses, req.Origin.Name)
		}
		upstreams = append(upstreams, req.Sources.Stages...)
	}
	// De-dupe the upstreams
	slices.Sort(upstreams)
	upstreams = slices.Compact(upstreams)

	freightFromWarehouses, err := s.getFreightFromWarehousesFn(ctx, project, warehouses)
	if err != nil {
		return nil, fmt.Errorf("get freight from warehouses: %w", err)
	}

	verifiedFreight, err := s.getVerifiedFreightFn(ctx, project, upstreams)
	if err != nil {
		return nil, fmt.Errorf("get verified freight: %w", err)
	}

	var approvedFreight kargoapi.FreightList
	if err = s.listFreightFn(
		ctx,
		&approvedFreight,
		&client.ListOptions{
			Namespace: project,
			FieldSelector: fields.OneTermEqualSelector(
				indexer.FreightApprovedForStagesIndexField,
				stage,
			),
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Freight approved for Stage %q in namespace %q: %w",
			stage,
			project,
			err,
		)
	}
	if len(freightFromWarehouses) == 0 &&
		len(verifiedFreight) == 0 &&
		len(approvedFreight.Items) == 0 {
		return nil, nil
	}

	// Concatenate all available Freight
	availableFreight := append(freightFromWarehouses, verifiedFreight...)
	availableFreight = append(availableFreight, approvedFreight.Items...)

	// De-dupe the available Freight
	slices.SortFunc(availableFreight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	availableFreight = slices.CompactFunc(availableFreight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	return availableFreight, nil
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
					indexer.FreightByWarehouseIndexField,
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
					indexer.FreightByVerifiedStagesIndexField,
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

func groupByImageRepo(
	freight []kargoapi.Freight,
	group string,
) map[string]*svcv1alpha1.FreightList {
	groups := make(map[string]*svcv1alpha1.FreightList)
	for _, f := range freight {
		for _, i := range f.Images {
			if group == "" || i.RepoURL == group {
				groups[i.RepoURL] = appendToFreightList(groups[i.RepoURL], f)
			}
		}
	}
	return groups
}

func groupByGitRepo(
	freight []kargoapi.Freight,
	group string,
) map[string]*svcv1alpha1.FreightList {
	groups := make(map[string]*svcv1alpha1.FreightList)
	for _, f := range freight {
		for _, c := range f.Commits {
			if group == "" || c.RepoURL == group {
				groups[c.RepoURL] = appendToFreightList(groups[c.RepoURL], f)
			}
		}
	}
	return groups
}

func groupByChart(
	freight []kargoapi.Freight,
	group string,
) map[string]*svcv1alpha1.FreightList {
	groups := make(map[string]*svcv1alpha1.FreightList)
	for _, f := range freight {
		for _, c := range f.Charts {
			// path.Join accounts for the possibility that chart.Name is empty
			key := path.Join(c.RepoURL, c.Name)
			if group == "" || key == group {
				groups[key] = appendToFreightList(groups[key], f)
			}
		}
	}
	return groups
}

func noGroupBy(freight []kargoapi.Freight) map[string]*svcv1alpha1.FreightList {
	freightList := &svcv1alpha1.FreightList{}
	for _, f := range freight {
		freightList = appendToFreightList(freightList, f)
	}
	return map[string]*svcv1alpha1.FreightList{
		"": freightList,
	}
}

func appendToFreightList(list *svcv1alpha1.FreightList, f kargoapi.Freight) *svcv1alpha1.FreightList {
	if list == nil {
		list = &svcv1alpha1.FreightList{}
	}
	list.Freight = append(list.Freight, &f)
	return list
}

// NOTE: sorting by tag will sort by the first container image we found
// or the first helm chart we found in the freight.
//
// TODO: KR: We might want to think about whether the current sorting behavior
// is useful at all, given the limitations noted above.
func sortFreightGroups(orderBy string, reverse bool, groups map[string]*svcv1alpha1.FreightList) {
	for k := range groups {
		slices.SortFunc(groups[k].Freight, func(lhs, rhs *kargoapi.Freight) int {
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
			return lhs.CreationTimestamp.Time.Compare(rhs.CreationTimestamp.Time)
		})
		if reverse {
			slices.Reverse(groups[k].Freight)
		}
	}
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
