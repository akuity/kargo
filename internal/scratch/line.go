package scratch

// Line represents the logical relationship between environments -- each of
// which is, in turn, represented by an Argo CD Application resource.
//
// TODO: Replace this with a CRD.
type Line struct {
	// A name for this Line. This should be unique, but that's not currently
	// enforced in any way. Currently, if two or more Lines have the same value in
	// their Name field, the LAST Line defined wins.
	Name string `json:"name"`
	// ImageRepositories specifies image repositories that the Line is effectively
	// subscribed to. When a push to any one of these repositories is detected, it
	// will trigger the progressive deployment of the new image through the Line's
	// environments.
	ImageRepositories []string `json:"imageRepositories"`
	// Namespace is the Kubernetes namespace in which all the Argo CD Application
	// resources named by the Environment field can/should be found.
	Namespace string `json:"namespace"`
	// Environments enumerates a logical ordering of environments, each
	// represented by a string reference to an (assumed to be) existing Argo CD
	// Application resource in the Kubernetes namespace indicated by the Namespace
	// field.
	Environments []string `json:"environments"`
	// TODO: Need to allow a Line to specify some sort of promotion strategy for
	// moving changes through the environments. These strategies can be built with
	// deep knowledge of prominent GitOps patterns and tools. This probably needs
	// to be more than a string. For instance, for some strategies, we may need
	// information that cannot be obtained seamlessly from related Argo CD
	// configuration.
	// PromotionStrategy string `json:"promotionStrategy"`
}
