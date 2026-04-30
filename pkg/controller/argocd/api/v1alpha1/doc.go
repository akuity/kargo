package v1alpha1

// This package reproduces just enough of
// github.com/pkg/apis/application/v1alpha1 to support Kargo without having
// to incur undesired dependencies on Argo CD and GitOps Engine -- both of
// which have transitive dependencies on Kubernetes and can sometimes hold us
// back from upgrading important Kubernetes packages.
