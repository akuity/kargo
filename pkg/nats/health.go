package nats

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nats-io/nats.go"
)

const defaultLivenessGracePeriod = 10 * time.Minute

// HealthCheckerConfig represents configuration for a HealthChecker.
type HealthCheckerConfig struct {
	// LivenessGracePeriod is how long a connection may stay disconnected
	// before it is reported as not live. It should comfortably exceed the
	// time it takes rotated credentials to reach the component (the kubelet's
	// sync period plus its Secret cache TTL), so that a routine rotation never
	// causes a restart. This is the last line of defense in case credentials
	// are no longer valid.
	LivenessGracePeriod time.Duration `envconfig:"NATS_LIVENESS_GRACE_PERIOD" default:"10m"`
}

// HealthCheckerConfigFromEnv returns a HealthCheckerConfig populated from
// environment variables.
func HealthCheckerConfigFromEnv() HealthCheckerConfig {
	cfg := HealthCheckerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// statusSource is the part of a *nats.Conn a HealthChecker depends on.
type statusSource interface {
	Status() nats.Status
	StatusChanged(statuses ...nats.Status) chan nats.Status
}

// HealthChecker reports on the health of a NATS connection in the terms used
// by Kubernetes probes.
//
// It follows the connection's status changes as they happen rather than
// asking the connection when it is checked. The connection holds its lock for
// the whole of every reconnect attempt, including the dial and the handshake,
// so asking it would block a probe for as long as an attempt takes. That can
// be longer than a probe's timeout, which would fail a liveness probe long
// before the grace period runs out.
type HealthChecker struct {
	gracePeriod time.Duration

	mu     sync.Mutex
	status nats.Status
	// disconnectedSince is when the connection was disconnected. It is zero
	// while the connection is connected.
	disconnectedSince time.Time
}

// NewHealthChecker returns a HealthChecker for the provided connection. It
// follows the connection's status until the connection is closed.
func NewHealthChecker(conn *nats.Conn, cfg HealthCheckerConfig) *HealthChecker {
	return newHealthChecker(conn, cfg)
}

func newHealthChecker(conn statusSource, cfg HealthCheckerConfig) *HealthChecker {
	gracePeriod := cfg.LivenessGracePeriod
	if gracePeriod <= 0 {
		gracePeriod = defaultLivenessGracePeriod
	}
	h := &HealthChecker{gracePeriod: gracePeriod}
	// Listen before reading the current status so that no change can fall
	// between the two. A change that lands in between is applied again by the
	// listener, which is harmless.
	changes := conn.StatusChanged()
	h.setStatus(conn.Status())
	go h.follow(conn, changes)
	return h
}

// follow records the connection's status each time it is notified of a
// change, until the connection is closed.
//
// It reads the status from the connection rather than taking it from the
// notification. The connection drops notifications rather than block when the
// channel is full, which a run of failed reconnect attempts can cause, and
// the one dropped could be the last one, leaving every notification still in
// the channel stale. Reading the status can block for as long as a reconnect
// attempt takes, but that only delays this goroutine, never a probe.
func (h *HealthChecker) follow(conn statusSource, changes <-chan nats.Status) {
	for range changes {
		status := conn.Status()
		h.setStatus(status)
		if status == nats.CLOSED {
			// The library never reopens a closed connection, so there is
			// nothing more to follow.
			return
		}
	}
}

func (h *HealthChecker) setStatus(status nats.Status) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
	if status == nats.CONNECTED {
		h.disconnectedSince = time.Time{}
	} else if h.disconnectedSince.IsZero() {
		h.disconnectedSince = time.Now()
	}
}

// Ready returns an error if the connection is not currently connected.
func (h *HealthChecker) Ready() error {
	if status, _ := h.observe(); status != nats.CONNECTED {
		return fmt.Errorf("NATS connection is %s", status)
	}
	return nil
}

// Live returns an error if the connection has been closed for good, or has
// been disconnected for longer than the configured grace period.
func (h *HealthChecker) Live() error {
	status, disconnectedFor := h.observe()
	switch {
	case status == nats.CLOSED:
		return errors.New("NATS connection is closed")
	case disconnectedFor > h.gracePeriod:
		return fmt.Errorf(
			"NATS connection has been disconnected for %s",
			disconnectedFor.Truncate(time.Second),
		)
	}
	return nil
}

// observe returns the connection's last known status and, if it is not
// connected, how long it has been disconnected.
func (h *HealthChecker) observe() (nats.Status, time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.status == nats.CONNECTED {
		return h.status, 0
	}
	return h.status, time.Since(h.disconnectedSince)
}
