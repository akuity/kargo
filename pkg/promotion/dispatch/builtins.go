package dispatch

import (
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
	"github.com/teambition/rrule-go"
)

// builtins returns the kargo.* custom Rego built-ins backed by rrule-go.
// Recurring windows are hard to express in native Rego (no RRULE support),
// so the recurrence math lives here and policy only decides what to do with
// it.
func builtins() []func(*rego.Rego) {
	return []func(*rego.Rego){
		rego.FunctionDyn(
			&rego.Function{
				Name: "kargo.rrule_active",
				Decl: types.NewFunction(
					types.Args(types.S, types.S, types.S, types.S, types.S),
					types.B,
				),
				Memoize: true,
			},
			rruleActive,
		),
		rego.Function4(
			&rego.Function{
				Name: "kargo.rrule_next",
				Decl: types.NewFunction(
					types.Args(types.S, types.S, types.S, types.S),
					types.S,
				),
				Memoize: true,
			},
			rruleNext,
		),
		rego.FunctionDyn(
			&rego.Function{
				Name: "kargo.rrule_active_end",
				Decl: types.NewFunction(
					types.Args(types.S, types.S, types.S, types.S, types.S),
					types.S,
				),
				Memoize: true,
			},
			rruleActiveEnd,
		),
	}
}

// rruleActive implements kargo.rrule_active(recurrence, start, end,
// location, now): whether now falls within an occurrence of the recurring
// window. An end at or before start closes on the following day.
func rruleActive(_ rego.BuiltinContext, operands []*ast.Term) (*ast.Term, error) {
	args, err := stringOperands(operands, 5)
	if err != nil {
		return nil, fmt.Errorf("kargo.rrule_active: %w", err)
	}
	rule, now, duration, err := parseWindow(args[0], args[1], args[2], args[3], args[4])
	if err != nil {
		return nil, fmt.Errorf("kargo.rrule_active: %w", err)
	}
	opening := rule.Before(now, true)
	if opening.IsZero() {
		return ast.BooleanTerm(false), nil
	}
	active := !now.Before(opening) && now.Before(opening.Add(duration))
	return ast.BooleanTerm(active), nil
}

// rruleNext implements kargo.rrule_next(recurrence, start, location, now):
// the RFC 3339 time at which the window next opens strictly after now.
func rruleNext(_ rego.BuiltinContext, a, b, c, d *ast.Term) (*ast.Term, error) {
	args, err := stringOperands([]*ast.Term{a, b, c, d}, 4)
	if err != nil {
		return nil, fmt.Errorf("kargo.rrule_next: %w", err)
	}
	rule, now, _, err := parseWindow(args[0], args[1], "", args[2], args[3])
	if err != nil {
		return nil, fmt.Errorf("kargo.rrule_next: %w", err)
	}
	next := rule.After(now, false)
	if next.IsZero() {
		return nil, fmt.Errorf(
			"kargo.rrule_next: recurrence %q has no occurrence after %q",
			args[0], args[3],
		)
	}
	return ast.StringTerm(next.UTC().Format(time.RFC3339)), nil
}

// rruleActiveEnd implements kargo.rrule_active_end(recurrence, start, end,
// location, now): the RFC 3339 time at which the occurrence currently active
// closes. It errors when now falls outside every occurrence, so callers guard
// with kargo.rrule_active. An end at or before start closes on the following
// day.
func rruleActiveEnd(_ rego.BuiltinContext, operands []*ast.Term) (*ast.Term, error) {
	args, err := stringOperands(operands, 5)
	if err != nil {
		return nil, fmt.Errorf("kargo.rrule_active_end: %w", err)
	}
	rule, now, duration, err := parseWindow(args[0], args[1], args[2], args[3], args[4])
	if err != nil {
		return nil, fmt.Errorf("kargo.rrule_active_end: %w", err)
	}
	opening := rule.Before(now, true)
	if opening.IsZero() || !now.Before(opening.Add(duration)) {
		return nil, fmt.Errorf(
			"kargo.rrule_active_end: no active occurrence at %q", args[4],
		)
	}
	return ast.StringTerm(opening.Add(duration).UTC().Format(time.RFC3339)), nil
}

// parseWindow builds an rrule whose occurrences open at the start wall-clock
// time in the given location, anchored well before now so that Before(now)
// finds the current or most recent opening. It returns the rule, now
// (localized), and the window duration (zero when end is empty).
func parseWindow(
	recurrence string,
	start string,
	end string,
	location string,
	nowStr string,
) (*rrule.RRule, time.Time, time.Duration, error) {
	loc := time.UTC
	if location != "" {
		var err error
		if loc, err = time.LoadLocation(location); err != nil {
			return nil, time.Time{}, 0, fmt.Errorf("invalid location %q: %w", location, err)
		}
	}
	now, err := time.Parse(time.RFC3339, nowStr)
	if err != nil {
		return nil, time.Time{}, 0, fmt.Errorf("invalid now %q: %w", nowStr, err)
	}
	now = now.In(loc)
	startMins, err := parseClock(start)
	if err != nil {
		return nil, time.Time{}, 0, fmt.Errorf("invalid start %q: %w", start, err)
	}
	var duration time.Duration
	if end != "" {
		endMins, endErr := parseClock(end)
		if endErr != nil {
			return nil, time.Time{}, 0, fmt.Errorf("invalid end %q: %w", end, endErr)
		}
		duration = time.Duration(endMins-startMins) * time.Minute
		if duration <= 0 {
			duration += 24 * time.Hour
		}
	}
	opt, err := rrule.StrToROption(recurrence)
	if err != nil {
		return nil, time.Time{}, 0, fmt.Errorf("invalid recurrence %q: %w", recurrence, err)
	}
	// Anchor the rule's start a year before now, at the window's opening
	// wall-clock time, so occurrences enumerate from before any time we will
	// be asked about.
	anchor := now.AddDate(-1, 0, 0)
	opt.Dtstart = time.Date(
		anchor.Year(), anchor.Month(), anchor.Day(),
		startMins/60, startMins%60, 0, 0,
		loc,
	)
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return nil, time.Time{}, 0, fmt.Errorf("invalid recurrence %q: %w", recurrence, err)
	}
	return rule, now, duration, nil
}

// parseClock parses "HH:MM" into minutes since midnight.
func parseClock(s string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, fmt.Errorf("expected HH:MM: %w", err)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("expected HH:MM, got %q", s)
	}
	return h*60 + m, nil
}

// stringOperands extracts n string operands from Rego terms.
func stringOperands(operands []*ast.Term, n int) ([]string, error) {
	if len(operands) != n {
		return nil, fmt.Errorf("expected %d operands, got %d", n, len(operands))
	}
	out := make([]string, n)
	for i, term := range operands {
		s, ok := term.Value.(ast.String)
		if !ok {
			return nil, fmt.Errorf("operand %d is not a string", i+1)
		}
		out[i] = string(s)
	}
	return out, nil
}
