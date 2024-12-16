package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ApplicationSpec   `json:"spec"`
	Status            ApplicationStatus `json:"status,"`
	Operation         *Operation        `json:"operation,omitempty"`
}

type ApplicationSpec struct {
	Source      *ApplicationSource     `json:"source,omitempty"`
	Destination ApplicationDestination `json:"destination"`
	Project     string                 `json:"project"`
	SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
	Sources     ApplicationSources     `json:"sources,omitempty"`
}

type ApplicationSource struct {
	RepoURL        string                      `json:"repoURL"`
	TargetRevision string                      `json:"targetRevision,omitempty"`
	Helm           *ApplicationSourceHelm      `json:"helm,omitempty"`
	Kustomize      *ApplicationSourceKustomize `json:"kustomize,omitempty"`
	Chart          string                      `json:"chart,omitempty"`
}

// Equals compares two instances of ApplicationSource and returns true if
// they are equal.
func (source *ApplicationSource) Equals(other *ApplicationSource) bool {
	if source == nil && other == nil {
		return true
	}
	if source == nil || other == nil {
		return false
	}
	return reflect.DeepEqual(source.DeepCopy(), other.DeepCopy())
}

type ApplicationSources []ApplicationSource

// Equals compares two instances of ApplicationSources and returns true if
// they are equal.
func (s ApplicationSources) Equals(other ApplicationSources) bool {
	if len(s) != len(other) {
		return false
	}
	for i := range s {
		if !s[i].Equals(&other[i]) {
			return false
		}
	}
	return true
}

type RefreshType string

const (
	RefreshTypeHard RefreshType = "hard"
)

type ApplicationSourceHelm struct {
	Parameters []HelmParameter `json:"parameters,omitempty"`
}

type HelmParameter struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type KustomizeImage string

type KustomizeImages []KustomizeImage

type ApplicationSourceKustomize struct {
	Images KustomizeImages `json:"images,omitempty"`
}

type ApplicationDestination struct {
	Server    string `json:"server,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type ApplicationStatus struct {
	Health         HealthStatus           `json:"health,omitempty"`
	Sync           SyncStatus             `json:"sync,omitempty"`
	Conditions     []ApplicationCondition `json:"conditions,omitempty"`
	OperationState *OperationState        `json:"operationState,omitempty"`
}

type OperationInitiator struct {
	Username  string `json:"username,omitempty"`
	Automated bool   `json:"automated,omitempty"`
}

type Operation struct {
	Sync        *SyncOperation     `json:"sync,omitempty"`
	InitiatedBy OperationInitiator `json:"initiatedBy,omitempty"`
	Info        []*Info            `json:"info,omitempty"`
	Retry       RetryStrategy      `json:"retry,omitempty"`
}

type SyncOperation struct {
	SyncOptions SyncOptions `json:"syncOptions,omitempty"`
	Revisions   []string    `json:"revisions,omitempty"`
}

type Info struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SyncOptions []string

type SyncPolicy struct {
	SyncOptions SyncOptions    `json:"syncOptions,omitempty"`
	Retry       *RetryStrategy `json:"retry,omitempty"`
}

type RetryStrategy struct {
	Limit   int64    `json:"limit,omitempty"`
	Backoff *Backoff `json:"backoff,omitempty"`
}

type Backoff struct {
	Duration    string `json:"duration,omitempty"`
	Factor      *int64 `json:"factor,omitempty"`
	MaxDuration string `json:"maxDuration,omitempty"`
}

//+kubebuilder:object:root=true

type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Application `json:"items"`
}

type SyncStatusCode string

const (
	SyncStatusCodeSynced    SyncStatusCode = "Synced"
	SyncStatusCodeOutOfSync SyncStatusCode = "OutOfSync"
	SyncStatusCodeUnknown   SyncStatusCode = "Unknown"
)

type SyncStatus struct {
	Status    SyncStatusCode `json:"status"`
	Revision  string         `json:"revision,omitempty"`
	Revisions []string       `json:"revisions,omitempty"`
}

type HealthStatus struct {
	Status  HealthStatusCode `json:"status,omitempty"`
	Message string           `json:"message,omitempty"`
}

type ApplicationConditionType string

var (
	ApplicationConditionInvalidSpecError ApplicationConditionType = "InvalidSpecError"
	ApplicationConditionComparisonError  ApplicationConditionType = "ComparisonError"
)

type ApplicationCondition struct {
	Type               ApplicationConditionType `json:"type"`
	Message            string                   `json:"message"`
	LastTransitionTime *metav1.Time             `json:"lastTransitionTime,omitempty"`
}

type OperationState struct {
	Operation  Operation            `json:"operation,omitempty"`
	Phase      OperationPhase       `json:"phase,omitempty"`
	Message    string               `json:"message,omitempty"`
	SyncResult *SyncOperationResult `json:"syncResult,omitempty"`
	FinishedAt *metav1.Time         `json:"finishedAt,omitempty"`
}

type SyncOperationResult struct {
	Revision  string             `json:"revision,omitempty"`
	Revisions []string           `json:"revisions,omitempty"`
	Source    ApplicationSource  `json:"source,omitempty"`
	Sources   ApplicationSources `json:"sources,omitempty"`
}
