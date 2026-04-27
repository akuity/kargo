// Package heartbeat manages per-controller liveness reporting for Kargo
// controllers. Each controller writes a coordination.k8s.io/v1.Lease to the
// Kargo namespace and renews it periodically; the API server lists those
// leases to derive controller liveness for the UI.
package heartbeat

import (
	"context"
	"fmt"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Status is point-in-time liveness synthesized from a single controller's
// heartbeat.
type Status string // @name HeartbeatStatus

const (
	// StatusAlive represents a controller that has reported a heartbeat recently
	// enough to be considered alive.
	StatusAlive Status = "Alive"
	// StatusDead represents a controller that has not reported a heartbeat
	// recently enough to be considered alive.
	StatusDead Status = "Dead"
)

// Heartbeat describes what the heartbeat system knows about a single
// controller.
type Heartbeat struct {
	// Controller is the name of the controller as reported in the Lease's
	// kargoapi.LabelKeyController label. This may be empty if the controller that
	// produced the heartbeat was unnamed.
	Controller string `json:"controller"`
	// Status is point-in-time liveness synthesized from a heartbeat record.
	Status Status `json:"status"`
	// Timestamp is the timestamp of the heartbeat. nil when the underlying record
	// carried no parseable timestamp.
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

// GetAll returns a map of Heartbeats from every controller that has reported
// in, indexed by controller name. Any controller without a corresponding key in
// the map has never produced a heartbeat and can therefore be assumed by the
// caller to be dead or nonexistent.
func GetAll(
	ctx context.Context,
	c client.Reader,
	namespace string,
) (map[string]Heartbeat, error) {
	list := &coordinationv1.LeaseList{}
	if err := c.List(
		ctx, list,
		client.InNamespace(namespace),
		client.HasLabels{kargoapi.LabelKeyController},
	); err != nil {
		return nil, fmt.Errorf("listing heartbeat leases: %w", err)
	}
	now := time.Now()
	out := make(map[string]Heartbeat, len(list.Items))
	for _, lease := range list.Items {
		controllerName := lease.Labels[kargoapi.LabelKeyController]
		out[controllerName] = leaseToHeartbeat(controllerName, &lease, now)
	}
	return out, nil
}

// leaseToHeartbeat synthesizes a Heartbeat from a Lease and the current time.
func leaseToHeartbeat(
	controllerName string,
	lease *coordinationv1.Lease,
	now time.Time,
) Heartbeat {
	heartbeat := Heartbeat{Controller: controllerName, Status: StatusDead}
	if lease.Spec.RenewTime == nil {
		return heartbeat
	}
	renewedAt := lease.Spec.RenewTime.Time
	heartbeat.Timestamp = &renewedAt
	if lease.Spec.LeaseDurationSeconds == nil {
		return heartbeat
	}
	deadline := time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second
	if now.Sub(renewedAt) < deadline {
		heartbeat.Status = StatusAlive
	}
	return heartbeat
}
