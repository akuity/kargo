package v1alpha1

// PromotionWindowKind indicates whether a PromotionWindow permits or forbids
// promotions while it is active.
//
// +kubebuilder:validation:Enum=Allow;Deny
type PromotionWindowKind string

const (
	// PromotionWindowKindAllow indicates that a PromotionWindow permits
	// promotions while it is active. When any Allow window matches a Stage, that
	// Stage's schedule is open only while at least one such window is active.
	PromotionWindowKindAllow PromotionWindowKind = "Allow"
	// PromotionWindowKindDeny indicates that a PromotionWindow forbids promotions
	// while it is active. An active Deny window always closes the schedule,
	// regardless of any Allow windows.
	PromotionWindowKindDeny PromotionWindowKind = "Deny"
)

// PromotionWindow describes a recurring or one-shot time window that gates
// promotions for the Stages it matches. Windows may appear on both
// ProjectConfig and ClusterConfig; a Stage's effective schedule is the union of
// all matching windows from both. A schedule is open at a given time when no
// matching Deny window is active and, if any Allow windows match the Stage, at
// least one of them is active. Windows gate all promotions uniformly (auto,
// manual, and rollback).
//
// Kargo Enterprise only: This type is ignored in Kargo OSS. The schedule is
// evaluated and enforced only by Kargo Enterprise; OSS carries the API for
// compatibility, as it does for other Enterprise-only fields.
type PromotionWindow struct {
	// Name is a symbolic name for the window, unique within its list. It is used
	// to identify the window in denial messages and events.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Name string `json:"name"`
	// StageSelector selects the Stages this window applies to. When omitted, the
	// window applies to all Stages in scope (project-wide on ProjectConfig,
	// cluster-wide on ClusterConfig). It reuses PromotionPolicySelector, so it
	// can match by exact name, glob/regex pattern, or label selector.
	//
	// +optional
	StageSelector *PromotionPolicySelector `json:"stageSelector,omitempty"`
	// ProjectSelector selects the Projects this window applies to. It is only
	// meaningful on ClusterConfig windows; on ProjectConfig windows the Project
	// is implicit and this field is rejected. When omitted on a ClusterConfig
	// window, the window applies to all Projects. It reuses
	// PromotionPolicySelector, matching Projects by exact name, glob/regex
	// pattern, or label selector.
	//
	// +optional
	ProjectSelector *PromotionPolicySelector `json:"projectSelector,omitempty"`
	// Kind indicates whether this window allows or denies promotions while it is
	// active.
	//
	// +kubebuilder:validation:Required
	Kind PromotionWindowKind `json:"kind"`
	// RRule is an optional RFC 5545 recurrence rule (e.g. "FREQ=DAILY") that
	// makes the window recurring. When omitted, the window is a one-shot interval
	// defined by DTStart and DTEnd. The full value is parsed and validated by
	// Kargo Enterprise.
	//
	// +optional
	RRule string `json:"rrule,omitempty"`
	// DTStart is the window's start as an iCal date-time, with an optional
	// "TZID=" prefix carrying the time zone (e.g.
	// "TZID=America/New_York:20260101T090000"). The value is parsed and validated
	// by Kargo Enterprise.
	//
	// +optional
	DTStart string `json:"dtstart,omitempty"`
	// DTEnd is the window's end in the same format as DTStart. When combined with
	// RRule, DTEnd - DTStart defines the duration of each occurrence. The value
	// is parsed and validated by Kargo Enterprise.
	//
	// +optional
	DTEnd string `json:"dtend,omitempty"`
}
