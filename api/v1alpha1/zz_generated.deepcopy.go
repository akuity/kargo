//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDAppUpdate) DeepCopyInto(out *ArgoCDAppUpdate) {
	*out = *in
	if in.SourceUpdates != nil {
		in, out := &in.SourceUpdates, &out.SourceUpdates
		*out = make([]ArgoCDSourceUpdate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDAppUpdate.
func (in *ArgoCDAppUpdate) DeepCopy() *ArgoCDAppUpdate {
	if in == nil {
		return nil
	}
	out := new(ArgoCDAppUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDHelm) DeepCopyInto(out *ArgoCDHelm) {
	*out = *in
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]ArgoCDHelmImageUpdate, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDHelm.
func (in *ArgoCDHelm) DeepCopy() *ArgoCDHelm {
	if in == nil {
		return nil
	}
	out := new(ArgoCDHelm)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDHelmImageUpdate) DeepCopyInto(out *ArgoCDHelmImageUpdate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDHelmImageUpdate.
func (in *ArgoCDHelmImageUpdate) DeepCopy() *ArgoCDHelmImageUpdate {
	if in == nil {
		return nil
	}
	out := new(ArgoCDHelmImageUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDKustomize) DeepCopyInto(out *ArgoCDKustomize) {
	*out = *in
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDKustomize.
func (in *ArgoCDKustomize) DeepCopy() *ArgoCDKustomize {
	if in == nil {
		return nil
	}
	out := new(ArgoCDKustomize)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArgoCDSourceUpdate) DeepCopyInto(out *ArgoCDSourceUpdate) {
	*out = *in
	if in.Kustomize != nil {
		in, out := &in.Kustomize, &out.Kustomize
		*out = new(ArgoCDKustomize)
		(*in).DeepCopyInto(*out)
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(ArgoCDHelm)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArgoCDSourceUpdate.
func (in *ArgoCDSourceUpdate) DeepCopy() *ArgoCDSourceUpdate {
	if in == nil {
		return nil
	}
	out := new(ArgoCDSourceUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BookkeeperPromotionMechanism) DeepCopyInto(out *BookkeeperPromotionMechanism) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BookkeeperPromotionMechanism.
func (in *BookkeeperPromotionMechanism) DeepCopy() *BookkeeperPromotionMechanism {
	if in == nil {
		return nil
	}
	out := new(BookkeeperPromotionMechanism)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Chart) DeepCopyInto(out *Chart) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Chart.
func (in *Chart) DeepCopy() *Chart {
	if in == nil {
		return nil
	}
	out := new(Chart)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartSubscription) DeepCopyInto(out *ChartSubscription) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartSubscription.
func (in *ChartSubscription) DeepCopy() *ChartSubscription {
	if in == nil {
		return nil
	}
	out := new(ChartSubscription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Freight) DeepCopyInto(out *Freight) {
	*out = *in
	if in.FirstSeen != nil {
		in, out := &in.FirstSeen, &out.FirstSeen
		*out = (*in).DeepCopy()
	}
	if in.Commits != nil {
		in, out := &in.Commits, &out.Commits
		*out = make([]GitCommit, len(*in))
		copy(*out, *in)
	}
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]Image, len(*in))
		copy(*out, *in)
	}
	if in.Charts != nil {
		in, out := &in.Charts, &out.Charts
		*out = make([]Chart, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Freight.
func (in *Freight) DeepCopy() *Freight {
	if in == nil {
		return nil
	}
	out := new(Freight)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in FreightStack) DeepCopyInto(out *FreightStack) {
	{
		in := &in
		*out = make(FreightStack, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FreightStack.
func (in FreightStack) DeepCopy() FreightStack {
	if in == nil {
		return nil
	}
	out := new(FreightStack)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitCommit) DeepCopyInto(out *GitCommit) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitCommit.
func (in *GitCommit) DeepCopy() *GitCommit {
	if in == nil {
		return nil
	}
	out := new(GitCommit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitRepoUpdate) DeepCopyInto(out *GitRepoUpdate) {
	*out = *in
	if in.Bookkeeper != nil {
		in, out := &in.Bookkeeper, &out.Bookkeeper
		*out = new(BookkeeperPromotionMechanism)
		**out = **in
	}
	if in.Kustomize != nil {
		in, out := &in.Kustomize, &out.Kustomize
		*out = new(KustomizePromotionMechanism)
		(*in).DeepCopyInto(*out)
	}
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(HelmPromotionMechanism)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitRepoUpdate.
func (in *GitRepoUpdate) DeepCopy() *GitRepoUpdate {
	if in == nil {
		return nil
	}
	out := new(GitRepoUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitSubscription) DeepCopyInto(out *GitSubscription) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitSubscription.
func (in *GitSubscription) DeepCopy() *GitSubscription {
	if in == nil {
		return nil
	}
	out := new(GitSubscription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Health) DeepCopyInto(out *Health) {
	*out = *in
	if in.Issues != nil {
		in, out := &in.Issues, &out.Issues
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Health.
func (in *Health) DeepCopy() *Health {
	if in == nil {
		return nil
	}
	out := new(Health)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChartDependencyUpdate) DeepCopyInto(out *HelmChartDependencyUpdate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChartDependencyUpdate.
func (in *HelmChartDependencyUpdate) DeepCopy() *HelmChartDependencyUpdate {
	if in == nil {
		return nil
	}
	out := new(HelmChartDependencyUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmImageUpdate) DeepCopyInto(out *HelmImageUpdate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmImageUpdate.
func (in *HelmImageUpdate) DeepCopy() *HelmImageUpdate {
	if in == nil {
		return nil
	}
	out := new(HelmImageUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmPromotionMechanism) DeepCopyInto(out *HelmPromotionMechanism) {
	*out = *in
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]HelmImageUpdate, len(*in))
		copy(*out, *in)
	}
	if in.Charts != nil {
		in, out := &in.Charts, &out.Charts
		*out = make([]HelmChartDependencyUpdate, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmPromotionMechanism.
func (in *HelmPromotionMechanism) DeepCopy() *HelmPromotionMechanism {
	if in == nil {
		return nil
	}
	out := new(HelmPromotionMechanism)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Image) DeepCopyInto(out *Image) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Image.
func (in *Image) DeepCopy() *Image {
	if in == nil {
		return nil
	}
	out := new(Image)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSubscription) DeepCopyInto(out *ImageSubscription) {
	*out = *in
	if in.IgnoreTags != nil {
		in, out := &in.IgnoreTags, &out.IgnoreTags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSubscription.
func (in *ImageSubscription) DeepCopy() *ImageSubscription {
	if in == nil {
		return nil
	}
	out := new(ImageSubscription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KustomizeImageUpdate) DeepCopyInto(out *KustomizeImageUpdate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KustomizeImageUpdate.
func (in *KustomizeImageUpdate) DeepCopy() *KustomizeImageUpdate {
	if in == nil {
		return nil
	}
	out := new(KustomizeImageUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KustomizePromotionMechanism) DeepCopyInto(out *KustomizePromotionMechanism) {
	*out = *in
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]KustomizeImageUpdate, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KustomizePromotionMechanism.
func (in *KustomizePromotionMechanism) DeepCopy() *KustomizePromotionMechanism {
	if in == nil {
		return nil
	}
	out := new(KustomizePromotionMechanism)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Promotion) DeepCopyInto(out *Promotion) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Spec != nil {
		in, out := &in.Spec, &out.Spec
		*out = new(PromotionSpec)
		**out = **in
	}
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Promotion.
func (in *Promotion) DeepCopy() *Promotion {
	if in == nil {
		return nil
	}
	out := new(Promotion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Promotion) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionInfo) DeepCopyInto(out *PromotionInfo) {
	*out = *in
	in.Freight.DeepCopyInto(&out.Freight)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionInfo.
func (in *PromotionInfo) DeepCopy() *PromotionInfo {
	if in == nil {
		return nil
	}
	out := new(PromotionInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionList) DeepCopyInto(out *PromotionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Promotion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionList.
func (in *PromotionList) DeepCopy() *PromotionList {
	if in == nil {
		return nil
	}
	out := new(PromotionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PromotionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionMechanisms) DeepCopyInto(out *PromotionMechanisms) {
	*out = *in
	if in.GitRepoUpdates != nil {
		in, out := &in.GitRepoUpdates, &out.GitRepoUpdates
		*out = make([]GitRepoUpdate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ArgoCDAppUpdates != nil {
		in, out := &in.ArgoCDAppUpdates, &out.ArgoCDAppUpdates
		*out = make([]ArgoCDAppUpdate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionMechanisms.
func (in *PromotionMechanisms) DeepCopy() *PromotionMechanisms {
	if in == nil {
		return nil
	}
	out := new(PromotionMechanisms)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionPolicy) DeepCopyInto(out *PromotionPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionPolicy.
func (in *PromotionPolicy) DeepCopy() *PromotionPolicy {
	if in == nil {
		return nil
	}
	out := new(PromotionPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PromotionPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionPolicyList) DeepCopyInto(out *PromotionPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PromotionPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionPolicyList.
func (in *PromotionPolicyList) DeepCopy() *PromotionPolicyList {
	if in == nil {
		return nil
	}
	out := new(PromotionPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PromotionPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionSpec) DeepCopyInto(out *PromotionSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionSpec.
func (in *PromotionSpec) DeepCopy() *PromotionSpec {
	if in == nil {
		return nil
	}
	out := new(PromotionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PromotionStatus) DeepCopyInto(out *PromotionStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PromotionStatus.
func (in *PromotionStatus) DeepCopy() *PromotionStatus {
	if in == nil {
		return nil
	}
	out := new(PromotionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoSubscriptions) DeepCopyInto(out *RepoSubscriptions) {
	*out = *in
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = make([]GitSubscription, len(*in))
		copy(*out, *in)
	}
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]ImageSubscription, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Charts != nil {
		in, out := &in.Charts, &out.Charts
		*out = make([]ChartSubscription, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoSubscriptions.
func (in *RepoSubscriptions) DeepCopy() *RepoSubscriptions {
	if in == nil {
		return nil
	}
	out := new(RepoSubscriptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Stage) DeepCopyInto(out *Stage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Spec != nil {
		in, out := &in.Spec, &out.Spec
		*out = new(StageSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Stage.
func (in *Stage) DeepCopy() *Stage {
	if in == nil {
		return nil
	}
	out := new(Stage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Stage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StageList) DeepCopyInto(out *StageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Stage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StageList.
func (in *StageList) DeepCopy() *StageList {
	if in == nil {
		return nil
	}
	out := new(StageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StageSpec) DeepCopyInto(out *StageSpec) {
	*out = *in
	if in.Subscriptions != nil {
		in, out := &in.Subscriptions, &out.Subscriptions
		*out = new(Subscriptions)
		(*in).DeepCopyInto(*out)
	}
	if in.PromotionMechanisms != nil {
		in, out := &in.PromotionMechanisms, &out.PromotionMechanisms
		*out = new(PromotionMechanisms)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StageSpec.
func (in *StageSpec) DeepCopy() *StageSpec {
	if in == nil {
		return nil
	}
	out := new(StageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StageStatus) DeepCopyInto(out *StageStatus) {
	*out = *in
	if in.AvailableFreight != nil {
		in, out := &in.AvailableFreight, &out.AvailableFreight
		*out = make(FreightStack, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CurrentFreight != nil {
		in, out := &in.CurrentFreight, &out.CurrentFreight
		*out = new(Freight)
		(*in).DeepCopyInto(*out)
	}
	if in.History != nil {
		in, out := &in.History, &out.History
		*out = make(FreightStack, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Health != nil {
		in, out := &in.Health, &out.Health
		*out = new(Health)
		(*in).DeepCopyInto(*out)
	}
	if in.CurrentPromotion != nil {
		in, out := &in.CurrentPromotion, &out.CurrentPromotion
		*out = new(PromotionInfo)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StageStatus.
func (in *StageStatus) DeepCopy() *StageStatus {
	if in == nil {
		return nil
	}
	out := new(StageStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StageSubscription) DeepCopyInto(out *StageSubscription) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StageSubscription.
func (in *StageSubscription) DeepCopy() *StageSubscription {
	if in == nil {
		return nil
	}
	out := new(StageSubscription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Subscriptions) DeepCopyInto(out *Subscriptions) {
	*out = *in
	if in.Repos != nil {
		in, out := &in.Repos, &out.Repos
		*out = new(RepoSubscriptions)
		(*in).DeepCopyInto(*out)
	}
	if in.UpstreamStages != nil {
		in, out := &in.UpstreamStages, &out.UpstreamStages
		*out = make([]StageSubscription, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Subscriptions.
func (in *Subscriptions) DeepCopy() *Subscriptions {
	if in == nil {
		return nil
	}
	out := new(Subscriptions)
	in.DeepCopyInto(out)
	return out
}
