package heartbeat

import (
	"context"
	"fmt"
	"os"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	// unnamedControllerName is a substitute controller name used in liveness
	// reporting when a controller is configured without a name.
	unnamedControllerName = "unnamed"

	// leaseNamePrefix is prepended to the controller name to form the Lease name.
	leaseNamePrefix = "kargo-controller-"
)

// renewer is an implementation of controller-runtime's manager.Runnable
// interface that creates, renews, and (on shutdown) deletes a heartbeat record
// for a Kargo controller.
type renewer struct {
	client         client.Client
	namespace      string
	controllerName string
	leaseName      string
	holderIdentity string
	leaseDuration  time.Duration
	renewInterval  time.Duration
}

// NewRenewer returns an implementation of controller-runtime's manager.Runnable
// interface that creates, renews, and (on shutdown) deletes a heartbeat record
// for a Kargo controller.
func NewRenewer(
	c client.Client,
	namespace string,
	controllerName string,
	leaseDuration time.Duration,
) manager.Runnable {
	if leaseDuration <= 0 {
		panic(fmt.Sprintf(
			"heartbeat: leaseDuration must be positive; got %v",
			leaseDuration,
		))
	}
	leaseName := controllerName
	if leaseName != "" {
		leaseName = leaseNamePrefix + controllerName
	} else {
		leaseName = leaseNamePrefix + unnamedControllerName
	}
	holderIdentity, err := os.Hostname()
	if err != nil || holderIdentity == "" {
		holderIdentity = "kargo-controller"
	}
	return &renewer{
		client:         c,
		namespace:      namespace,
		controllerName: controllerName,
		leaseName:      leaseName,
		holderIdentity: holderIdentity,
		leaseDuration:  leaseDuration,
		// One third of the lease duration so a single missed renewal doesn't
		// cause the controller to be considered dead.
		renewInterval: leaseDuration / 3,
	}
}

// Start implements controller-runtime's manager.Runnable interface. It produces
// a heartbeat at a scheduled interval by creating or updating a Lease resource.
func (r *renewer) Start(ctx context.Context) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"lease.name", r.leaseName,
		"lease.duration", r.leaseDuration,
		"interval", r.renewInterval,
	)
	logger.Info("Starting controller heartbeat")

	if err := r.renew(ctx); err != nil {
		logger.Error(err, "initial heartbeat (lease) failed; will retry")
	}

	ticker := time.NewTicker(r.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := r.renew(ctx); err != nil {
				logger.Error(err, "failed to renew heartbeat (lease)")
			}
		case <-ctx.Done():
			// Use a fresh context for the shutdown delete; the original
			// is already canceled.
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			err := r.delete(shutdownCtx)
			cancel()
			if err != nil {
				logger.Error(err, "failed to delete heartbeat (lease) on shutdown")
			}
			logger.Debug("controller heartbeat stopped")
			return nil
		}
	}
}

// NeedLeaderElection implements controller-runtime's manager.Runnable
// interface. It explicitly reports false so the renewer runs on every replica
// rather than just the leader. Kargo controllers don't currently leader-elect,
// but we make this explicit for safety.
func (r *renewer) NeedLeaderElection() bool { return false }

func (r *renewer) renew(ctx context.Context) error {
	now := metav1.MicroTime{Time: time.Now()}
	// #nosec G115 -- leaseDuration is validated > 0 in NewRenewer and represents
	// a lease validity window; any sane operator-supplied value fits in int32
	// (which the Kubernetes API requires here -- LeaseDurationSeconds is *int32).
	durationSeconds := int32(r.leaseDuration.Seconds())

	cur := &coordinationv1.Lease{}
	err := r.client.Get(
		ctx,
		types.NamespacedName{
			Name:      r.leaseName,
			Namespace: r.namespace,
		},
		cur,
	)
	if apierrors.IsNotFound(err) {
		return r.client.Create(
			ctx,
			&coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.leaseName,
					Namespace: r.namespace,
					Labels: map[string]string{
						kargoapi.LabelKeyController: r.controllerName,
					},
				},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       ptr.To(r.holderIdentity),
					LeaseDurationSeconds: ptr.To(durationSeconds),
					AcquireTime:          &now,
					RenewTime:            &now,
				},
			},
		)
	}
	if err != nil {
		return fmt.Errorf("get existing lease: %w", err)
	}

	if cur.Labels == nil {
		cur.Labels = map[string]string{}
	}
	cur.Labels[kargoapi.LabelKeyController] = r.controllerName
	cur.Spec.HolderIdentity = ptr.To(r.holderIdentity)
	cur.Spec.LeaseDurationSeconds = ptr.To(durationSeconds)
	if cur.Spec.AcquireTime == nil {
		cur.Spec.AcquireTime = &now
	}
	cur.Spec.RenewTime = &now
	return r.client.Update(ctx, cur)
}

func (r *renewer) delete(ctx context.Context) error {
	if err := r.client.Delete(
		ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: r.leaseName, Namespace: r.namespace},
		},
	); !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
