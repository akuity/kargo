package webhook

import (
	"regexp"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// kubernetesSystemControllerUsernames is the set of well-known Kubernetes
// system identities that perform garbage collection and namespace teardown.
// These must be allowed to delete replicated secrets when a namespace is
// removed so that namespaces do not become stuck in Terminating.
var kubernetesSystemControllerUsernames = map[string]struct{}{
	"system:serviceaccount:kube-system:namespace-controller":      {},
	"system:serviceaccount:kube-system:generic-garbage-collector": {},
	"system:kube-controller-manager":                              {},
}

type IsRequestFromKargoControlplaneFn func(admission.Request) bool

func IsRequestFromKargoControlplane(regex *regexp.Regexp) IsRequestFromKargoControlplaneFn {
	return func(req admission.Request) bool {
		// Always return false if regex is not provided
		if regex == nil {
			return false
		}
		return regex.Match([]byte(req.UserInfo.Username))
	}
}

// IsRequestFromKubernetesSystemController returns true when the admission
// request originates from one of the well-known Kubernetes system controllers
// responsible for garbage collection and namespace teardown.
func IsRequestFromKubernetesSystemController(req admission.Request) bool {
	_, ok := kubernetesSystemControllerUsernames[req.UserInfo.Username]
	return ok
}

// IsRequestFromClusterAdmin returns true when the admission request originates
// from a member of the system:masters group, which maps to cluster-admin
// privileges and is used for break-glass operations.
func IsRequestFromClusterAdmin(req admission.Request) bool {
	return slices.Contains(req.UserInfo.Groups, "system:masters")
}
