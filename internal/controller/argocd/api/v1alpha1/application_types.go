package v1alpha1

import (
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
	Source     *ApplicationSource `json:"source,omitempty"`
	SyncPolicy *SyncPolicy        `json:"syncPolicy,omitempty"`
	Sources    ApplicationSources `json:"sources,omitempty"`
}

type ApplicationSource struct {
	RepoURL        string                      `json:"repoURL"`
	TargetRevision string                      `json:"targetRevision,omitempty"`
	Helm           *ApplicationSourceHelm      `json:"helm,omitempty"`
	Kustomize      *ApplicationSourceKustomize `json:"kustomize,omitempty"`
	Chart          string                      `json:"chart,omitempty"`
}

type ApplicationSources []ApplicationSource

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
	SyncStatusCodeSynced SyncStatusCode = "Synced"
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
}

type SyncOperationResult struct {
	Revision string `json:"revision,omitempty"`
}
