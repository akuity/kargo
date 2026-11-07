package promotionrolloutstore

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// recordingPublisher is a nats.Publisher that keeps every event published to
// it, decoded, for a test to inspect. It is safe for the parallel cases of
// one test to share.
type recordingPublisher struct {
	mu     sync.Mutex
	events []published
	err    error
}

type published struct {
	subject string
	event   nats.Event[Key, database.PromotionRolloutRow]
}

func (p *recordingPublisher) Publish(subject string, data []byte) error {
	if p.err != nil {
		return p.err
	}
	var e nats.Event[Key, database.PromotionRolloutRow]
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, published{subject: subject, event: e})
	return nil
}

func TestPublish(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	old := database.PromotionRolloutRow{
		ID:          id,
		ProjectName: "demo",
		Name:        "prod.01A.abc",
		StageName:   "prod",
		StatusPhase: "Pending",
	}
	updated := old
	updated.StatusPhase = "Running"

	testCases := []struct {
		name    string
		events  *recordingPublisher
		publish func(*store)
		assert  func(*testing.T, *recordingPublisher)
	}{
		{
			name:   "created carries the row under the rollout's key",
			events: &recordingPublisher{},
			publish: func(s *store) {
				s.publishCreated(context.Background(), old)
			},
			assert: func(t *testing.T, events *recordingPublisher) {
				require.Len(t, events.events, 1)
				e := events.events[0]
				require.Equal(t, "akuity.platform.internal.promotionrollouts.created", e.subject)
				require.Equal(t, nats.Created, e.event.Kind)
				require.Equal(t, Key{Project: "demo", Name: "prod.01A.abc"}, e.event.Key)
				require.Nil(t, e.event.Old)
				require.Equal(t, old, *e.event.New)
			},
		},
		{
			name:   "updated carries the row before and after",
			events: &recordingPublisher{},
			publish: func(s *store) {
				s.publishUpdated(context.Background(), old, updated)
			},
			assert: func(t *testing.T, events *recordingPublisher) {
				require.Len(t, events.events, 1)
				e := events.events[0]
				require.Equal(t, "akuity.platform.internal.promotionrollouts.updated", e.subject)
				require.Equal(t, nats.Updated, e.event.Kind)
				require.Equal(t, Key{Project: "demo", Name: "prod.01A.abc"}, e.event.Key)
				require.Equal(t, old, *e.event.Old)
				require.Equal(t, updated, *e.event.New)
			},
		},
		{
			name:   "a failure to publish is logged, not raised",
			events: &recordingPublisher{err: errors.New("disconnected")},
			publish: func(s *store) {
				s.publishCreated(context.Background(), old)
				s.publishUpdated(context.Background(), old, updated)
			},
			assert: func(t *testing.T, events *recordingPublisher) {
				require.Empty(t, events.events)
			},
		},
		{
			name: "no publisher, nothing published",
			publish: func(s *store) {
				s.publishCreated(context.Background(), old)
				s.publishUpdated(context.Background(), old, updated)
			},
			assert: func(*testing.T, *recordingPublisher) {},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s := &store{}
			if testCase.events != nil {
				s.events = testCase.events
			}
			testCase.publish(s)
			testCase.assert(t, testCase.events)
		})
	}
}
