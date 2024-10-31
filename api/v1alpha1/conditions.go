package v1alpha1

// For more information on the use of conditions in Kubernetes, see:
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
// nolint:lll

const (
	// ConditionTypeReady denotes that the resource is ready, i.e. it is
	// fully operational.
	//
	// The exact meaning of "ready" is specific to the resource type. For
	// example, a Warehouse resource may consider itself ready if it can
	// successfully connect to its upstream sources, has discovered
	// artifacts, and is not currently in the process of reconciling.
	//
	// This is an "oscillating", and "normal-true" or "positive polarity"
	// condition, meaning that the presence of the condition with a status
	// of "True" indicates that the resource is ready, and the status of
	// the condition may change over time as the resource transitions between
	// ready and not ready states.
	ConditionTypeReady = "Ready"

	// ConditionTypeReconciling denotes that the resource is currently being
	// reconciled.
	//
	// This condition is used to indicate that the controller is actively
	// working on the resource, and further changes to the resource can be
	// expected. The condition is removed when the controller has finished
	// reconciling the resource and no further changes are expected. Which
	// MAY take multiple reconciliation attempts.
	//
	// This is a "normal-false" or "negative polarity" condition, meaning
	// that the presence of the condition with a status of "True" indicates
	// that the resource is being reconciled, and the absence of the condition
	// or a status of "False" indicates that the resource is not being
	// reconciled.
	ConditionTypeReconciling = "Reconciling"

	// ConditionTypeStalled denotes that the reconciliation of the resource
	// has stalled.
	//
	// This condition is used to indicate that the controller has stopped
	// making progress on the resource, and further changes to the resource
	// are not expected until the reason for the stall is resolved, which
	// MAY require manual intervention. The condition is removed when the
	// controller has resumed making progress on the resource.
	//
	// This is a "normal-false" or "negative polarity" condition, meaning
	// that the presence of the condition with a status of "True" indicates
	// that the resource has stalled, and the absence of the condition or
	// a status of "False" indicates that the resource is operating as
	// expected.
	ConditionTypeStalled = "Stalled"

	// ConditionTypeHealthy denotes that the resource is healthy.
	//
	// The meaning of "healthy" is specific to the resource type. For example,
	// a Warehouse resource may consider itself healthy if it can successfully
	// connect to its upstream sources, while a Stage resource may consider
	// itself healthy if the underlying applications are running as expected.
	//
	// This is a "normal-true" or "positive polarity" condition, meaning that
	// the presence of the condition with a status of "True" indicates that
	// the resource is operating as expected.
	ConditionTypeHealthy = "Healthy"
)
