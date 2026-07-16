// Package dispatch decides whether a Pending Promotion may be dispatched
// (acknowledged by its Stage and allowed to begin Running). The decision is
// delegated to an OPA policy: a built-in default composed of standard,
// data-driven library blocks (see policy/), into which a project-authored
// custom policy (ProjectConfig spec.customPolicy) and an operator-authored
// one (ClusterConfig spec.customPolicy) may compose additional violations
// and exclusion bypasses.
package dispatch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
)

// decisionQuery is the entry point every dispatch policy must produce.
const decisionQuery = "data.kargo.dispatch.decision"

// Decision is a dispatch policy verdict.
type Decision struct {
	// Allow indicates whether the Promotion may be dispatched.
	Allow bool
	// Message enumerates the reasons the Promotion is held. Empty or
	// informational when Allow is true.
	Message string
	// RequeueAfter is the soonest time at which a denial may clear — the
	// "when" that lets a held Promotion resume on its own. Zero when the
	// policy offered none.
	RequeueAfter time.Duration
}

// Engine evaluates dispatch policies, caching one prepared query per
// distinct pair of custom policy sources (including compile failures, so a
// broken policy is not recompiled on every reconcile).
type Engine struct {
	mu       sync.Mutex
	prepared map[string]*preparedPolicy
}

// NewEngine returns a new Engine.
func NewEngine() *Engine {
	return &Engine{prepared: map[string]*preparedPolicy{}}
}

// Evaluate evaluates the dispatch policy against the given input and data
// documents. projectCustom and clusterCustom are rules-only custom policy
// sources (ProjectConfig and ClusterConfig spec.customPolicy respectively);
// either or both may be empty, in which case the built-in default policy
// behavior applies.
func (e *Engine) Evaluate(
	ctx context.Context,
	projectCustom string,
	clusterCustom string,
	input map[string]any,
	data map[string]any,
) (*Decision, error) {
	pp := e.policyFor(projectCustom, clusterCustom)
	pp.once.Do(func() { pp.prepare(ctx, projectCustom, clusterCustom) })
	if pp.err != nil {
		return nil, fmt.Errorf("error preparing dispatch policy: %w", pp.err)
	}
	return pp.eval(ctx, input, data)
}

func (e *Engine) policyFor(projectCustom, clusterCustom string) *preparedPolicy {
	sum := sha256.Sum256([]byte(projectCustom + "\x00" + clusterCustom))
	key := hex.EncodeToString(sum[:])
	e.mu.Lock()
	defer e.mu.Unlock()
	pp, ok := e.prepared[key]
	if !ok {
		pp = &preparedPolicy{}
		e.prepared[key] = pp
	}
	return pp
}

// preparedPolicy is a compiled dispatch policy. The store is bound at
// preparation time, so per-evaluation data is supplied through a write
// transaction guarded by mu (the inmem store is single-writer).
type preparedPolicy struct {
	once  sync.Once
	err   error
	mu    sync.Mutex
	query rego.PreparedEvalQuery
	store storage.Store
}

func (p *preparedPolicy) prepare(ctx context.Context, projectCustom, clusterCustom string) {
	mods, err := policyModules(projectCustom, clusterCustom)
	if err != nil {
		p.err = err
		return
	}
	schemas, err := policySchemas()
	if err != nil {
		p.err = err
		return
	}
	modOpts, err := moduleOptions(mods)
	if err != nil {
		p.err = err
		return
	}
	p.store = inmem.New()
	opts := []func(*rego.Rego){
		rego.Query(decisionQuery),
		rego.Store(p.store),
		rego.StrictBuiltinErrors(true),
		rego.Schemas(schemas),
	}
	opts = append(opts, modOpts...)
	opts = append(opts, builtins()...)
	p.query, p.err = rego.New(opts...).PrepareForEval(ctx)
}

// moduleOptions parses modules with annotation processing enabled — the
// rego package's own parsing (rego.Module) skips annotations, which would
// silently disable schema type checking — and returns them as rego options.
func moduleOptions(mods map[string]string) ([]func(*rego.Rego), error) {
	opts := make([]func(*rego.Rego), 0, len(mods))
	for name, src := range mods {
		mod, err := ast.ParseModuleWithOpts(
			name,
			src,
			ast.ParserOptions{ProcessAnnotation: true},
		)
		if err != nil {
			return nil, fmt.Errorf("error parsing policy module %q: %w", name, err)
		}
		opts = append(opts, rego.ParsedModule(mod))
	}
	return opts, nil
}

func (p *preparedPolicy) eval(
	ctx context.Context,
	input map[string]any,
	data map[string]any,
) (*Decision, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	txn, err := p.store.NewTransaction(ctx, storage.WriteParams)
	if err != nil {
		return nil, fmt.Errorf("error starting policy data transaction: %w", err)
	}
	defer p.store.Abort(ctx, txn)
	if err = p.store.Write(ctx, txn, storage.AddOp, storage.Path{}, data); err != nil {
		return nil, fmt.Errorf("error writing policy data: %w", err)
	}
	rs, err := p.query.Eval(ctx, rego.EvalInput(input), rego.EvalTransaction(txn))
	if err != nil {
		return nil, fmt.Errorf("error evaluating dispatch policy: %w", err)
	}
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return nil, fmt.Errorf("dispatch policy did not produce a decision")
	}
	return decodeDecision(rs[0].Expressions[0].Value)
}

func decodeDecision(v any) (*Decision, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decision is %T, expected an object", v)
	}
	allow, ok := m["allow"].(bool)
	if !ok {
		return nil, fmt.Errorf("decision has no boolean \"allow\" key")
	}
	decision := &Decision{Allow: allow}
	if msg, ok := m["message"].(string); ok {
		decision.Message = msg
	}
	if raw, ok := m["requeue_after"]; ok {
		secs, err := toFloat(raw)
		if err != nil {
			return nil, fmt.Errorf("decision has invalid \"requeue_after\": %w", err)
		}
		if secs > 0 {
			decision.RequeueAfter = time.Duration(secs * float64(time.Second))
		}
	}
	return decision, nil
}

func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case json.Number:
		return n.Float64()
	case float64:
		return n, nil
	case int64:
		return float64(n), nil
	case int:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("value is %T, expected a number", v)
	}
}
