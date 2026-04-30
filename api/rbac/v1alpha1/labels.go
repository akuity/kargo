package v1alpha1

const (
	// LabelKeyAPIToken can be used to mark a Kubernetes Secret as a
	// token for the Kargo API.
	LabelKeyAPIToken = "rbac.kargo.akuity.io/api-token" // nolint: gosec
	// LabelKeySystemRole can be used to mark a ServiceAccount in Kargo's own
	// namespace as a "system role".
	LabelKeySystemRole = "rbac.kargo.akuity.io/system-role"

	// LabelValueTrue is used to identify a label that has a value of "true".
	LabelValueTrue = "true"
)
