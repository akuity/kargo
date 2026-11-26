package promotionrolloutstore

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/event/fake"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// published is an event a store published, as a subscriber decodes it.
type published struct {
	subject string
	event   nats.Event[Key, database.PromotionRolloutRow]
}

// newRecordingSender returns a sender that records every event published
// through it, under the subjects a real publisher would use.
func newRecordingSender() *fake.Sender {
	return &fake.Sender{SubjectPrefix: nats.InternalSubjectPrefix}
}

// decodePublished returns every event the sender recorded, decoded, in the
// order they were published.
func decodePublished(t *testing.T, sender *fake.Sender) []published {
	t.Helper()
	var events []published
	for _, evt := range sender.Sent() {
		e, err := Events.Decode(evt)
		require.NoError(t, err)
		events = append(events, published{subject: evt.Subject(), event: e})
	}
	return events
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
		sender  *fake.Sender
		publish func(*store)
		assert  func(*testing.T, *fake.Sender)
	}{
		{
			name:   "created carries the row under the rollout's key",
			sender: newRecordingSender(),
			publish: func(s *store) {
				s.publishCreated(context.Background(), old)
			},
			assert: func(t *testing.T, sender *fake.Sender) {
				events := decodePublished(t, sender)
				require.Len(t, events, 1)
				e := events[0]
				require.Equal(t, "akuity.kargo.internal.promotion.rollout.created", e.subject)
				require.Equal(t, nats.Created, e.event.Kind)
				require.Equal(t, Key{Project: "demo", Name: "prod.01A.abc"}, e.event.Key)
				require.Nil(t, e.event.Old)
				require.Equal(t, old, *e.event.New)
			},
		},
		{
			name:   "updated carries the row before and after",
			sender: newRecordingSender(),
			publish: func(s *store) {
				s.publishUpdated(context.Background(), old, updated)
			},
			assert: func(t *testing.T, sender *fake.Sender) {
				events := decodePublished(t, sender)
				require.Len(t, events, 1)
				e := events[0]
				require.Equal(t, "akuity.kargo.internal.promotion.rollout.updated", e.subject)
				require.Equal(t, nats.Updated, e.event.Kind)
				require.Equal(t, Key{Project: "demo", Name: "prod.01A.abc"}, e.event.Key)
				require.Equal(t, old, *e.event.Old)
				require.Equal(t, updated, *e.event.New)
			},
		},
		{
			name: "a failure to publish is logged, not raised",
			sender: &fake.Sender{
				SubjectPrefix: nats.InternalSubjectPrefix,
				SendErr:       errors.New("disconnected"),
			},
			publish: func(s *store) {
				s.publishCreated(context.Background(), old)
				s.publishUpdated(context.Background(), old, updated)
			},
			assert: func(t *testing.T, sender *fake.Sender) {
				require.Empty(t, sender.Sent())
			},
		},
		{
			name: "no publisher, nothing published",
			publish: func(s *store) {
				s.publishCreated(context.Background(), old)
				s.publishUpdated(context.Background(), old, updated)
			},
			assert: func(*testing.T, *fake.Sender) {},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s := &store{}
			if testCase.sender != nil {
				s.events = nats.NewPublisherWithSender(testCase.sender)
			}
			testCase.publish(s)
			testCase.assert(t, testCase.sender)
		})
	}
}
