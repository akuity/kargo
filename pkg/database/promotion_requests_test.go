package database

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPromotionRequestFromSnapshot(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2021, 6, 7, 8, 9, 10, 123456000, time.UTC)
	started := time.Date(2021, 6, 7, 8, 9, 11, 0, time.UTC)
	finished := time.Date(2021, 6, 7, 8, 9, 12, 0, time.UTC)
	base := PromotionRequestSnapshot{
		PromotionRequest: PromotionRequest{
			ID:        id,
			Seq:       7,
			Name:      "dev.01hxyz.abcdefg",
			Phase:     "Pending",
			CreatedAt: created,
			UpdatedAt: updated,
		},
		ProjectName: "demo",
		Stage:       "dev",
		Freight:     "abcdefg1234",
	}
	testCases := []struct {
		name     string
		snapshot func() PromotionRequestSnapshot
		assert   func(*testing.T, kargoapi.PromotionRequest)
	}{
		{
			name:     "freshly created request",
			snapshot: func() PromotionRequestSnapshot { return base },
			assert: func(t *testing.T, request kargoapi.PromotionRequest) {
				require.Equal(t, kargoapi.GroupVersion.String(), request.APIVersion)
				require.Equal(t, "PromotionRequest", request.Kind)
				require.Equal(t, "demo", request.Namespace)
				require.Equal(t, "dev.01hxyz.abcdefg", request.Name)
				require.Equal(t, types.UID(id.String()), request.UID)
				require.Equal(t, "1623053350123456", request.ResourceVersion)
				require.True(t, request.CreationTimestamp.Time.Equal(created))
				require.Equal(t, map[string]string{kargoapi.LabelKeyStage: "dev"}, request.Labels)
				require.Nil(t, request.Annotations)
				require.Equal(t, "dev", request.Spec.Stage)
				require.Equal(t, "abcdefg1234", request.Spec.Freight)
				// Required field: present and empty, never null.
				require.NotNil(t, request.Spec.Targets)
				require.Empty(t, request.Spec.Targets)
				require.Equal(t, kargoapi.PromotionRequestPhasePending, request.Status.Phase)
				require.Empty(t, request.Status.Message)
				require.Nil(t, request.Status.Targets)
				require.Nil(t, request.Status.Summary)
				require.Nil(t, request.Status.StartedAt)
				require.Nil(t, request.Status.FinishedAt)
				require.Nil(t, request.Status.Conditions)
			},
		},
		{
			name: "creator is restored as the create-actor annotation",
			snapshot: func() PromotionRequestSnapshot {
				s := base
				s.CreatedBy = "user:someone"
				return s
			},
			assert: func(t *testing.T, request kargoapi.PromotionRequest) {
				require.Equal(
					t,
					map[string]string{kargoapi.AnnotationKeyCreateActor: "user:someone"},
					request.Annotations,
				)
			},
		},
		{
			name: "targets without an outcome are in the spec only",
			snapshot: func() PromotionRequestSnapshot {
				s := base
				s.Targets = []PromotionRequestTargetRow{
					{Name: "us-east", Ordinal: 0},
					{Name: "eu-west", Ordinal: 1},
				}
				return s
			},
			assert: func(t *testing.T, request kargoapi.PromotionRequest) {
				require.Equal(t, []kargoapi.PromotionRequestTarget{
					{Name: "us-east"}, {Name: "eu-west"},
				}, request.Spec.Targets)
				require.Nil(t, request.Status.Targets)
				require.Nil(t, request.Status.Summary)
			},
		},
		{
			name: "outcomes appear in the status and are summarized",
			snapshot: func() PromotionRequestSnapshot {
				s := base
				s.Phase = "Running"
				s.Message = "2 of 3 done"
				s.StartedAt = pgtype.Timestamptz{Time: started, Valid: true}
				s.FinishedAt = pgtype.Timestamptz{Time: finished, Valid: true}
				s.Targets = []PromotionRequestTargetRow{
					{Name: "a", Promotion: "dev.a.01", Phase: "Succeeded"},
					{Name: "b", Promotion: "dev.b.01", Phase: "Running"},
					{Name: "c"}, // not acted on yet
					{Name: "d", Promotion: "dev.d.01", Phase: "Failed"},
					{Name: "e", Promotion: "dev.e.01", Phase: "Errored"},
					{Name: "f", Promotion: "dev.f.01", Phase: "Aborted"},
					{Name: "g", Promotion: "dev.g.01", Phase: "Pending"},
					{Name: "h", Promotion: "dev.h.01", Phase: "Succeeded"},
				}
				return s
			},
			assert: func(t *testing.T, request kargoapi.PromotionRequest) {
				require.Len(t, request.Spec.Targets, 8)
				require.Equal(t, kargoapi.PromotionRequestPhaseRunning, request.Status.Phase)
				require.Equal(t, "2 of 3 done", request.Status.Message)
				require.Equal(t, []kargoapi.PromotionRequestTargetStatus{
					{Name: "a", Promotion: "dev.a.01", Phase: kargoapi.PromotionPhaseSucceeded},
					{Name: "b", Promotion: "dev.b.01", Phase: kargoapi.PromotionPhaseRunning},
					{Name: "d", Promotion: "dev.d.01", Phase: kargoapi.PromotionPhaseFailed},
					{Name: "e", Promotion: "dev.e.01", Phase: kargoapi.PromotionPhaseErrored},
					{Name: "f", Promotion: "dev.f.01", Phase: kargoapi.PromotionPhaseAborted},
					{Name: "g", Promotion: "dev.g.01", Phase: kargoapi.PromotionPhasePending},
					{Name: "h", Promotion: "dev.h.01", Phase: kargoapi.PromotionPhaseSucceeded},
				}, request.Status.Targets)
				require.Equal(t, &kargoapi.PromotionRequestSummary{
					Pending: 1, Running: 1, Succeeded: 2, Failed: 1, Errored: 1, Aborted: 1,
				}, request.Status.Summary)
				require.True(t, request.Status.StartedAt.Time.Equal(started))
				require.True(t, request.Status.FinishedAt.Time.Equal(finished))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, PromotionRequestFromSnapshot(testCase.snapshot()))
		})
	}
}

func TestPromotionRequestsFromSnapshots(t *testing.T) {
	t.Parallel()
	requests := PromotionRequestsFromSnapshots([]PromotionRequestSnapshot{
		{PromotionRequest: PromotionRequest{Name: "a"}},
		{PromotionRequest: PromotionRequest{Name: "b"}},
	})
	require.Len(t, requests, 2)
	require.Equal(t, "a", requests[0].Name)
	require.Equal(t, "b", requests[1].Name)
	empty := PromotionRequestsFromSnapshots(nil)
	require.NotNil(t, empty)
	require.Empty(t, empty)
}

func TestNewPromotionRequestCreate(t *testing.T) {
	t.Parallel()
	create := NewPromotionRequestCreate(&kargoapi.PromotionRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "demo",
			Name:        "dev.01hxyz.abcdefg",
			Annotations: map[string]string{kargoapi.AnnotationKeyCreateActor: "user:someone"},
		},
		Spec: kargoapi.PromotionRequestSpec{
			Stage:   "dev",
			Freight: "abcdefg1234",
			Targets: []kargoapi.PromotionRequestTarget{{Name: "us-east"}, {Name: "eu-west"}},
		},
	})
	require.Equal(t, PromotionRequestCreate{
		CreatePromotionRequestParams: CreatePromotionRequestParams{
			ProjectName: "demo",
			Name:        "dev.01hxyz.abcdefg",
			Stage:       "dev",
			Freight:     "abcdefg1234",
			CreatedBy:   "user:someone",
		},
		Targets: []string{"us-east", "eu-west"},
	}, create)

	// No annotations and no targets: nothing panics and the slice is empty.
	create = NewPromotionRequestCreate(&kargoapi.PromotionRequest{})
	require.Empty(t, create.CreatedBy)
	require.NotNil(t, create.Targets)
	require.Empty(t, create.Targets)
}

func TestTimestampConversions(t *testing.T) {
	t.Parallel()
	require.Nil(t, optionalTime(pgtype.Timestamptz{}))
	require.False(t, timestamptz(nil).Valid)
	now := metav1.Now()
	ts := timestamptz(&now)
	require.True(t, ts.Valid)
	require.True(t, ts.Time.Equal(now.Time))
	require.True(t, optionalTime(ts).Time.Equal(now.Time))
}

func TestPromotionRequestSnapshotJSON(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	targetID := uuid.MustParse("66666666-7777-8888-9999-000000000000")
	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	started := time.Date(2020, 1, 2, 3, 4, 6, 123456000, time.UTC)
	snapshot := PromotionRequestSnapshot{
		PromotionRequest: PromotionRequest{
			ID:        id,
			Seq:       7,
			ProjectID: "project-uid",
			StageID:   "stage-uid",
			FreightID: "freight-uid",
			Name:      "dev.01j.abcdefg",
			CreatedBy: "user:someone",
			Phase:     "Running",
			Message:   "1 of 2 done",
			CreatedAt: created,
			UpdatedAt: started,
			StartedAt: pgtype.Timestamptz{Time: started, Valid: true},
		},
		ProjectName: "demo",
		Stage:       "dev",
		Freight:     "abcdefg1234",
		Targets: []PromotionRequestTargetRow{{
			PromotionRequestID: id,
			TargetID:           targetID,
			Name:               "us-east",
			Ordinal:            0,
			Promotion:          "dev.us-east.01j",
			Phase:              "Succeeded",
		}},
	}

	data, err := json.Marshal(snapshot)
	require.NoError(t, err)
	// The row's columns sit at the top level beside the names it references,
	// and an unset timestamp is null.
	require.JSONEq(t, `{
		"id": "11111111-2222-3333-4444-555555555555",
		"seq": 7,
		"project_id": "project-uid",
		"stage_id": "stage-uid",
		"freight_id": "freight-uid",
		"name": "dev.01j.abcdefg",
		"created_by": "user:someone",
		"phase": "Running",
		"message": "1 of 2 done",
		"created_at": "2020-01-02T03:04:05Z",
		"updated_at": "2020-01-02T03:04:06.123456Z",
		"started_at": "2020-01-02T03:04:06.123456Z",
		"finished_at": null,
		"project_name": "demo",
		"stage": "dev",
		"freight": "abcdefg1234",
		"targets": [{
			"promotion_request_id": "11111111-2222-3333-4444-555555555555",
			"target_id": "66666666-7777-8888-9999-000000000000",
			"name": "us-east",
			"ordinal": 0,
			"promotion": "dev.us-east.01j",
			"phase": "Succeeded"
		}]
	}`, string(data))

	decoded := PromotionRequestSnapshot{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, snapshot, decoded)
}
