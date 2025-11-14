package v1alpha1

// This package reproduces just enough of
// github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1 to support Kargo
// without having to incur undesired dependencies on Argo Rollouts, Argo CD,
// GitOps Engine, etc., since these have transitive dependencies on Kubernetes
// and can sometimes hold us back from upgrading important Kubernetes packages.

// TODO: KR: Once Analysis is fully-integrated into Kargo, many of the fields
// that we don't use can be removed from types in this package.
