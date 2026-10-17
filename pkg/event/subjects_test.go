package event

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewKargoSubject(t *testing.T) {
	require.Equal(t, "akuity.kargo.my-topic", NewKargoSubject("my-topic"))
}

func TestNewEventSubject(t *testing.T) {
	t.Run("valid event type", func(t *testing.T) {
		subject, err := NewEventSubject(string(kargoapi.EventTypePromotionCreated))
		require.NoError(t, err)
		require.Equal(t, "akuity.kargo.events.promotion.created", subject)
	})
	t.Run("invalid event type", func(t *testing.T) {
		_, err := NewEventSubject("Promotion.Created")
		require.ErrorIs(t, err, ErrInvalidEventType)
	})
}

func TestSubjectFromType(t *testing.T) {
	t.Run("valid event type", func(t *testing.T) {
		subject, err := SubjectFromType(
			"akuity.kargo.things",
			string(kargoapi.EventTypePromotionCreated),
		)
		require.NoError(t, err)
		require.Equal(t, "akuity.kargo.things.promotion.created", subject)
	})
	t.Run("invalid event type", func(t *testing.T) {
		_, err := SubjectFromType("akuity.kargo.things", "Promotion.Created")
		require.ErrorIs(t, err, ErrInvalidEventType)
	})
}

func TestSubjectSuffix(t *testing.T) {
	testCases := []struct {
		eventType string
		expected  string
	}{
		// Every built-in event type
		{string(kargoapi.EventTypePromotionCreated), "promotion.created"},
		{string(kargoapi.EventTypePromotionSucceeded), "promotion.succeeded"},
		{string(kargoapi.EventTypePromotionFailed), "promotion.failed"},
		{string(kargoapi.EventTypePromotionErrored), "promotion.errored"},
		{string(kargoapi.EventTypePromotionAborted), "promotion.aborted"},
		{string(kargoapi.EventTypePromotionDiscarded), "promotion.discarded"},
		{string(kargoapi.EventTypeFreightCreated), "freight.created"},
		{string(kargoapi.EventTypeFreightApproved), "freight.approved"},
		{
			string(kargoapi.EventTypeFreightVerificationSucceeded),
			"freight.verification.succeeded",
		},
		{
			string(kargoapi.EventTypeFreightVerificationFailed),
			"freight.verification.failed",
		},
		{
			string(kargoapi.EventTypeFreightVerificationErrored),
			"freight.verification.errored",
		},
		{
			string(kargoapi.EventTypeFreightVerificationAborted),
			"freight.verification.aborted",
		},
		{
			string(kargoapi.EventTypeFreightVerificationInconclusive),
			"freight.verification.inconclusive",
		},
		{
			string(kargoapi.EventTypeFreightVerificationUnknown),
			"freight.verification.unknown",
		},
		{string(kargoapi.EventTypeAPITokenCreated), "api.token.created"},
		{string(kargoapi.EventTypeAPITokenDeleted), "api.token.deleted"},
		// Edge cases
		{"Created", "created"},
		{"created", "created"},
		{"camelCase", "camel.case"},
		{"API", "api"},
		{"RefreshAPI", "refresh.api"},
		{"HTTPRouteUpdated", "http.route.updated"},
		{"V2Created", "v2.created"},
		{"S3BucketCreated", "s3.bucket.created"},
		{"Route53Updated", "route53.updated"},
		{"ABTest", "ab.test"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.eventType, func(t *testing.T) {
			t.Parallel()
			suffix, err := SubjectSuffix(testCase.eventType)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, suffix)
		})
	}

	invalid := []string{
		"",
		"Promotion.Created",
		"Promotion Created",
		"Promotion*",
		"Promotion>",
		"Promotion-Created",
		"Promotion_Created",
		"Prömotion",
	}
	for _, eventType := range invalid {
		t.Run("invalid "+eventType, func(t *testing.T) {
			t.Parallel()
			_, err := SubjectSuffix(eventType)
			require.ErrorIs(t, err, ErrInvalidEventType)
		})
	}
}
