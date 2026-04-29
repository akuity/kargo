// Package heartbeattest provides helpers for tests that need to seed the
// heartbeat package's underlying storage. Production code should not import
// this package; callers should consume heartbeats via heartbeat.List.
package heartbeattest

import (
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Option configures a heartbeat record produced by NewHeartbeatLease.
type Option func(*coordinationv1.Lease)

// WithRenewedAt sets the lease's renewTime, the moment of the most recent
// heartbeat. Omit it to seed a record that has never been renewed.
func WithRenewedAt(t time.Time) Option {
	return func(l *coordinationv1.Lease) {
		l.Spec.RenewTime = &metav1.MicroTime{Time: t}
	}
}

// WithDuration sets the lease's leaseDurationSeconds, the validity window
// applied to the renewTime by readers. Omit it to seed a record that
// readers should treat as dead regardless of renewTime.
func WithDuration(d time.Duration) Option {
	return func(l *coordinationv1.Lease) {
		// #nosec G115 -- test-only helper; durations are caller-controlled
		l.Spec.LeaseDurationSeconds = ptr.To(int32(d.Seconds()))
	}
}

// NewHeartbeatLease returns an object that the heartbeat package's reader
// will treat as the heartbeat record for the named controller. Tests use
// this to seed a fake client; they should never construct heartbeat-record
// objects directly.
//
// Pass an empty controllerName to seed the unnamed-controller heartbeat
// (label value is the empty string, matching what the production renewer
// writes for an unnamed controller).
func NewHeartbeatLease(
	namespace, controllerName string,
	opts ...Option,
) client.Object {
	nameSuffix := controllerName
	if nameSuffix == "" {
		nameSuffix = "unnamed"
	}
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kargo-controller-" + nameSuffix,
			Labels:    map[string]string{kargoapi.LabelKeyController: controllerName},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity: ptr.To("test-holder"),
		},
	}
	for _, opt := range opts {
		opt(lease)
	}
	return lease
}
