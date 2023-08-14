package v1alpha1

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	kubemetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	typesmetav1 "github.com/akuity/kargo/internal/api/types/metav1"
	"github.com/akuity/kargo/internal/version"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func FromStageProto(s *v1alpha1.Stage) *kubev1alpha1.Stage {
	if s == nil {
		return nil
	}
	var status kubev1alpha1.StageStatus
	if s.GetStatus() != nil {
		status = *FromStageStatusProto(s.GetStatus())
	}
	var objectMeta kubemetav1.ObjectMeta
	if s.GetMetadata() != nil {
		objectMeta = *typesmetav1.FromObjectMetaProto(s.GetMetadata())
	}
	return &kubev1alpha1.Stage{
		TypeMeta: kubemetav1.TypeMeta{
			APIVersion: kubev1alpha1.GroupVersion.String(),
			Kind:       "Stage",
		},
		ObjectMeta: objectMeta,
		Spec:       FromStageSpecProto(s.GetSpec()),
		Status:     status,
	}
}

func FromStageSpecProto(s *v1alpha1.StageSpec) *kubev1alpha1.StageSpec {
	return &kubev1alpha1.StageSpec{
		Subscriptions:       FromSubscriptionsProto(s.GetSubscriptions()),
		PromotionMechanisms: FromPromotionMechanismsProto(s.GetPromotionMechanisms()),
	}
}

func FromStageStatusProto(s *v1alpha1.StageStatus) *kubev1alpha1.StageStatus {
	if s == nil {
		return nil
	}
	availableStates := make(kubev1alpha1.StageStateStack, len(s.GetAvailableStates()))
	for idx, state := range s.GetAvailableStates() {
		availableStates[idx] = *FromStageStateProto(state)
	}
	history := make(kubev1alpha1.StageStateStack, len(s.GetHistory()))
	for idx, state := range s.GetHistory() {
		history[idx] = *FromStageStateProto(state)
	}
	return &kubev1alpha1.StageStatus{
		AvailableStates: availableStates,
		CurrentState:    FromStageStateProto(s.GetCurrentState()),
		History:         history,
		Error:           s.GetError(),
	}
}

func FromStageStateProto(s *v1alpha1.StageState) *kubev1alpha1.StageState {
	if s == nil {
		return nil
	}
	var firstSeen *kubemetav1.Time
	if s.GetFirstSeen() != nil {
		fs := kubemetav1.NewTime(s.GetFirstSeen().AsTime())
		firstSeen = &fs
	}
	commits := make([]kubev1alpha1.GitCommit, len(s.GetCommits()))
	for idx, commit := range s.GetCommits() {
		commits[idx] = *FromGitCommitProto(commit)
	}
	images := make([]kubev1alpha1.Image, len(s.GetImages()))
	for idx, image := range s.GetImages() {
		images[idx] = *FromImageProto(image)
	}
	charts := make([]kubev1alpha1.Chart, len(s.GetCharts()))
	for idx, chart := range s.GetCharts() {
		charts[idx] = *FromChartProto(chart)
	}
	return &kubev1alpha1.StageState{
		ID:         s.GetId(),
		FirstSeen:  firstSeen,
		Provenance: s.GetProvenance(),
		Commits:    commits,
		Images:     images,
		Charts:     charts,
		Health:     FromHealthProto(s.GetHealth()),
	}
}

func FromGitCommitProto(g *v1alpha1.GitCommit) *kubev1alpha1.GitCommit {
	if g == nil {
		return nil
	}
	return &kubev1alpha1.GitCommit{
		RepoURL:           g.GetRepoUrl(),
		ID:                g.GetId(),
		Branch:            g.GetBranch(),
		HealthCheckCommit: g.GetHealthCheckCommit(),
	}
}

func FromImageProto(i *v1alpha1.Image) *kubev1alpha1.Image {
	if i == nil {
		return nil
	}
	return &kubev1alpha1.Image{
		RepoURL: i.GetRepoUrl(),
		Tag:     i.GetTag(),
	}
}

func FromChartProto(c *v1alpha1.Chart) *kubev1alpha1.Chart {
	if c == nil {
		return nil
	}
	return &kubev1alpha1.Chart{
		RegistryURL: c.GetRegistryUrl(),
		Name:        c.GetName(),
		Version:     c.GetVersion(),
	}
}

func FromHealthProto(h *v1alpha1.Health) *kubev1alpha1.Health {
	if h == nil {
		return nil
	}
	return &kubev1alpha1.Health{
		Status: kubev1alpha1.HealthState(h.GetStatus()),
		Issues: h.GetIssues(),
	}
}

func FromSubscriptionsProto(s *v1alpha1.Subscriptions) *kubev1alpha1.Subscriptions {
	if s == nil {
		return nil
	}
	upstreamStages := make([]kubev1alpha1.StageSubscription, len(s.GetUpstreamStages()))
	for idx, stage := range s.GetUpstreamStages() {
		upstreamStages[idx] = *FromStageSubscriptionProto(stage)
	}
	return &kubev1alpha1.Subscriptions{
		Repos:          FromRepoSubscriptionsProto(s.GetRepos()),
		UpstreamStages: upstreamStages,
	}
}

func FromRepoSubscriptionsProto(s *v1alpha1.RepoSubscriptions) *kubev1alpha1.RepoSubscriptions {
	if s == nil {
		return nil
	}
	gitSubscriptions := make([]kubev1alpha1.GitSubscription, len(s.GetGit()))
	for idx, git := range s.GetGit() {
		gitSubscriptions[idx] = *FromGitSubscriptionProto(git)
	}
	imageSubscriptions := make([]kubev1alpha1.ImageSubscription, len(s.GetImages()))
	for idx, image := range s.GetImages() {
		imageSubscriptions[idx] = *FromImageSubscriptionProto(image)
	}
	chartSubscriptions := make([]kubev1alpha1.ChartSubscription, len(s.GetCharts()))
	for idx, chart := range s.GetCharts() {
		chartSubscriptions[idx] = *FromChartSubscriptionProto(chart)
	}
	return &kubev1alpha1.RepoSubscriptions{
		Git:    gitSubscriptions,
		Images: imageSubscriptions,
		Charts: chartSubscriptions,
	}
}

func FromGitSubscriptionProto(s *v1alpha1.GitSubscription) *kubev1alpha1.GitSubscription {
	if s == nil {
		return nil
	}
	return &kubev1alpha1.GitSubscription{
		RepoURL: s.GetRepoUrl(),
		Branch:  s.GetBranch(),
	}
}

func FromImageSubscriptionProto(s *v1alpha1.ImageSubscription) *kubev1alpha1.ImageSubscription {
	if s == nil {
		return nil
	}
	return &kubev1alpha1.ImageSubscription{
		RepoURL:          s.GetRepoUrl(),
		UpdateStrategy:   kubev1alpha1.ImageUpdateStrategy(s.GetUpdateStrategy()),
		SemverConstraint: s.GetSemverConstraint(),
		AllowTags:        s.GetAllowTags(),
		IgnoreTags:       s.GetIgnoreTags(),
		Platform:         s.GetPlatform(),
	}
}

func FromChartSubscriptionProto(s *v1alpha1.ChartSubscription) *kubev1alpha1.ChartSubscription {
	if s == nil {
		return nil
	}
	return &kubev1alpha1.ChartSubscription{
		RegistryURL:      s.GetRegistryUrl(),
		Name:             s.GetName(),
		SemverConstraint: s.GetSemverConstraint(),
	}
}

func FromPromotionMechanismsProto(m *v1alpha1.PromotionMechanisms) *kubev1alpha1.PromotionMechanisms {
	if m == nil {
		return nil
	}
	gitUpdates := make([]kubev1alpha1.GitRepoUpdate, len(m.GetGitRepoUpdates()))
	for idx, git := range m.GetGitRepoUpdates() {
		gitUpdates[idx] = *FromGitRepoUpdateProto(git)
	}
	argoUpdates := make([]kubev1alpha1.ArgoCDAppUpdate, len(m.GetArgocdAppUpdates()))
	for idx, argo := range m.GetArgocdAppUpdates() {
		argoUpdates[idx] = *FromArgoCDAppUpdatesProto(argo)
	}
	return &kubev1alpha1.PromotionMechanisms{
		GitRepoUpdates:   gitUpdates,
		ArgoCDAppUpdates: argoUpdates,
	}
}

func FromGitRepoUpdateProto(u *v1alpha1.GitRepoUpdate) *kubev1alpha1.GitRepoUpdate {
	if u == nil {
		return nil
	}
	return &kubev1alpha1.GitRepoUpdate{
		RepoURL:     u.GetRepoUrl(),
		ReadBranch:  u.GetReadBranch(),
		WriteBranch: u.GetWriteBranch(),
		Bookkeeper:  FromBookkeeperPromotionMechanismProto(u.GetBookkeeper()),
		Kustomize:   FromKustomizePromotionMechanismProto(u.GetKustomize()),
		Helm:        FromHelmPromotionMechanismProto(u.GetHelm()),
	}
}

func FromBookkeeperPromotionMechanismProto(
	m *v1alpha1.BookkeeperPromotionMechanism,
) *kubev1alpha1.BookkeeperPromotionMechanism {
	if m == nil {
		return nil
	}
	return &kubev1alpha1.BookkeeperPromotionMechanism{}
}

func FromKustomizePromotionMechanismProto(
	m *v1alpha1.KustomizePromotionMechanism,
) *kubev1alpha1.KustomizePromotionMechanism {
	if m == nil {
		return nil
	}
	images := make([]kubev1alpha1.KustomizeImageUpdate, len(m.GetImages()))
	for idx, image := range m.GetImages() {
		images[idx] = *FromKustomizeImageUpdateProto(image)
	}
	return &kubev1alpha1.KustomizePromotionMechanism{
		Images: images,
	}
}

func FromKustomizeImageUpdateProto(u *v1alpha1.KustomizeImageUpdate) *kubev1alpha1.KustomizeImageUpdate {
	if u == nil {
		return nil
	}
	return &kubev1alpha1.KustomizeImageUpdate{
		Image: u.GetImage(),
		Path:  u.GetPath(),
	}
}

func FromHelmPromotionMechanismProto(
	m *v1alpha1.HelmPromotionMechanism,
) *kubev1alpha1.HelmPromotionMechanism {
	if m == nil {
		return nil
	}
	images := make([]kubev1alpha1.HelmImageUpdate, len(m.GetImages()))
	for idx, image := range m.GetImages() {
		images[idx] = *FromHelmImageUpdateProto(image)
	}
	charts := make([]kubev1alpha1.HelmChartDependencyUpdate, len(m.GetCharts()))
	for idx, chart := range m.GetCharts() {
		charts[idx] = *FromHelmChartDependencyUpdateProto(chart)
	}
	return &kubev1alpha1.HelmPromotionMechanism{
		Images: images,
		Charts: charts,
	}
}

func FromHelmImageUpdateProto(u *v1alpha1.HelmImageUpdate) *kubev1alpha1.HelmImageUpdate {
	if u == nil {
		return nil
	}
	return &kubev1alpha1.HelmImageUpdate{
		Image:          u.GetImage(),
		ValuesFilePath: u.GetValuesFilePath(),
		Key:            u.GetKey(),
		Value:          kubev1alpha1.ImageUpdateValueType(u.GetValue()),
	}
}

func FromHelmChartDependencyUpdateProto(
	u *v1alpha1.HelmChartDependencyUpdate,
) *kubev1alpha1.HelmChartDependencyUpdate {
	if u == nil {
		return nil
	}
	return &kubev1alpha1.HelmChartDependencyUpdate{
		RegistryURL: u.GetRegistryUrl(),
		Name:        u.GetName(),
		ChartPath:   u.GetChartPath(),
	}
}

func FromArgoCDAppUpdatesProto(u *v1alpha1.ArgoCDAppUpdate) *kubev1alpha1.ArgoCDAppUpdate {
	if u == nil {
		return nil
	}
	sourceUpdates := make([]kubev1alpha1.ArgoCDSourceUpdate, len(u.GetSourceUpdates()))
	for idx, update := range u.GetSourceUpdates() {
		sourceUpdates[idx] = *FromArgoCDSourceUpdateProto(update)
	}
	return &kubev1alpha1.ArgoCDAppUpdate{
		AppName:       u.GetAppName(),
		AppNamespace:  u.GetAppNamespace(),
		SourceUpdates: sourceUpdates,
	}
}

func FromArgoCDSourceUpdateProto(u *v1alpha1.ArgoCDSourceUpdate) *kubev1alpha1.ArgoCDSourceUpdate {
	if u == nil {
		return nil
	}
	return &kubev1alpha1.ArgoCDSourceUpdate{
		RepoURL:              u.GetRepoUrl(),
		Chart:                u.GetChart(),
		UpdateTargetRevision: u.GetUpdateTargetRevision(),
		Kustomize:            FromArgoCDKustomizeProto(u.GetKustomize()),
		Helm:                 FromArgoCDHelm(u.GetHelm()),
	}
}

func FromArgoCDKustomizeProto(k *v1alpha1.ArgoCDKustomize) *kubev1alpha1.ArgoCDKustomize {
	if k == nil {
		return nil
	}
	return &kubev1alpha1.ArgoCDKustomize{
		Images: k.GetImages(),
	}
}

func FromArgoCDHelm(h *v1alpha1.ArgoCDHelm) *kubev1alpha1.ArgoCDHelm {
	if h == nil {
		return nil
	}
	images := make([]kubev1alpha1.ArgoCDHelmImageUpdate, len(h.GetImages()))
	for idx, image := range h.GetImages() {
		images[idx] = *FromArgoCDHelmImageUpdateProto(image)
	}
	return &kubev1alpha1.ArgoCDHelm{
		Images: images,
	}
}

func FromArgoCDHelmImageUpdateProto(u *v1alpha1.ArgoCDHelmImageUpdate) *kubev1alpha1.ArgoCDHelmImageUpdate {
	if u == nil {
		return nil
	}
	return &kubev1alpha1.ArgoCDHelmImageUpdate{
		Image: u.GetImage(),
		Key:   u.GetKey(),
		Value: kubev1alpha1.ImageUpdateValueType(u.GetValue()),
	}
}

func FromStageSubscriptionProto(s *v1alpha1.StageSubscription) *kubev1alpha1.StageSubscription {
	if s == nil {
		return nil
	}
	return &kubev1alpha1.StageSubscription{
		Name: s.GetName(),
	}
}

func FromPromotionProto(p *v1alpha1.Promotion) *kubev1alpha1.Promotion {
	if p == nil {
		return nil
	}
	var status kubev1alpha1.PromotionStatus
	if p.GetStatus() != nil {
		status = *FromPromotionStatusProto(p.GetStatus())
	}
	var objectMeta kubemetav1.ObjectMeta
	if p.GetMetadata() != nil {
		objectMeta = *typesmetav1.FromObjectMetaProto(p.GetMetadata())
	}
	return &kubev1alpha1.Promotion{
		TypeMeta: kubemetav1.TypeMeta{
			APIVersion: kubev1alpha1.GroupVersion.String(),
			Kind:       "Promotion",
		},
		ObjectMeta: objectMeta,
		Spec:       FromPromotionSpecProto(p.GetSpec()),
		Status:     status,
	}
}

func FromPromotionSpecProto(s *v1alpha1.PromotionSpec) *kubev1alpha1.PromotionSpec {
	if s == nil {
		return nil
	}
	return &kubev1alpha1.PromotionSpec{
		Stage: s.GetStage(),
		State: s.GetState(),
	}
}

func FromPromotionStatusProto(s *v1alpha1.PromotionStatus) *kubev1alpha1.PromotionStatus {
	if s == nil {
		return nil
	}
	return &kubev1alpha1.PromotionStatus{
		Phase: kubev1alpha1.PromotionPhase(s.GetPhase()),
		Error: s.GetError(),
	}
}

func FromPromotionPolicyProto(p *v1alpha1.PromotionPolicy) *kubev1alpha1.PromotionPolicy {
	if p == nil {
		return nil
	}
	var objectMeta kubemetav1.ObjectMeta
	if p.GetMetadata() != nil {
		objectMeta = *typesmetav1.FromObjectMetaProto(p.GetMetadata())
	}
	return &kubev1alpha1.PromotionPolicy{
		TypeMeta: kubemetav1.TypeMeta{
			APIVersion: kubev1alpha1.GroupVersion.String(),
			Kind:       "PromotionPolicy",
		},
		ObjectMeta:          objectMeta,
		Stage:               p.GetStage(),
		EnableAutoPromotion: p.GetEnableAutoPromotion(),
	}
}

func ToStageProto(e kubev1alpha1.Stage) *v1alpha1.Stage {
	// Status
	availableStates := make([]*v1alpha1.StageState, len(e.Status.AvailableStates))
	for idx := range e.Status.AvailableStates {
		availableStates[idx] = ToStageStateProto(e.Status.AvailableStates[idx])
	}
	var currentState *v1alpha1.StageState
	if e.Status.CurrentState != nil {
		currentState = ToStageStateProto(*e.Status.CurrentState)
	}
	history := make([]*v1alpha1.StageState, len(e.Status.History))
	for idx := range e.Status.History {
		history[idx] = ToStageStateProto(e.Status.History[idx])
	}

	metadata := e.ObjectMeta.DeepCopy()
	metadata.SetManagedFields(nil)

	return &v1alpha1.Stage{
		Metadata: typesmetav1.ToObjectMetaProto(*metadata),
		Spec: &v1alpha1.StageSpec{
			Subscriptions:       ToSubscriptionsProto(*e.Spec.Subscriptions),
			PromotionMechanisms: ToPromotionMechanismsProto(*e.Spec.PromotionMechanisms),
		},
		Status: &v1alpha1.StageStatus{
			AvailableStates: availableStates,
			CurrentState:    currentState,
			History:         history,
			Error:           e.Status.Error,
		},
	}
}

func ToSubscriptionsProto(s kubev1alpha1.Subscriptions) *v1alpha1.Subscriptions {
	var repos *v1alpha1.RepoSubscriptions
	if s.Repos != nil {
		repos = &v1alpha1.RepoSubscriptions{
			Git:    make([]*v1alpha1.GitSubscription, len(s.Repos.Git)),
			Images: make([]*v1alpha1.ImageSubscription, len(s.Repos.Images)),
			Charts: make([]*v1alpha1.ChartSubscription, len(s.Repos.Charts)),
		}
		for idx := range s.Repos.Git {
			repos.Git[idx] = ToGitSubscriptionProto(s.Repos.Git[idx])
		}
		for idx := range s.Repos.Images {
			repos.Images[idx] = ToImageSubscriptionProto(s.Repos.Images[idx])
		}
		for idx := range s.Repos.Charts {
			repos.Charts[idx] = ToChartSubscriptionProto(s.Repos.Charts[idx])
		}
	}

	upstreamStages := make([]*v1alpha1.StageSubscription, len(s.UpstreamStages))
	for idx := range s.UpstreamStages {
		upstreamStages[idx] = ToStageSubscriptionProto(s.UpstreamStages[idx])
	}
	return &v1alpha1.Subscriptions{
		Repos:          repos,
		UpstreamStages: upstreamStages,
	}
}

func ToGitSubscriptionProto(g kubev1alpha1.GitSubscription) *v1alpha1.GitSubscription {
	return &v1alpha1.GitSubscription{
		RepoUrl: g.RepoURL,
		Branch:  g.Branch,
	}
}

func ToImageSubscriptionProto(i kubev1alpha1.ImageSubscription) *v1alpha1.ImageSubscription {
	return &v1alpha1.ImageSubscription{
		RepoUrl:          i.RepoURL,
		UpdateStrategy:   string(i.UpdateStrategy),
		SemverConstraint: proto.String(i.SemverConstraint),
		AllowTags:        proto.String(i.AllowTags),
		IgnoreTags:       i.IgnoreTags,
		Platform:         proto.String(i.Platform),
	}
}

func ToChartSubscriptionProto(c kubev1alpha1.ChartSubscription) *v1alpha1.ChartSubscription {
	return &v1alpha1.ChartSubscription{
		RegistryUrl:      c.RegistryURL,
		Name:             proto.String(c.Name),
		SemverConstraint: proto.String(c.SemverConstraint),
	}
}

func ToStageSubscriptionProto(e kubev1alpha1.StageSubscription) *v1alpha1.StageSubscription {
	return &v1alpha1.StageSubscription{
		Name: e.Name,
	}
}

func ToPromotionMechanismsProto(p kubev1alpha1.PromotionMechanisms) *v1alpha1.PromotionMechanisms {
	gitRepoUpdates := make([]*v1alpha1.GitRepoUpdate, len(p.GitRepoUpdates))
	for idx := range p.GitRepoUpdates {
		gitRepoUpdates[idx] = ToGitRepoUpdateProto(p.GitRepoUpdates[idx])
	}
	argoCDAppUpdates := make([]*v1alpha1.ArgoCDAppUpdate, len(p.ArgoCDAppUpdates))
	for idx := range p.ArgoCDAppUpdates {
		argoCDAppUpdates[idx] = ToArgoCDAppUpdateProto(p.ArgoCDAppUpdates[idx])
	}
	return &v1alpha1.PromotionMechanisms{
		GitRepoUpdates:   gitRepoUpdates,
		ArgocdAppUpdates: argoCDAppUpdates,
	}
}

func ToGitRepoUpdateProto(g kubev1alpha1.GitRepoUpdate) *v1alpha1.GitRepoUpdate {
	var bookkeeper *v1alpha1.BookkeeperPromotionMechanism
	if g.Bookkeeper != nil {
		bookkeeper = ToBookkeeperPromotionMechanismProto(*g.Bookkeeper)
	}
	var kustomize *v1alpha1.KustomizePromotionMechanism
	if g.Kustomize != nil {
		kustomize = ToKustomizePromotionMechanismProto(*g.Kustomize)
	}
	var helm *v1alpha1.HelmPromotionMechanism
	if g.Helm != nil {
		helm = ToHelmPromotionMechanismProto(*g.Helm)
	}
	return &v1alpha1.GitRepoUpdate{
		RepoUrl:     g.RepoURL,
		ReadBranch:  proto.String(g.ReadBranch),
		WriteBranch: g.WriteBranch,
		Bookkeeper:  bookkeeper,
		Kustomize:   kustomize,
		Helm:        helm,
	}
}

func ToBookkeeperPromotionMechanismProto(
	_ kubev1alpha1.BookkeeperPromotionMechanism,
) *v1alpha1.BookkeeperPromotionMechanism {
	return &v1alpha1.BookkeeperPromotionMechanism{}
}

func ToKustomizePromotionMechanismProto(
	k kubev1alpha1.KustomizePromotionMechanism,
) *v1alpha1.KustomizePromotionMechanism {
	images := make([]*v1alpha1.KustomizeImageUpdate, len(k.Images))
	for idx := range k.Images {
		images[idx] = ToKustomizeImageUpdateProto(k.Images[idx])
	}
	return &v1alpha1.KustomizePromotionMechanism{
		Images: images,
	}
}

func ToKustomizeImageUpdateProto(k kubev1alpha1.KustomizeImageUpdate) *v1alpha1.KustomizeImageUpdate {
	return &v1alpha1.KustomizeImageUpdate{
		Image: k.Image,
		Path:  k.Path,
	}
}

func ToHelmPromotionMechanismProto(h kubev1alpha1.HelmPromotionMechanism) *v1alpha1.HelmPromotionMechanism {
	images := make([]*v1alpha1.HelmImageUpdate, len(h.Images))
	for idx := range h.Images {
		images[idx] = ToHelmImageUpdateProto(h.Images[idx])
	}
	charts := make([]*v1alpha1.HelmChartDependencyUpdate, len(h.Charts))
	for idx := range h.Charts {
		charts[idx] = ToHelmChartDependencyUpdateProto(h.Charts[idx])
	}
	return &v1alpha1.HelmPromotionMechanism{
		Images: images,
		Charts: charts,
	}
}

func ToHelmImageUpdateProto(h kubev1alpha1.HelmImageUpdate) *v1alpha1.HelmImageUpdate {
	return &v1alpha1.HelmImageUpdate{
		Image:          h.Image,
		ValuesFilePath: h.ValuesFilePath,
		Key:            h.Key,
		Value:          string(h.Value),
	}
}

func ToHelmChartDependencyUpdateProto(h kubev1alpha1.HelmChartDependencyUpdate) *v1alpha1.HelmChartDependencyUpdate {
	return &v1alpha1.HelmChartDependencyUpdate{
		RegistryUrl: h.RegistryURL,
		Name:        h.Name,
		ChartPath:   h.ChartPath,
	}
}

func ToArgoCDAppUpdateProto(h kubev1alpha1.ArgoCDAppUpdate) *v1alpha1.ArgoCDAppUpdate {
	sourceUpdates := make([]*v1alpha1.ArgoCDSourceUpdate, len(h.SourceUpdates))
	for idx := range h.SourceUpdates {
		sourceUpdates[idx] = ToArgoCDSourceUpdateProto(h.SourceUpdates[idx])
	}
	return &v1alpha1.ArgoCDAppUpdate{
		AppName:       h.AppName,
		AppNamespace:  proto.String(h.AppNamespace),
		SourceUpdates: sourceUpdates,
	}
}

func ToArgoCDSourceUpdateProto(a kubev1alpha1.ArgoCDSourceUpdate) *v1alpha1.ArgoCDSourceUpdate {
	var kustomize *v1alpha1.ArgoCDKustomize
	if a.Kustomize != nil {
		kustomize = ToArgoCDKustomizeProto(*a.Kustomize)
	}
	var helm *v1alpha1.ArgoCDHelm
	if a.Helm != nil {
		helm = ToArgoCDHelmProto(*a.Helm)
	}
	return &v1alpha1.ArgoCDSourceUpdate{
		RepoUrl:              a.RepoURL,
		Chart:                proto.String(a.Chart),
		UpdateTargetRevision: proto.Bool(a.UpdateTargetRevision),
		Kustomize:            kustomize,
		Helm:                 helm,
	}
}

func ToArgoCDKustomizeProto(a kubev1alpha1.ArgoCDKustomize) *v1alpha1.ArgoCDKustomize {
	return &v1alpha1.ArgoCDKustomize{
		Images: a.Images,
	}
}

func ToArgoCDHelmProto(a kubev1alpha1.ArgoCDHelm) *v1alpha1.ArgoCDHelm {
	images := make([]*v1alpha1.ArgoCDHelmImageUpdate, len(a.Images))
	for idx := range images {
		images[idx] = ToArgoCDHelmImageUpdateProto(a.Images[idx])
	}
	return &v1alpha1.ArgoCDHelm{
		Images: images,
	}
}

func ToArgoCDHelmImageUpdateProto(a kubev1alpha1.ArgoCDHelmImageUpdate) *v1alpha1.ArgoCDHelmImageUpdate {
	return &v1alpha1.ArgoCDHelmImageUpdate{
		Image: a.Image,
		Key:   a.Key,
		Value: string(a.Value),
	}
}

func ToStageStateProto(e kubev1alpha1.StageState) *v1alpha1.StageState {
	var firstSeen *timestamppb.Timestamp
	if e.FirstSeen != nil {
		firstSeen = timestamppb.New(e.FirstSeen.Time)
	}
	commits := make([]*v1alpha1.GitCommit, len(e.Commits))
	for idx := range e.Commits {
		commits[idx] = ToGitCommitProto(e.Commits[idx])
	}
	images := make([]*v1alpha1.Image, len(e.Images))
	for idx := range e.Images {
		images[idx] = ToImageProto(e.Images[idx])
	}
	charts := make([]*v1alpha1.Chart, len(e.Charts))
	for idx := range e.Charts {
		charts[idx] = ToChartProto(e.Charts[idx])
	}
	var health *v1alpha1.Health
	if e.Health != nil {
		health = ToHealthProto(*e.Health)
	}
	return &v1alpha1.StageState{
		Id:         e.ID,
		FirstSeen:  firstSeen,
		Provenance: proto.String(e.Provenance),
		Commits:    commits,
		Images:     images,
		Charts:     charts,
		Health:     health,
	}
}

func ToGitCommitProto(g kubev1alpha1.GitCommit) *v1alpha1.GitCommit {
	return &v1alpha1.GitCommit{
		RepoUrl:           g.RepoURL,
		Id:                g.ID,
		Branch:            g.Branch,
		HealthCheckCommit: proto.String(g.HealthCheckCommit),
	}
}

func ToImageProto(i kubev1alpha1.Image) *v1alpha1.Image {
	return &v1alpha1.Image{
		RepoUrl: i.RepoURL,
		Tag:     i.Tag,
	}
}

func ToChartProto(c kubev1alpha1.Chart) *v1alpha1.Chart {
	return &v1alpha1.Chart{
		RegistryUrl: c.RegistryURL,
		Name:        c.Name,
		Version:     c.Version,
	}
}

func ToHealthProto(h kubev1alpha1.Health) *v1alpha1.Health {
	return &v1alpha1.Health{
		Status: proto.String(string(h.Status)),
		Issues: h.Issues,
	}
}

func ToPromotionProto(p kubev1alpha1.Promotion) *v1alpha1.Promotion {
	metadata := p.ObjectMeta.DeepCopy()
	metadata.SetManagedFields(nil)

	return &v1alpha1.Promotion{
		Metadata: typesmetav1.ToObjectMetaProto(*metadata),
		Spec: &v1alpha1.PromotionSpec{
			Stage: p.Spec.Stage,
			State: p.Spec.State,
		},
		Status: &v1alpha1.PromotionStatus{
			Phase: string(p.Status.Phase),
			Error: p.Status.Error,
		},
	}
}

func ToPromotionPolicyProto(p kubev1alpha1.PromotionPolicy) *v1alpha1.PromotionPolicy {
	metadata := p.ObjectMeta.DeepCopy()
	metadata.SetManagedFields(nil)

	return &v1alpha1.PromotionPolicy{
		Metadata:            typesmetav1.ToObjectMetaProto(*metadata),
		Stage:               p.Stage,
		EnableAutoPromotion: p.EnableAutoPromotion,
	}
}

func ToVersionProto(v version.Version) *svcv1alpha1.VersionInfo {
	return &svcv1alpha1.VersionInfo{
		Version:      v.Version,
		GitCommit:    v.GitCommit,
		GitTreeDirty: v.GitTreeDirty,
		BuildTime:    timestamppb.New(v.BuildDate),
		GoVersion:    v.GoVersion,
		Compiler:     v.Compiler,
		Platform:     v.Platform,
	}
}
