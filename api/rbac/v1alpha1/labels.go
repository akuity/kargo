package v1alpha1

const (
	// LabelKeyServiceAccount can be used to mark a Kubernetes ServiceAccount as a
	// also being a Kargo ServiceAccount.
	LabelKeyServiceAccount = "rbac.kargo.akuity.io/service-account"
	// LabelKeyServiceAccountToken can be used to mark a Kubernetes Secret as a
	// token for a Kargo ServiceAccount.
	LabelKeyServiceAccountToken = "rbac.kargo.akuity.io/service-account-token" // nolint: gosec

	// LabelValueTrue is used to identify a label that has a value of "true".
	LabelValueTrue = "true"
)
