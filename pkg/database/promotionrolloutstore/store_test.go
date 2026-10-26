package promotionrolloutstore

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestKeyString(t *testing.T) {
	t.Parallel()
	require.Equal(t, "demo/prod.01A.abc", Key{Project: "demo", Name: "prod.01A.abc"}.String())
}

func TestCheckPreconditions(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	updated := time.UnixMicro(1623053350123456)
	current := database.PromotionRolloutRow{ID: id, UpdatedAt: updated}
	testCases := []struct {
		name    string
		rollout database.PromotionRolloutRow
		assert  func(*testing.T, error)
	}{
		{
			name:    "no preconditions",
			rollout: database.PromotionRolloutRow{},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "matching preconditions",
			rollout: database.PromotionRolloutRow{ID: id, UpdatedAt: updated},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "different id",
			rollout: database.PromotionRolloutRow{ID: uuid.New()},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "deleted and recreated")
			},
		},
		{
			name:    "different version",
			rollout: database.PromotionRolloutRow{UpdatedAt: updated.Add(time.Microsecond)},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "has been modified")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, checkPreconditions(testCase.rollout, current))
		})
	}
}

func TestRolloutError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		err    error
		assert func(*testing.T, error)
	}{
		{
			name: "duplicate name",
			err:  &pgconn.PgError{Code: "23505", ConstraintName: nameConstraint},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrAlreadyExists)
			},
		},
		{
			name: "other unique violation",
			err:  &pgconn.PgError{Code: "23505", ConstraintName: "something_else"},
			assert: func(t *testing.T, err error) {
				require.NotErrorIs(t, err, database.ErrAlreadyExists)
			},
		},
		{
			name: "vanished Target",
			err:  &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "foreign key")
			},
		},
		{
			name: "check constraint",
			err:  &pgconn.PgError{Code: "23514", Message: "violates check constraint"},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrInvalid)
			},
		},
		{
			name: "data exception",
			err:  &pgconn.PgError{Code: "22P02", Message: "invalid input syntax"},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrInvalid)
			},
		},
		{
			name: "sentinel passes through",
			err:  database.ErrNotFound,
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrNotFound)
			},
		},
		{
			name: "unrelated error",
			err:  errors.New("connection reset"),
			assert: func(t *testing.T, err error) {
				require.NotErrorIs(t, err, database.ErrInvalid)
				require.ErrorContains(t, err, "connection reset")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := rolloutError(testCase.err, "demo", "prod.01A.abc")
			require.ErrorContains(t, err, `PromotionRollout "prod.01A.abc" in Project "demo"`)
			testCase.assert(t, err)
		})
	}
}

func TestGroupTargets(t *testing.T) {
	t.Parallel()
	first, second := uuid.New(), uuid.New()
	rollouts := []Rollout{
		{
			PromotionRolloutRow: database.PromotionRolloutRow{ID: first},
			Targets:             []database.ListPromotionRolloutTargetsRow{},
		},
		{
			PromotionRolloutRow: database.PromotionRolloutRow{ID: second},
			Targets:             []database.ListPromotionRolloutTargetsRow{},
		},
	}
	targets := []database.ListPromotionRolloutTargetsRow{
		{PromotionRolloutID: first, TargetName: "us-east"},
		{PromotionRolloutID: first, TargetName: "us-west"},
		{PromotionRolloutID: uuid.New(), TargetName: "orphan"},
	}
	grouped := groupTargets(rollouts, targets)
	require.Equal(t, []string{"us-east", "us-west"}, targetNames(grouped[0]))
	require.NotNil(t, grouped[1].Targets)
	require.Empty(t, grouped[1].Targets)
}

func targetNames(rollout Rollout) []string {
	names := make([]string, len(rollout.Targets))
	for i, target := range rollout.Targets {
		names[i] = target.TargetName
	}
	return names
}

func TestToAPI(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	finished := created.Add(2 * time.Minute)
	base := database.PromotionRolloutRow{
		ID:                id,
		ProjectName:       "demo",
		Name:              "prod.01A.abc",
		StageName:         "prod",
		Number:            7,
		FreightName:       "abc",
		MaxConcurrent:     2,
		MaxFailures:       1,
		CreatedBy:         "someone",
		CreatedAt:         created,
		UpdatedAt:         updated,
		TargetsResolvedAt: &updated,
		StatusPhase:       "Pending",
	}
	testCases := []struct {
		name   string
		row    func() database.PromotionRolloutRow
		assert func(*testing.T, *kargoapi.PromotionRollout)
	}{
		{
			name: "identity and spec",
			row:  func() database.PromotionRolloutRow { return base },
			assert: func(t *testing.T, rollout *kargoapi.PromotionRollout) {
				require.Equal(t, "demo", rollout.Namespace)
				require.Equal(t, "prod.01A.abc", rollout.Name)
				require.Equal(t, id.String(), string(rollout.UID))
				require.Equal(t, strconv.FormatInt(updated.UnixMicro(), 10), rollout.ResourceVersion)
				require.True(t, rollout.CreationTimestamp.Time.Equal(created))
				require.Equal(t, "prod", rollout.Labels[kargoapi.LabelKeyStage])
				require.Equal(t, "someone", rollout.Annotations[kargoapi.AnnotationKeyCreateActor])
				require.EqualValues(t, 7, rollout.Number)
				require.Equal(t, "prod", rollout.Spec.Stage)
				require.Equal(t, "abc", rollout.Spec.Freight)
				require.Equal(
					t,
					kargoapi.PromotionRolloutUpdateStrategy{MaxConcurrent: 2, MaxFailures: 1},
					rollout.Spec.UpdateStrategy,
				)
				require.Equal(t, []kargoapi.PromotionRolloutTarget{{Name: "us-east"}, {Name: "us-west"}}, rollout.Spec.Targets)
				require.Nil(t, rollout.Status.Targets)
				require.Nil(t, rollout.Status.Summary)
			},
		},
		{
			name: "a rollout with no recorded creator carries no annotation",
			row: func() database.PromotionRolloutRow {
				row := base
				row.CreatedBy = ""
				return row
			},
			assert: func(t *testing.T, rollout *kargoapi.PromotionRollout) {
				require.Nil(t, rollout.Annotations)
			},
		},
		{
			name: "targets are absent until resolved, then a list even when empty",
			row: func() database.PromotionRolloutRow {
				row := base
				row.TargetsResolvedAt = nil
				return row
			},
			assert: func(t *testing.T, rollout *kargoapi.PromotionRollout) {
				require.Nil(t, rollout.Spec.Targets)
				resolved := ToAPI(Rollout{PromotionRolloutRow: base, Targets: []database.ListPromotionRolloutTargetsRow{}})
				require.NotNil(t, resolved.Spec.Targets)
				require.Empty(t, resolved.Spec.Targets)
			},
		},
		{
			name: "a pending rollout is not yet Ready",
			row:  func() database.PromotionRolloutRow { return base },
			assert: func(t *testing.T, rollout *kargoapi.PromotionRollout) {
				require.Equal(t, kargoapi.PromotionRolloutPhasePending, rollout.Status.Phase)
				require.Nil(t, rollout.Status.StartedAt)
				require.Nil(t, rollout.Status.FinishedAt)
				require.Equal(t, []metav1.Condition{{
					Type:   kargoapi.ConditionTypeReady,
					Status: metav1.ConditionUnknown,
					Reason: "Pending",
				}}, rollout.Status.Conditions)
			},
		},
		{
			name: "a succeeded rollout is Ready",
			row: func() database.PromotionRolloutRow {
				row := base
				row.StatusPhase = "Succeeded"
				row.StatusStartedAt = &updated
				row.StatusFinishedAt = &finished
				return row
			},
			assert: func(t *testing.T, rollout *kargoapi.PromotionRollout) {
				require.Equal(t, kargoapi.PromotionRolloutPhaseSucceeded, rollout.Status.Phase)
				require.True(t, rollout.Status.StartedAt.Time.Equal(updated))
				require.True(t, rollout.Status.FinishedAt.Time.Equal(finished))
				require.Equal(t, metav1.ConditionTrue, rollout.Status.Conditions[0].Status)
				require.Equal(t, "Succeeded", rollout.Status.Conditions[0].Reason)
			},
		},
		{
			name: "an errored rollout is not Ready, with its reason and message",
			row: func() database.PromotionRolloutRow {
				row := base
				row.StatusPhase = "Errored"
				row.StatusReason = "EnterpriseOnlyFeature"
				row.StatusMessage = "nope"
				return row
			},
			assert: func(t *testing.T, rollout *kargoapi.PromotionRollout) {
				require.Equal(t, "nope", rollout.Status.Message)
				require.Equal(t, []metav1.Condition{{
					Type:    kargoapi.ConditionTypeReady,
					Status:  metav1.ConditionFalse,
					Reason:  "EnterpriseOnlyFeature",
					Message: "nope",
				}}, rollout.Status.Conditions)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rollout := Rollout{
				PromotionRolloutRow: testCase.row(),
				Targets: []database.ListPromotionRolloutTargetsRow{
					{PromotionRolloutID: id, TargetName: "us-east"},
					{PromotionRolloutID: id, TargetName: "us-west"},
				},
			}
			testCase.assert(t, ToAPI(rollout))
		})
	}
}

func TestFromSpec(t *testing.T) {
	t.Parallel()
	row := FromSpec("prod.01A.abc", kargoapi.PromotionRolloutSpec{
		Stage:   "prod",
		Freight: "abc",
		Targets: []kargoapi.PromotionRolloutTarget{{Name: "ignored"}},
		UpdateStrategy: kargoapi.PromotionRolloutUpdateStrategy{
			MaxConcurrent: 3,
			MaxFailures:   1,
		},
	})
	require.Equal(t, database.PromotionRolloutRow{
		Name:          "prod.01A.abc",
		StageName:     "prod",
		FreightName:   "abc",
		MaxConcurrent: 3,
		MaxFailures:   1,
	}, row)
}
