package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Target represents a single destination -- a cluster, for instance -- that
// Stages promote Freight to. A Target is descriptive: it holds target-specific
// values consumed by the promotion steps of Stages that govern it. It defines
// no promotion steps and no Freight sources of its own and therefore cannot
// effect any promotion itself. Its status records, per governing Stage, what
// Freight was last promoted to it and how that Freight is faring there.
type Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes the Target.
	Spec TargetSpec `json:"spec,omitempty"`
	// Status describes the current status of the Target.
	Status TargetStatus `json:"status,omitempty"`
}

func (t *Target) GetStatus() *TargetStatus {
	return &t.Status
}

// TargetSpec describes a Target.
type TargetSpec struct {
	// Params is a map of arbitrary, target-specific values. Values may be any
	// valid JSON -- including nested objects and arrays -- so promotion steps
	// can reference deeply nested data. Promotion steps of Stages that govern
	// this Target may reference these values by key in their expressions (for
	// example, target.params.branch or target.params.cluster.region).
	//
	// +optional
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty"`
}

// TargetStatus describes the current status of a Target.
type TargetStatus struct {
	// Stages records the Target's state with respect to each Stage that
	// governs it, keyed by Stage name. A Target may be governed by several
	// Stages at once, each promoting its own Freight to it, so everything
	// observed about the Target is scoped to the Stage that observed it.
	// This mirrors how Freight status keys currentlyIn, verifiedIn, and
	// approvedFor by Stage name.
	//
	// An entry exists only once a Promotion from that Stage to this Target
	// has succeeded. A Stage that governs the Target but has never promoted
	// to it has no entry.
	//
	// +optional
	Stages map[string]TargetStageStatus `json:"stages,omitempty"`
}

// TargetStageStatus describes a Target's state with respect to a single Stage
// that governs it: the Freight that Stage last successfully promoted to the
// Target, the health checks that Promotion left behind, and the Target's
// current health as assessed from them. It is the per-Target counterpart of
// the current Freight, health, and verification state a Stage keeps for
// itself.
type TargetStageStatus struct {
	// CurrentFreight is the FreightCollection that the Stage's most recent
	// successful Promotion to this Target rendered. Because a Stage may
	// request Freight from several origins and a single Promotion changes
	// only one of them, this is a collection rather than a single Freight
	// reference: it holds one entry per origin, exactly as the Stage's own
	// current collection does, so the Target's full desired state is known
	// even for origins the latest Promotion did not touch.
	//
	// The collection is copied from the Promotion's status rather than
	// derived from this Target's previous state, so a Target that missed a
	// round or was newly discovered converges to the Stage's desired state
	// on its next successful Promotion. Its ID is deterministic in its
	// contents, so a Target that received the same round as the Stage carries
	// the same ID as the Stage's current collection; comparing the two tells
	// whether the Target is up to date. Its verification history records
	// verifications of this Freight on this Target specifically.
	//
	// +optional
	CurrentFreight *FreightCollection `json:"currentFreight,omitempty"`
	// Health is the Target's health with respect to this Stage's Freight, as
	// last assessed by executing HealthChecks. It is absent until the first
	// assessment. Health is an ongoing observation rather than a Promotion
	// outcome: it is reassessed periodically and whenever the systems the
	// health checks observe change, so it may become Unhealthy long after the
	// Promotion that produced CurrentFreight succeeded.
	//
	// A target-aware Stage does not assess health itself. Its own health is
	// an aggregate of this field across every Target it governs: Healthy only
	// when each is Healthy and on the Stage's current Freight, Unhealthy when
	// any is not, Unknown while any has yet to be assessed.
	//
	// +optional
	Health *Health `json:"health,omitempty"`
	// HealthChecks are the health check directives produced by the steps of
	// the Stage's most recent successful Promotion to this Target -- for
	// instance, which Argo CD Applications that Promotion updated and to what
	// revisions. They are the input from which Health is assessed, and they
	// are recorded here because Health must be reassessed for as long as
	// CurrentFreight remains on the Target, while the Promotion that produced
	// them is eventually garbage collected. A Stage keeps the same
	// information for itself in its lastPromotion.
	//
	// +optional
	HealthChecks []HealthCheckStep `json:"healthChecks,omitempty"`
}

// SetStatusForStage records the Target's status with respect to the specified
// Stage, replacing any existing record for that Stage.
func (s *TargetStatus) SetStatusForStage(stage string, status TargetStageStatus) {
	if s.Stages == nil {
		s.Stages = make(map[string]TargetStageStatus)
	}
	s.Stages[stage] = status
}

// RemoveStatusForStage removes the Target's status with respect to the
// specified Stage, if any. It is a no-op when no such record exists.
func (s *TargetStatus) RemoveStatusForStage(stage string) {
	delete(s.Stages, stage)
}

// +kubebuilder:object:root=true

// TargetList is a list of Target resources.
type TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Target `json:"items"`
}
