package api

import (
	"context"
	"fmt"
	"path"
	"slices"
	"sort"

	"connectrpc.com/connect"
	"github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
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
	reverse := req.Msg.GetReverse()

	var freight []kargoapi.Freight
	if stageName != "" {
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
			stage.Spec.Subscriptions,
		)
		if err != nil {
			return nil, fmt.Errorf("get available freight for stage: %w", err)
		}
	} else {
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
	subs kargoapi.Subscriptions,
) ([]kargoapi.Freight, error) {
	if subs.Warehouse != "" {
		return s.getFreightFromWarehouseFn(ctx, project, subs.Warehouse)
	}
	verifiedFreight, err := s.getVerifiedFreightFn(
		ctx,
		project,
		subs.UpstreamStages,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing Freight verified in Stages upstream from Stage %q in namespace %q: %w",
			stage,
			project,
			err,
		)
	}
	var approvedFreight kargoapi.FreightList
	if err = s.listFreightFn(
		ctx,
		&approvedFreight,
		&client.ListOptions{
			Namespace: project,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightApprovedForStagesIndexField,
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
	if len(verifiedFreight) == 0 &&
		len(approvedFreight.Items) == 0 {
		return nil, nil
	}
	// De-dupe
	availableFreightMap := make(
		map[string]kargoapi.Freight,
		len(verifiedFreight)+len(approvedFreight.Items),
	)
	for _, freight := range verifiedFreight {
		availableFreightMap[freight.Name] = freight
	}
	for _, freight := range approvedFreight.Items {
		availableFreightMap[freight.Name] = freight
	}
	// Turn the map to a list
	availableFreight := make([]kargoapi.Freight, len(availableFreightMap))
	var i int
	for _, freight := range availableFreightMap {
		availableFreight[i] = freight
		i++
	}
	return availableFreight, nil
}

func (s *server) getFreightFromWarehouse(
	ctx context.Context,
	project string,
	warehouse string,
) ([]kargoapi.Freight, error) {
	var freight kargoapi.FreightList
	if err := s.listFreightFn(
		ctx,
		&freight,
		&client.ListOptions{
			Namespace: project,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightByWarehouseIndexField,
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
	return freight.Items, nil
}

func (s *server) getVerifiedFreight(
	ctx context.Context,
	project string,
	stageSubs []kargoapi.StageSubscription,
) ([]kargoapi.Freight, error) {
	// Start by building a de-duped map of Freight verified in any upstream
	// Stage(s)
	verifiedFreight := map[string]kargoapi.Freight{}
	for _, stageSub := range stageSubs {
		var freight kargoapi.FreightList
		if err := s.listFreightFn(
			ctx,
			&freight,
			&client.ListOptions{
				Namespace: project,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.FreightByVerifiedStagesIndexField,
					stageSub.Name,
				),
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight verified in Stage %q in namespace %q: %w",
				stageSub.Name,
				project,
				err,
			)
		}
		for _, freight := range freight.Items {
			verifiedFreight[freight.Name] = freight
		}
	}
	if len(verifiedFreight) == 0 {
		return nil, nil
	}
	// Turn the map to a list
	verifiedFreightList := make([]kargoapi.Freight, len(verifiedFreight))
	i := 0
	for _, freight := range verifiedFreight {
		verifiedFreightList[i] = freight
		i++
	}
	return verifiedFreightList, nil
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

func sortFreightGroups(orderBy string, reverse bool, groups map[string]*svcv1alpha1.FreightList) {
	// TODO
	for k := range groups {
		var dataToSort sort.Interface
		switch orderBy {
		case OrderByTag:
			dataToSort = ByTag(groups[k].Freight)
		default:
			dataToSort = ByFirstSeen(groups[k].Freight)
		}
		if reverse {
			dataToSort = sort.Reverse(dataToSort)
		}
		sort.Sort(dataToSort)
	}
}

type ByFirstSeen []*kargoapi.Freight

func (a ByFirstSeen) Len() int      { return len(a) }
func (a ByFirstSeen) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByFirstSeen) Less(i, j int) bool {
	return a[i].CreationTimestamp.Time.Before(a[j].CreationTimestamp.Time)
}

// NOTE: sorting by tag will sort by the first container image we found
// or the first helm chart we found in the freight.
//
// TODO: KR: We might want to think about whether the current sorting behavior
// is useful at all, given the limitations noted above.
type ByTag []*kargoapi.Freight

func (a ByTag) Len() int      { return len(a) }
func (a ByTag) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTag) Less(i, j int) bool {
	iRepo, iTag, iVer := getRepoAndTag(a[i])
	jRepo, jTag, jVer := getRepoAndTag(a[j])
	// Only compare the two freight if we are comparing against the same repository
	if iRepo == jRepo {
		if iVer != nil && jVer != nil {
			return iVer.LessThan(jVer)
		}
		// repo is the same, but tags are not a semver. do lexicographical comparison
		return iTag < jTag
	}
	// They are not comparable. Fallback to firstSeen
	return a[i].CreationTimestamp.Time.Before(a[j].CreationTimestamp.Time)
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
