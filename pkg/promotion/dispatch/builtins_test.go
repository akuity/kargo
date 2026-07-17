package dispatch

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/stretchr/testify/require"
)

func strTerms(args ...string) []*ast.Term {
	terms := make([]*ast.Term, len(args))
	for i, a := range args {
		terms[i] = ast.StringTerm(a)
	}
	return terms
}

func TestRRuleActive(t *testing.T) {
	t.Parallel()

	const weekdays = "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"

	testCases := []struct {
		name   string
		args   []string // recurrence, start, end, location, now
		assert func(*testing.T, *ast.Term, error)
	}{
		{
			name: "inside window",
			// 2026-07-15 is a Wednesday; 15:00 UTC is within 09:00-17:00.
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(true), term)
			},
		},
		{
			name: "before window opens",
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-15T08:59:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(false), term)
			},
		},
		{
			name: "after window closes",
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-15T17:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(false), term)
			},
		},
		{
			name: "weekend day is not an occurrence",
			// 2026-07-18 is a Saturday.
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-18T12:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(false), term)
			},
		},
		{
			name: "timezone is honored",
			// 16:00 UTC is 09:00 in Los Angeles (PDT, UTC-7).
			args: []string{weekdays, "09:00", "17:00", "America/Los_Angeles", "2026-07-15T16:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(true), term)
			},
		},
		{
			name: "timezone excludes UTC morning",
			// 08:00 UTC is 01:00 in Los Angeles.
			args: []string{weekdays, "09:00", "17:00", "America/Los_Angeles", "2026-07-15T08:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(false), term)
			},
		},
		{
			name: "midnight-crossing window is active after midnight",
			// Tuesday 22:00 - Wednesday 02:00; Wednesday 01:00 is inside the
			// occurrence that opened Tuesday.
			args: []string{weekdays, "22:00", "02:00", "UTC", "2026-07-15T01:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(true), term)
			},
		},
		{
			name: "midnight-crossing window is inactive mid-day",
			args: []string{weekdays, "22:00", "02:00", "UTC", "2026-07-15T12:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(false), term)
			},
		},
		{
			name: "empty location defaults to UTC",
			args: []string{weekdays, "09:00", "17:00", "", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.BooleanTerm(true), term)
			},
		},
		{
			name: "malformed recurrence errors",
			args: []string{"FREQ=BOGUS", "09:00", "17:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, _ *ast.Term, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid recurrence")
			},
		},
		{
			name: "malformed clock errors",
			args: []string{weekdays, "9am", "17:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, _ *ast.Term, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid start")
			},
		},
		{
			name: "malformed now errors",
			args: []string{weekdays, "09:00", "17:00", "UTC", "yesterday"},
			assert: func(t *testing.T, _ *ast.Term, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid now")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			term, err := rruleActive(rego.BuiltinContext{}, strTerms(testCase.args...))
			testCase.assert(t, term, err)
		})
	}
}

func TestRRuleNext(t *testing.T) {
	t.Parallel()

	const weekdays = "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"

	testCases := []struct {
		name   string
		args   []string // recurrence, start, location, now
		assert func(*testing.T, *ast.Term, error)
	}{
		{
			name: "same-day opening still ahead",
			args: []string{weekdays, "18:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-15T18:00:00Z"), term)
			},
		},
		{
			name: "opening passed rolls to next occurrence",
			args: []string{weekdays, "09:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-16T09:00:00Z"), term)
			},
		},
		{
			name: "weekend rolls to Monday",
			// 2026-07-17 is a Friday; after its 18:00 opening the next
			// weekday occurrence is Monday 2026-07-20.
			args: []string{weekdays, "18:00", "UTC", "2026-07-17T19:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-20T18:00:00Z"), term)
			},
		},
		{
			name: "result is rendered in UTC for non-UTC locations",
			// Next 09:00 Los Angeles (PDT) = 16:00 UTC.
			args: []string{weekdays, "09:00", "America/Los_Angeles", "2026-07-15T17:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-16T16:00:00Z"), term)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			terms := strTerms(testCase.args...)
			term, err := rruleNext(rego.BuiltinContext{}, terms[0], terms[1], terms[2], terms[3])
			testCase.assert(t, term, err)
		})
	}
}

func TestRRuleActiveEnd(t *testing.T) {
	t.Parallel()

	const weekdays = "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"

	testCases := []struct {
		name   string
		args   []string // recurrence, start, end, location, now
		assert func(*testing.T, *ast.Term, error)
	}{
		{
			name: "inside window returns its close",
			// 2026-07-15 is a Wednesday; the 09:00-17:00 occurrence closes 17:00.
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-15T17:00:00Z"), term)
			},
		},
		{
			name: "before window opens has no active occurrence",
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-15T08:59:00Z"},
			assert: func(t *testing.T, _ *ast.Term, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no active occurrence")
			},
		},
		{
			name: "at window close has no active occurrence",
			args: []string{weekdays, "09:00", "17:00", "UTC", "2026-07-15T17:00:00Z"},
			assert: func(t *testing.T, _ *ast.Term, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no active occurrence")
			},
		},
		{
			name: "midnight-crossing window closes the next day",
			// Tuesday 22:00 - Wednesday 02:00; at Wednesday 01:00 the active
			// occurrence closes Wednesday 02:00.
			args: []string{weekdays, "22:00", "02:00", "UTC", "2026-07-15T01:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-15T02:00:00Z"), term)
			},
		},
		{
			name: "close is rendered in UTC for non-UTC locations",
			// 16:00 UTC is 09:00 Los Angeles (PDT); the window closes 17:00 LA,
			// which is 00:00 UTC the following day.
			args: []string{weekdays, "09:00", "17:00", "America/Los_Angeles", "2026-07-15T16:00:00Z"},
			assert: func(t *testing.T, term *ast.Term, err error) {
				require.NoError(t, err)
				require.Equal(t, ast.StringTerm("2026-07-16T00:00:00Z"), term)
			},
		},
		{
			name: "malformed recurrence errors",
			args: []string{"FREQ=BOGUS", "09:00", "17:00", "UTC", "2026-07-15T15:00:00Z"},
			assert: func(t *testing.T, _ *ast.Term, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid recurrence")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			term, err := rruleActiveEnd(rego.BuiltinContext{}, strTerms(testCase.args...))
			testCase.assert(t, term, err)
		})
	}
}

func TestParseClock(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		in      string
		mins    int
		wantErr bool
	}{
		{in: "00:00", mins: 0},
		{in: "09:30", mins: 570},
		{in: "23:59", mins: 1439},
		{in: "24:00", wantErr: true},
		{in: "12:60", wantErr: true},
		{in: "noon", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.in, func(t *testing.T) {
			t.Parallel()
			mins, err := parseClock(testCase.in)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.mins, mins)
		})
	}
}
