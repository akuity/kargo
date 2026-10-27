package fake

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/event"
)

func TestSender_Send(t *testing.T) {
	testCases := []struct {
		name          string
		sendErr       error
		subjectPrefix string
		eventType     string
		assert        func(*testing.T, *Sender, error)
	}{
		{
			name:      "SendErr set",
			sendErr:   errors.New("something went wrong"),
			eventType: string(kargoapi.EventTypePromotionCreated),
			assert: func(t *testing.T, s *Sender, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Empty(t, s.Sent())
			},
		},
		{
			name:      "invalid event type",
			eventType: "Promotion.Created",
			assert: func(t *testing.T, s *Sender, err error) {
				require.ErrorIs(t, err, event.ErrInvalidEventType)
				require.Empty(t, s.Sent())
			},
		},
		{
			name:      "success",
			eventType: string(kargoapi.EventTypePromotionCreated),
			assert: func(t *testing.T, s *Sender, err error) {
				require.NoError(t, err)
				sent := s.Sent()
				require.Len(t, sent, 1)
				require.Equal(
					t,
					"akuity.kargo.events.promotion.created",
					sent[0].Subject(),
				)
			},
		},
		{
			name:          "success with custom subject prefix",
			subjectPrefix: "akuity.kargo.things",
			eventType:     string(kargoapi.EventTypePromotionCreated),
			assert: func(t *testing.T, s *Sender, err error) {
				require.NoError(t, err)
				sent := s.Sent()
				require.Len(t, sent, 1)
				require.Equal(
					t,
					"akuity.kargo.things.promotion.created",
					sent[0].Subject(),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			evt, err := event.NewCloudEvent(testCase.eventType, nil)
			require.NoError(t, err)
			s := &Sender{
				SendErr:       testCase.sendErr,
				SubjectPrefix: testCase.subjectPrefix,
			}
			err = s.Send(t.Context(), evt)
			testCase.assert(t, s, err)
		})
	}
}
