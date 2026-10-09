package targetstore

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/database"
)

func TestColumns(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		target database.TargetRow
		assert func(*testing.T, []byte, []byte, error)
	}{
		{
			name:   "absent labels and params become empty objects",
			target: database.TargetRow{},
			assert: func(t *testing.T, lbls, params []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(t, `{}`, string(lbls))
				require.JSONEq(t, `{}`, string(params))
			},
		},
		{
			name: "values are kept as written",
			target: database.TargetRow{
				Labels: map[string]string{"region": "us"},
				Params: json.RawMessage(`{"cluster": {"replicas": 3}, "flag": true}`),
			},
			assert: func(t *testing.T, lbls, params []byte, err error) {
				require.NoError(t, err)
				require.JSONEq(t, `{"region":"us"}`, string(lbls))
				require.Equal(t, `{"cluster": {"replicas": 3}, "flag": true}`, string(params))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			lbls, params, err := columns(testCase.target)
			testCase.assert(t, lbls, params, err)
		})
	}
}

func TestCheckPreconditions(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	updated := time.UnixMicro(1623053350123456)
	current := database.TargetRow{ID: id, UpdatedAt: updated}
	testCases := []struct {
		name   string
		target database.TargetRow
		assert func(*testing.T, error)
	}{
		{
			name:   "no preconditions",
			target: database.TargetRow{},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:   "matching preconditions",
			target: database.TargetRow{ID: id, UpdatedAt: updated},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name:   "a different ID",
			target: database.TargetRow{ID: uuid.MustParse("99999999-2222-3333-4444-555555555555")},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "recreated")
			},
		},
		{
			name:   "a stale version",
			target: database.TargetRow{UpdatedAt: updated.Add(-time.Microsecond)},
			assert: func(t *testing.T, err error) {
				require.ErrorIs(t, err, database.ErrConflict)
				require.ErrorContains(t, err, "modified")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.assert(t, checkPreconditions(testCase.target, current))
		})
	}
}

func TestTargetError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		err    error
		expect error
	}{
		{
			name:   "duplicate name",
			err:    &pgconn.PgError{Code: "23505", ConstraintName: nameConstraint},
			expect: database.ErrAlreadyExists,
		},
		{
			name:   "check constraint",
			err:    &pgconn.PgError{Code: "23514", Message: "labels"},
			expect: database.ErrInvalid,
		},
		{
			name:   "bad value",
			err:    &pgconn.PgError{Code: "22P05", Message: "unsupported Unicode escape"},
			expect: database.ErrInvalid,
		},
		{
			name:   "sentinel passes through",
			err:    database.ErrNotFound,
			expect: database.ErrNotFound,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := targetError(testCase.err, "demo", "us-east")
			require.ErrorIs(t, err, testCase.expect)
			require.ErrorContains(t, err, `Target "us-east" in Project "demo"`)
		})
	}
	other := errors.New("boom")
	err := targetError(other, "demo", "us-east")
	require.ErrorIs(t, err, other)
	require.NotErrorIs(t, err, database.ErrInvalid)
}
