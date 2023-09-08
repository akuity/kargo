package api

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	apiv1alpha1 "github.com/akuity/kargo/pkg/api/v1alpha1"
)

const (
	GroupByContainerRepository = "container_repo"
	GroupByGitRepository       = "git_repo"
	GroupByHelmRepository      = "helm_repo"

	OrderByFirstSeen = "first_seen"
	OrderByTag       = "tag"
)

func (s *server) QueryFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.QueryFreightRequest],
) (*connect.Response[svcv1alpha1.QueryFreightResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}
	if err := validateGroupByOrderBy(req.Msg.GetGroup(), req.Msg.GetGroupBy(), req.Msg.GetOrderBy()); err != nil {
		return nil, err
	}

	var stages []kargoapi.Stage
	if req.Msg.GetStage() != "" {
		stage, err := getStage(ctx, s.client, req.Msg.GetProject(), req.Msg.GetStage())
		if err != nil {
			return nil, err
		}
		stages = []kargoapi.Stage{*stage}
	} else {
		var list kargoapi.StageList
		if err := s.client.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		stages = list.Items
	}

	seen := make(map[string]bool)
	freightGroups := make(map[string]*svcv1alpha1.FreightList)
	for _, s := range stages {
		addToGroups(req.Msg, freightGroups, s, seen)
	}
	sortFreightGroups(req.Msg.GetOrderBy(), req.Msg.GetReverse(), freightGroups)

	return connect.NewResponse(&svcv1alpha1.QueryFreightResponse{
		Groups: freightGroups,
	}), nil
}

func addToGroups(
	req *svcv1alpha1.QueryFreightRequest,
	groups map[string]*svcv1alpha1.FreightList,
	stage kargoapi.Stage,
	seen map[string]bool,
) {

	appendToStageGroups := func(stack kargoapi.FreightStack) {
		for _, f := range stack {
			if seen[f.ID] {
				continue
			}
			// clear out stage-specific information
			f.Qualified = false // Qualification is WRT a Stage
			f.Provenance = ""
			switch req.GetGroupBy() {
			case GroupByContainerRepository:
				for _, i := range f.Images {
					if req.GetGroup() == "" || i.RepoURL == req.GetGroup() {
						groups[i.RepoURL] = appendToFreightList(groups[i.RepoURL], f)
					}
				}
			case GroupByGitRepository:
				for _, c := range f.Commits {
					if req.GetGroup() == "" || c.RepoURL == req.GetGroup() {
						groups[c.RepoURL] = appendToFreightList(groups[c.RepoURL], f)
					}
				}
			case GroupByHelmRepository:
				for _, c := range f.Charts {
					if req.GetGroup() == "" || c.RegistryURL == req.GetGroup() {
						groups[c.RegistryURL] = appendToFreightList(groups[c.RegistryURL], f)
					}
				}
			default:
				if req.GetGroup() == "" {
					groups[""] = appendToFreightList(groups[""], f)
				}
			}
			seen[f.ID] = true
		}
	}
	appendToStageGroups(stage.Status.AvailableFreight)
	appendToStageGroups(stage.Status.History)
}

func appendToFreightList(list *svcv1alpha1.FreightList, f kargoapi.Freight) *svcv1alpha1.FreightList {
	if list == nil {
		list = &svcv1alpha1.FreightList{}
	}
	list.Freight = append(list.Freight, v1alpha1.ToFreightProto(f))
	return list
}

func sortFreightGroups(orderBy string, reverse bool, groups map[string]*svcv1alpha1.FreightList) {
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

type ByFirstSeen []*apiv1alpha1.Freight

func (a ByFirstSeen) Len() int      { return len(a) }
func (a ByFirstSeen) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByFirstSeen) Less(i, j int) bool {
	return a[i].FirstSeen.AsTime().Before(a[j].FirstSeen.AsTime())
}

// NOTE: sorting by tag will sort by the first container image we found
// or the first helm chart we found in the freight.
type ByTag []*apiv1alpha1.Freight

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
	return a[i].FirstSeen.AsTime().Before(a[j].FirstSeen.AsTime())
}

func getRepoAndTag(s *apiv1alpha1.Freight) (string, string, *semver.Version) {
	var repo, tag string
	if len(s.Images) > 0 {
		repo = s.Images[0].RepoUrl
		tag = s.Images[0].Tag
	} else if len(s.Charts) > 0 {
		repo = s.Charts[0].RegistryUrl + "/" + s.Charts[0].Name
		tag = s.Charts[0].Version
	} else {
		return "", "", nil
	}
	v, _ := semver.NewVersion(tag)
	return repo, tag, v
}
