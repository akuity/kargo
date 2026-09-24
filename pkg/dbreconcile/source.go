package dbreconcile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/akuity/kargo/pkg/logging"
)

// Source delivers events about resources of type T, identified by keys of
// type K.
type Source[K comparable, T any] interface {
	// Start begins delivering events to emit and returns once delivery is set
	// up. Delivery stops when ctx is done.
	//
	// queueGroup names the group through which the replicas of a controller
	// share this source's events, so that each event is delivered to only one
	// of them. The controller assigns it; an empty group means every replica
	// receives every event.
	Start(ctx context.Context, emit func(context.Context, Event[K, T]), queueGroup string) error
}

// SubjectOption configures a Source created by Subject.
type SubjectOption func(*subjectOptions)

type subjectOptions struct {
	fanOut bool
}

// WithoutQueueGroup makes every replica of a controller receive every event
// on the subject, rather than sharing them. It is for a controller whose
// replicas each need to see every event: one that keeps state in memory, say.
func WithoutQueueGroup() SubjectOption {
	return func(o *subjectOptions) { o.fanOut = true }
}

// subjectSource is a Source that subscribes to a NATS subject.
type subjectSource[K comparable, T any] struct {
	conn    *nats.Conn
	subject string
	opts    subjectOptions
}

// Subject returns a Source that delivers the events published to a NATS
// subject. The subject may use wildcards: kargo.promotionrequests.> delivers
// every event about PromotionRequests. A message that is not a valid Event is
// logged and dropped; a controller's resync recovers whatever it would have
// caused.
//
// The subscription joins the queue group its controller assigns, so replicas
// of the controller share the subject's events. Other controllers subscribed
// to the same subject are in groups of their own, and still receive every
// event.
func Subject[K comparable, T any](conn *nats.Conn, subject string, opts ...SubjectOption) Source[K, T] {
	s := &subjectSource[K, T]{conn: conn, subject: subject}
	for _, opt := range opts {
		opt(&s.opts)
	}
	return s
}

func (s *subjectSource[K, T]) Start(
	ctx context.Context,
	emit func(context.Context, Event[K, T]),
	queueGroup string,
) error {
	if s.conn == nil {
		return errors.New("a NATS connection is required")
	}
	if s.subject == "" {
		return errors.New("a subject is required")
	}
	if s.opts.fanOut {
		queueGroup = ""
	}
	logger := logging.LoggerFromContext(ctx).WithValues("subject", s.subject, "queueGroup", queueGroup)
	handle := func(msg *nats.Msg) {
		if ctx.Err() != nil {
			return
		}
		e := Event[K, T]{}
		if decodeErr := json.Unmarshal(msg.Data, &e); decodeErr != nil {
			logger.Error(decodeErr, "dropping message that is not an event", "messageSubject", msg.Subject)
			return
		}
		if validateErr := e.validate(); validateErr != nil {
			logger.Error(validateErr, "dropping invalid event", "messageSubject", msg.Subject)
			return
		}
		emit(ctx, e)
	}
	var (
		sub *nats.Subscription
		err error
	)
	if queueGroup == "" {
		sub, err = s.conn.Subscribe(s.subject, handle)
	} else {
		sub, err = s.conn.QueueSubscribe(s.subject, queueGroup, handle)
	}
	if err != nil {
		return fmt.Errorf("error subscribing to %q: %w", s.subject, err)
	}
	go func() {
		<-ctx.Done()
		if unsubErr := sub.Unsubscribe(); unsubErr != nil && !errors.Is(unsubErr, nats.ErrConnectionClosed) {
			logger.Error(unsubErr, "error unsubscribing")
		}
	}()
	return nil
}
