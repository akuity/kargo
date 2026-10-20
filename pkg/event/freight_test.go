package event

import (
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_newFreightVerification(t *testing.T) {
	testCases := map[string]struct {
		verificationInfo *kargoapi.VerificationInfo
		expected         FreightVerification
	}{
		"complete verification info": {
			verificationInfo: &kargoapi.VerificationInfo{
				StartTime:  &metav1.Time{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
				FinishTime: &metav1.Time{Time: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
				AnalysisRun: &kargoapi.AnalysisRunReference{
					Name: "test-analysis",
				},
			},
			expected: FreightVerification{
				StartTime:       ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				FinishTime:      ptr.To(time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)),
				AnalysisRunName: ptr.To("test-analysis"),
			},
		},
		"nil verification info": {
			verificationInfo: nil,
			expected:         FreightVerification{},
		},
		"partial verification info": {
			verificationInfo: &kargoapi.VerificationInfo{
				StartTime: &metav1.Time{Time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			},
			expected: FreightVerification{
				StartTime: ptr.To(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := newFreightVerification(tc.verificationInfo)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestNewFreightCommon(t *testing.T) {
	testCases := map[string]struct {
		message         string
		actor           string
		stageName       string
		freight         *kargoapi.Freight
		expectedCommon  Common
		expectedFreight Freight
	}{
		"complete freight": {
			message:   "test message",
			actor:     "test-actor",
			stageName: "test-stage",
			freight: &kargoapi.Freight{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-freight",
					Namespace:         "test-project",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
				Alias: "v1.0.0",
			},
			expectedCommon: Common{
				Project: "test-project",
				Message: "test message",
				Actor:   ptr.To("test-actor"),
			},
			expectedFreight: Freight{
				CreateTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Name:       "test-freight",
				StageName:  "test-stage",
				Alias:      ptr.To("v1.0.0"),
			},
		},
		"nil freight": {
			message:   "test message",
			actor:     "test-actor",
			stageName: "test-stage",
			freight:   nil,
			expectedCommon: Common{
				Message: "test message",
				Actor:   ptr.To("test-actor"),
			},
			expectedFreight: Freight{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			common, freight := NewFreightCommon(tc.message, tc.actor, tc.stageName, tc.freight)
			require.Equal(t, tc.expectedCommon, common)
			require.Equal(t, tc.expectedFreight, freight)
		})
	}
}

func TestNewFreightVerification(t *testing.T) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	finishTime := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	newVerification := func(phase kargoapi.VerificationPhase) *kargoapi.VerificationInfo {
		return &kargoapi.VerificationInfo{
			Phase:       phase,
			Message:     "verification message",
			StartTime:   &metav1.Time{Time: startTime},
			FinishTime:  &metav1.Time{Time: finishTime},
			AnalysisRun: &kargoapi.AnalysisRunReference{Name: "test-analysis"},
		}
	}
	expectedParts := func(message string) (Common, Freight, FreightVerification) {
		return Common{
				Project: "test-project",
				Actor:   ptr.To("test-actor"),
				Message: message,
			},
			Freight{
				Name:       "test-freight",
				StageName:  "test-stage",
				CreateTime: freight.CreationTimestamp.Time,
			},
			FreightVerification{
				StartTime:                    &startTime,
				FinishTime:                   &finishTime,
				AnalysisRunName:              ptr.To("test-analysis"),
				AnalysisTriggeredByPromotion: ptr.To("test-promotion"),
			}
	}

	testCases := []struct {
		name         string
		phase        kargoapi.VerificationPhase
		expectedType kargoapi.EventType
		assert       func(*testing.T, cloudevents.Event)
	}{
		{
			name:         "successful",
			phase:        kargoapi.VerificationPhaseSuccessful,
			expectedType: kargoapi.EventTypeFreightVerificationSucceeded,
			assert: func(t *testing.T, evt cloudevents.Event) {
				// A successful verification gets a fixed message
				common, fr, ver := expectedParts("Freight verification succeeded")
				require.Equal(
					t,
					&FreightVerificationSucceeded{Common: common, Freight: fr, FreightVerification: ver},
					dataAs[FreightVerificationSucceeded](t, evt),
				)
			},
		},
		{
			name:         "failed",
			phase:        kargoapi.VerificationPhaseFailed,
			expectedType: kargoapi.EventTypeFreightVerificationFailed,
			assert: func(t *testing.T, evt cloudevents.Event) {
				common, fr, ver := expectedParts("verification message")
				require.Equal(
					t,
					&FreightVerificationFailed{Common: common, Freight: fr, FreightVerification: ver},
					dataAs[FreightVerificationFailed](t, evt),
				)
			},
		},
		{
			name:         "errored",
			phase:        kargoapi.VerificationPhaseError,
			expectedType: kargoapi.EventTypeFreightVerificationErrored,
			assert: func(t *testing.T, evt cloudevents.Event) {
				common, fr, ver := expectedParts("verification message")
				require.Equal(
					t,
					&FreightVerificationErrored{Common: common, Freight: fr, FreightVerification: ver},
					dataAs[FreightVerificationErrored](t, evt),
				)
			},
		},
		{
			name:         "aborted",
			phase:        kargoapi.VerificationPhaseAborted,
			expectedType: kargoapi.EventTypeFreightVerificationAborted,
			assert: func(t *testing.T, evt cloudevents.Event) {
				common, fr, ver := expectedParts("verification message")
				require.Equal(
					t,
					&FreightVerificationAborted{Common: common, Freight: fr, FreightVerification: ver},
					dataAs[FreightVerificationAborted](t, evt),
				)
			},
		},
		{
			name:         "inconclusive",
			phase:        kargoapi.VerificationPhaseInconclusive,
			expectedType: kargoapi.EventTypeFreightVerificationInconclusive,
			assert: func(t *testing.T, evt cloudevents.Event) {
				common, fr, ver := expectedParts("verification message")
				require.Equal(
					t,
					&FreightVerificationInconclusive{Common: common, Freight: fr, FreightVerification: ver},
					dataAs[FreightVerificationInconclusive](t, evt),
				)
			},
		},
		{
			name:         "any other phase",
			phase:        kargoapi.VerificationPhasePending,
			expectedType: kargoapi.EventTypeFreightVerificationUnknown,
			assert: func(t *testing.T, evt cloudevents.Event) {
				common, fr, ver := expectedParts("verification message")
				require.Equal(
					t,
					&FreightVerificationUnknown{Common: common, Freight: fr, FreightVerification: ver},
					dataAs[FreightVerificationUnknown](t, evt),
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			evt, err := NewFreightVerification(
				"test-actor",
				"test-stage",
				freight,
				newVerification(testCase.phase),
				ptr.To("test-promotion"),
			)
			require.NoError(t, err)
			requireCloudEvent(
				t,
				evt,
				testCase.expectedType,
				"Freight",
			)
			testCase.assert(t, evt)
		})
	}
}

func TestNewFreightApproved(t *testing.T) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	evt, err := NewFreightApproved("Freight approved", "test-actor", "test-stage", freight)
	require.NoError(t, err)

	requireCloudEvent(
		t,
		evt,
		kargoapi.EventTypeFreightApproved,
		"Freight",
	)
	require.Equal(
		t,
		&FreightApproved{
			Common: Common{
				Project: "test-project",
				Actor:   ptr.To("test-actor"),
				Message: "Freight approved",
			},
			Freight: Freight{
				Name:       "test-freight",
				StageName:  "test-stage",
				CreateTime: freight.CreationTimestamp.Time,
			},
		},
		dataAs[FreightApproved](t, evt),
	)
}

func TestNewFreightCreated(t *testing.T) {
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-freight",
			Namespace:         "test-project",
			CreationTimestamp: metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Origin: kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: "test-warehouse",
		},
	}

	evt, err := NewFreightCreated("Freight created", "test-actor", freight)
	require.NoError(t, err)

	requireCloudEvent(
		t,
		evt,
		kargoapi.EventTypeFreightCreated,
		"Freight",
	)
	require.Equal(
		t,
		&FreightCreated{
			Common: Common{
				Project: "test-project",
				Actor:   ptr.To("test-actor"),
				Message: "Freight created",
			},
			Freight: Freight{
				Name:          "test-freight",
				WarehouseName: "test-warehouse",
				CreateTime:    freight.CreationTimestamp.Time,
			},
		},
		dataAs[FreightCreated](t, evt),
	)
}
