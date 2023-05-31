package handler

import (
	"google.golang.org/protobuf/proto"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func toEnvironmentProto(e kubev1alpha1.Environment) *v1alpha1.Environment {
	// Status
	availableStates := make([]*v1alpha1.EnvironmentState, len(e.Status.AvailableStates))
	for idx := range e.Status.AvailableStates {
		availableStates[idx] = toEnvironmentStateProto(e.Status.AvailableStates[idx])
	}
	var currentState *v1alpha1.EnvironmentState
	if e.Status.CurrentState != nil {
		currentState = toEnvironmentStateProto(*e.Status.CurrentState)
	}
	history := make([]*v1alpha1.EnvironmentState, len(e.Status.History))
	for idx := range e.Status.History {
		history[idx] = toEnvironmentStateProto(e.Status.History[idx])
	}

	metadata := e.ObjectMeta.DeepCopy()
	metadata.SetManagedFields(nil)

	return &v1alpha1.Environment{
		Metadata: metadata,
		Spec: &v1alpha1.EnvironmentSpec{
			Subscriptions:       toSubscriptionsProto(*e.Spec.Subscriptions),
			PromotionMechanisms: toPromotionMechanismsProto(*e.Spec.PromotionMechanisms),
		},
		Status: &v1alpha1.EnvironmentStatus{
			AvailableStates: availableStates,
			CurrentState:    currentState,
			History:         history,
			Error:           proto.String(e.Status.Error),
		},
	}
}

func toSubscriptionsProto(s kubev1alpha1.Subscriptions) *v1alpha1.Subscriptions {
	var repos *v1alpha1.RepoSubscriptions
	if s.Repos != nil {
		repos = &v1alpha1.RepoSubscriptions{
			Git:    make([]*v1alpha1.GitSubscription, len(s.Repos.Git)),
			Images: make([]*v1alpha1.ImageSubscription, len(s.Repos.Images)),
			Charts: make([]*v1alpha1.ChartSubscription, len(s.Repos.Charts)),
		}
		for idx := range s.Repos.Git {
			repos.Git[idx] = toGitSubscriptionProto(s.Repos.Git[idx])
		}
		for idx := range s.Repos.Images {
			repos.Images[idx] = toImageSubscriptionProto(s.Repos.Images[idx])
		}
		for idx := range s.Repos.Charts {
			repos.Charts[idx] = toChartSubscriptionProto(s.Repos.Charts[idx])
		}
	}

	upstreamEnvs := make([]*v1alpha1.EnvironmentSubscription, len(s.UpstreamEnvs))
	for idx := range s.UpstreamEnvs {
		upstreamEnvs[idx] = toEnvironmentSubscriptionProto(s.UpstreamEnvs[idx])
	}
	return &v1alpha1.Subscriptions{
		Repos:        repos,
		UpstreamEnvs: upstreamEnvs,
	}
}

func toGitSubscriptionProto(g kubev1alpha1.GitSubscription) *v1alpha1.GitSubscription {
	return &v1alpha1.GitSubscription{
		RepoURL: proto.String(g.RepoURL),
		Branch:  proto.String(g.Branch),
	}
}

func toImageSubscriptionProto(i kubev1alpha1.ImageSubscription) *v1alpha1.ImageSubscription {
	return &v1alpha1.ImageSubscription{
		RepoURL:          proto.String(i.RepoURL),
		UpdateStrategy:   proto.String(string(i.UpdateStrategy)),
		SemverConstraint: proto.String(i.SemverConstraint),
		AllowTags:        proto.String(i.AllowTags),
		IgnoreTags:       i.IgnoreTags,
		Platform:         proto.String(i.Platform),
	}
}

func toChartSubscriptionProto(c kubev1alpha1.ChartSubscription) *v1alpha1.ChartSubscription {
	return &v1alpha1.ChartSubscription{
		RegistryURL:      proto.String(c.RegistryURL),
		Name:             proto.String(c.Name),
		SemverConstraint: proto.String(c.SemverConstraint),
	}
}

func toEnvironmentSubscriptionProto(e kubev1alpha1.EnvironmentSubscription) *v1alpha1.EnvironmentSubscription {
	return &v1alpha1.EnvironmentSubscription{
		Name:      proto.String(e.Name),
		Namespace: proto.String(e.Namespace),
	}
}

func toPromotionMechanismsProto(p kubev1alpha1.PromotionMechanisms) *v1alpha1.PromotionMechanisms {
	gitRepoUpdates := make([]*v1alpha1.GitRepoUpdate, len(p.GitRepoUpdates))
	for idx := range p.GitRepoUpdates {
		gitRepoUpdates[idx] = toGitRepoUpdateProto(p.GitRepoUpdates[idx])
	}
	argoCDAppUpdates := make([]*v1alpha1.ArgoCDAppUpdate, len(p.ArgoCDAppUpdates))
	for idx := range p.ArgoCDAppUpdates {
		argoCDAppUpdates[idx] = toArgoCDAppUpdateProto(p.ArgoCDAppUpdates[idx])
	}
	return &v1alpha1.PromotionMechanisms{
		GitRepoUpdates:   gitRepoUpdates,
		ArgoCDAppUpdates: argoCDAppUpdates,
	}
}

func toGitRepoUpdateProto(g kubev1alpha1.GitRepoUpdate) *v1alpha1.GitRepoUpdate {
	var bookkeeper *v1alpha1.BookkeeperPromotionMechanism
	if g.Bookkeeper != nil {
		bookkeeper = toBookkeeperPromotionMechanismProto(*g.Bookkeeper)
	}
	var kustomize *v1alpha1.KustomizePromotionMechanism
	if g.Kustomize != nil {
		kustomize = toKustomizePromotionMechanismProto(*g.Kustomize)
	}
	var helm *v1alpha1.HelmPromotionMechanism
	if g.Helm != nil {
		helm = toHelmPromotionMechanismProto(*g.Helm)
	}
	return &v1alpha1.GitRepoUpdate{
		RepoURL:     proto.String(g.RepoURL),
		ReadBranch:  proto.String(g.ReadBranch),
		WriteBranch: proto.String(g.WriteBranch),
		Bookkeeper:  bookkeeper,
		Kustomize:   kustomize,
		Helm:        helm,
	}
}

func toBookkeeperPromotionMechanismProto(
	_ kubev1alpha1.BookkeeperPromotionMechanism,
) *v1alpha1.BookkeeperPromotionMechanism {
	return &v1alpha1.BookkeeperPromotionMechanism{}
}

func toKustomizePromotionMechanismProto(
	k kubev1alpha1.KustomizePromotionMechanism,
) *v1alpha1.KustomizePromotionMechanism {
	images := make([]*v1alpha1.KustomizeImageUpdate, len(k.Images))
	for idx := range k.Images {
		images[idx] = toKustomizeImageUpdateProto(k.Images[idx])
	}
	return &v1alpha1.KustomizePromotionMechanism{
		Images: images,
	}
}

func toKustomizeImageUpdateProto(k kubev1alpha1.KustomizeImageUpdate) *v1alpha1.KustomizeImageUpdate {
	return &v1alpha1.KustomizeImageUpdate{
		Image: proto.String(k.Image),
		Path:  proto.String(k.Path),
	}
}

func toHelmPromotionMechanismProto(h kubev1alpha1.HelmPromotionMechanism) *v1alpha1.HelmPromotionMechanism {
	images := make([]*v1alpha1.HelmImageUpdate, len(h.Images))
	for idx := range h.Images {
		images[idx] = toHelmImageUpdateProto(h.Images[idx])
	}
	charts := make([]*v1alpha1.HelmChartDependencyUpdate, len(h.Charts))
	for idx := range h.Charts {
		charts[idx] = toHelmChartDependencyUpdateProto(h.Charts[idx])
	}
	return &v1alpha1.HelmPromotionMechanism{
		Images: images,
		Charts: charts,
	}
}

func toHelmImageUpdateProto(h kubev1alpha1.HelmImageUpdate) *v1alpha1.HelmImageUpdate {
	return &v1alpha1.HelmImageUpdate{
		Image:          proto.String(h.Image),
		ValuesFilePath: proto.String(h.ValuesFilePath),
		Key:            proto.String(h.Key),
		Value:          proto.String(string(h.Value)),
	}
}

func toHelmChartDependencyUpdateProto(h kubev1alpha1.HelmChartDependencyUpdate) *v1alpha1.HelmChartDependencyUpdate {
	return &v1alpha1.HelmChartDependencyUpdate{
		RegistryURL: proto.String(h.RegistryURL),
		Name:        proto.String(h.Name),
		ChartPath:   proto.String(h.ChartPath),
	}
}

func toArgoCDAppUpdateProto(h kubev1alpha1.ArgoCDAppUpdate) *v1alpha1.ArgoCDAppUpdate {
	sourceUpdates := make([]*v1alpha1.ArgoCDSourceUpdate, len(h.SourceUpdates))
	for idx := range h.SourceUpdates {
		sourceUpdates[idx] = toArgoCDSourceUpdateProto(h.SourceUpdates[idx])
	}
	return &v1alpha1.ArgoCDAppUpdate{
		AppName:       proto.String(h.AppName),
		AppNamespace:  proto.String(h.AppNamespace),
		SourceUpdates: sourceUpdates,
	}
}

func toArgoCDSourceUpdateProto(a kubev1alpha1.ArgoCDSourceUpdate) *v1alpha1.ArgoCDSourceUpdate {
	var kustomize *v1alpha1.ArgoCDKustomize
	if a.Kustomize != nil {
		kustomize = toArgoCDKustomizeProto(*a.Kustomize)
	}
	var helm *v1alpha1.ArgoCDHelm
	if a.Helm != nil {
		helm = toArgoCDHelmProto(*a.Helm)
	}
	return &v1alpha1.ArgoCDSourceUpdate{
		RepoURL:              proto.String(a.RepoURL),
		Chart:                proto.String(a.Chart),
		UpdateTargetRevision: proto.Bool(a.UpdateTargetRevision),
		Kustomize:            kustomize,
		Helm:                 helm,
	}
}

func toArgoCDKustomizeProto(a kubev1alpha1.ArgoCDKustomize) *v1alpha1.ArgoCDKustomize {
	return &v1alpha1.ArgoCDKustomize{
		Images: a.Images,
	}
}

func toArgoCDHelmProto(a kubev1alpha1.ArgoCDHelm) *v1alpha1.ArgoCDHelm {
	images := make([]*v1alpha1.ArgoCDHelmImageUpdate, len(a.Images))
	for idx := range images {
		images[idx] = toArgoCDHelmImageUpdateProto(a.Images[idx])
	}
	return &v1alpha1.ArgoCDHelm{
		Images: images,
	}
}

func toArgoCDHelmImageUpdateProto(a kubev1alpha1.ArgoCDHelmImageUpdate) *v1alpha1.ArgoCDHelmImageUpdate {
	return &v1alpha1.ArgoCDHelmImageUpdate{
		Image: proto.String(a.Image),
		Key:   proto.String(a.Key),
		Value: proto.String(string(a.Value)),
	}
}

func toEnvironmentStateProto(e kubev1alpha1.EnvironmentState) *v1alpha1.EnvironmentState {
	commits := make([]*v1alpha1.GitCommit, len(e.Commits))
	for idx := range e.Commits {
		commits[idx] = toGitCommitProto(e.Commits[idx])
	}
	images := make([]*v1alpha1.Image, len(e.Images))
	for idx := range e.Images {
		images[idx] = toImageProto(e.Images[idx])
	}
	charts := make([]*v1alpha1.Chart, len(e.Charts))
	for idx := range e.Charts {
		charts[idx] = toChartProto(e.Charts[idx])
	}
	var health *v1alpha1.Health
	if e.Health != nil {
		health = toHealthProto(*e.Health)
	}
	return &v1alpha1.EnvironmentState{
		Id:         proto.String(e.ID),
		FirstSeen:  e.FirstSeen,
		Provenance: proto.String(e.Provenance),
		Commits:    commits,
		Images:     images,
		Charts:     charts,
		Health:     health,
	}
}

func toGitCommitProto(g kubev1alpha1.GitCommit) *v1alpha1.GitCommit {
	return &v1alpha1.GitCommit{
		RepoURL:           proto.String(g.RepoURL),
		Id:                proto.String(g.ID),
		Branch:            proto.String(g.Branch),
		HealthCheckCommit: proto.String(g.HealthCheckCommit),
	}
}

func toImageProto(i kubev1alpha1.Image) *v1alpha1.Image {
	return &v1alpha1.Image{
		RepoURL: proto.String(i.RepoURL),
		Tag:     proto.String(i.Tag),
	}
}

func toChartProto(c kubev1alpha1.Chart) *v1alpha1.Chart {
	return &v1alpha1.Chart{
		RegistryURL: proto.String(c.RegistryURL),
		Name:        proto.String(c.Name),
		Version:     proto.String(c.Version),
	}
}

func toHealthProto(h kubev1alpha1.Health) *v1alpha1.Health {
	return &v1alpha1.Health{
		Status:       proto.String(string(h.Status)),
		StatusReason: proto.String(h.StatusReason),
	}
}
