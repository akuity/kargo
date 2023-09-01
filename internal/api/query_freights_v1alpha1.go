package api

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	apiv1alpha1 "github.com/akuity/kargo/pkg/api/v1alpha1"
)

const (
	GroupByContainerRepository = "container_repo"
	GroupByGitRepository       = "git_repo"
	GroupByHelmRepository      = "helm_repo"

	OrderByFirstSeen       = "first_seen"
	OrderBySemanticVersion = "semantic_version"
)

func (s *server) QueryFreights(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.QueryFreightsRequest],
) (*connect.Response[svcv1alpha1.QueryFreightsResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}
	if err := validateGroupBy(req.Msg.GetGroupBy()); err != nil {
		return nil, err
	}
	if err := validateOrderBy(req.Msg.GetOrderBy()); err != nil {
		return nil, err
	}

	var stages []kubev1alpha1.Stage
	if req.Msg.GetStage() != "" {
		stage, err := getStage(ctx, s.client, req.Msg.GetProject(), req.Msg.GetStage())
		if err != nil {
			return nil, err
		}
		stages = append(stages, *stage)
	} else {
		var list kubev1alpha1.StageList
		if err := s.client.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		stages = list.Items
	}

	seen := make(map[string]kubev1alpha1.StageState)
	freightGroups := make(map[string]*svcv1alpha1.FreightList)
	for _, s := range stages {
		addToGroups(req.Msg, freightGroups, s, seen)
	}
	sortFreightGroups(req.Msg.GetOrderBy(), freightGroups)

	return connect.NewResponse(&svcv1alpha1.QueryFreightsResponse{
		Groups: freightGroups,
	}), nil
}

func addToGroups(
	req *svcv1alpha1.QueryFreightsRequest,
	groups map[string]*svcv1alpha1.FreightList,
	stage kubev1alpha1.Stage,
	seen map[string]kubev1alpha1.StageState,
) {

	appendToStageGroups := func(stack kubev1alpha1.StageStateStack) {
		for _, f := range stack {
			if _, ok := seen[f.ID]; ok {
				continue
			}
			// clear out state-specific information
			f.Health = nil
			f.Provenance = ""
			switch req.GetGroupBy() {
			case GroupByContainerRepository:
				for _, i := range f.Images {
					groups[i.RepoURL] = appendToFreightList(groups[i.RepoURL], f)
				}
			case GroupByGitRepository:
				for _, c := range f.Commits {
					groups[c.RepoURL] = appendToFreightList(groups[c.RepoURL], f)
				}
			case GroupByHelmRepository:
				for _, c := range f.Charts {
					groups[c.RegistryURL] = appendToFreightList(groups[c.RegistryURL], f)
				}
			default:
				groups[""] = appendToFreightList(groups[""], f)
			}
			seen[f.ID] = f
		}
	}
	appendToStageGroups(stage.Status.AvailableStates)
	appendToStageGroups(stage.Status.History)
}

func appendToFreightList(list *svcv1alpha1.FreightList, f kubev1alpha1.StageState) *svcv1alpha1.FreightList {
	if list == nil {
		list = &svcv1alpha1.FreightList{}
	}
	list.Freights = append(list.Freights, v1alpha1.ToStageStateProto(f))
	return list
}

func sortFreightGroups(orderBy string, groups map[string]*svcv1alpha1.FreightList) {
	for k := range groups {
		switch orderBy {
		case OrderBySemanticVersion:
			sort.Sort(BySemanticVersion(groups[k].Freights))
		case OrderByFirstSeen, "":
			sort.Sort(ByFirstSeen(groups[k].Freights))
		default:
			sort.Sort(ByFirstSeen(groups[k].Freights))
		}
	}
}

type ByFirstSeen []*apiv1alpha1.StageState

func (a ByFirstSeen) Len() int      { return len(a) }
func (a ByFirstSeen) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByFirstSeen) Less(i, j int) bool {
	return a[i].FirstSeen.AsTime().Before(a[j].FirstSeen.AsTime())
}

type BySemanticVersion []*apiv1alpha1.StageState

func (a BySemanticVersion) Len() int      { return len(a) }
func (a BySemanticVersion) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySemanticVersion) Less(i, j int) bool {
	// TODO: implement semantic version sorting
	return a[i].FirstSeen.AsTime().Before(a[j].FirstSeen.AsTime())
}
