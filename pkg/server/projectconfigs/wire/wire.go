// Package wire holds the request and response bodies of the ProjectConfig
// endpoints that are not ProjectConfigs themselves. It imports nothing of
// Kargo's but its API types, so that the API server and its clients can
// share these bodies without importing each other.
package wire

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// RefreshResponse is the response body of POST .../config/refresh.
type RefreshResponse struct {
	// Token identifies the refresh. The ProjectConfig's
	// status.lastHandledRefresh reports it once the refresh has been handled.
	Token string `json:"token"`
} // @name RefreshProjectConfigResponse

// StatusRequest is the request body of PUT .../config/status: the outcome of
// a reconcile of a ProjectConfig.
type StatusRequest struct {
	// ObservedUID is the metadata.uid of the ProjectConfig the reconcile
	// read. A ProjectConfig deleted and recreated since does not take the
	// status.
	ObservedUID string `json:"observedUID"`
	// ObservedGeneration is the metadata.generation of the ProjectConfig the
	// reconcile read.
	ObservedGeneration int64 `json:"observedGeneration"`
	// ObservedRefreshToken is the refresh token the reconcile read, if any.
	ObservedRefreshToken string                            `json:"observedRefreshToken,omitempty"`
	Conditions           []metav1.Condition                `json:"conditions,omitempty"`
	WebhookReceivers     []kargoapi.WebhookReceiverDetails `json:"webhookReceivers,omitempty"`
} // @name UpdateProjectConfigStatusRequest

// StatusResponse is the response body of PUT .../config/status.
type StatusResponse struct {
	// SkippedReceivers names the webhook receivers that were not recorded
	// because another ProjectConfig owns their paths, or an earlier receiver
	// has their names.
	SkippedReceivers []string `json:"skippedReceivers,omitempty"`
} // @name UpdateProjectConfigStatusResponse
