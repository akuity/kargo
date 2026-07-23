# METADATA
# scope: package
# description: |
#   Ordering invariants for out-of-order dispatch, so the built-in decision
#   honors promotion class priority and Stage monotonicity — not only custom
#   policies. Forwards yield to a queued rollback; auto-forwards additionally
#   yield to a queued manual-forward for the same origin. An auto-forward
#   that does not advance the Stage is stale; a manual-forward that would
#   regress it is held. An auto-forward for an origin with a committed
#   auto-promotion hold is denied (the dispatch-side complement of the
#   controller's creation-side check). Per-Promotion scheduling (notBefore)
#   holds a promotion until its time.
#
#   The yield rules defer only to queued promotions that could actually run
#   in the candidate's place: ones BEHIND the candidate (the gate evaluates
#   candidates in queue order, so anything ahead was already evaluated
#   against the same snapshot and held — deferring to it would starve the
#   candidate on a promotion that cannot itself run) and ones that are DUE
#   (a promote-after schedule in the future would otherwise embargo the
#   queue until its time). The strictly-newer bound also makes the yield
#   relation acyclic: no two promotions can ever wait on each other.
#
#   Coalescing of shadowed auto-forwards is deliberately NOT done here: a
#   gate coalesce rule is a retirement decision, which is owned by grooming
#   (Freight-aware, maintains one live auto-forward per origin); the
#   regression rule below is the create-to-groom race-closer.
#
#   Reads data.queue (backlog in gate order, each entry carrying name, class,
#   createdAt, and — when resolvable — origin and notBefore),
#   data.autoPromotionHolds (committed per-origin holds), the kargo.lib
#   Freight-ordering helpers (advances / regresses / current_freight), and
#   input.promotion / input.now.
# schemas:
#   - input: schema.input
#   - data.queue: schema.queue
#   - data.autoPromotionHolds: schema.autopromotionholds
package kargo.lib.ordering

import rego.v1

# The Freight-ordering helpers (data.kargo.lib.advances / regresses /
# current_freight) live in kargo.lib and are shared with custom policies, so
# the gate and custom rules never disagree on "newer". They are referenced by
# their fully-qualified path: kargo.lib is an ancestor package, so an import
# would be flagged as pointless.

# The promotion classes that move Freight forward (as opposed to a rollback).
forward_classes := {"auto-forward", "manual-forward"}

# The annotation carrying a promotion's earliest dispatch time. The ergonomic
# first-class spec.notBefore is a later upgrade; the annotation keeps the
# scheduling rule working today.
promote_after_key := "kargo.akuity.io/promote-after"

# AX5 — yield to a queued rollback. A forward candidate (auto or manual)
# defers while a newer, due rollback is awaiting dispatch, so recovery
# preempts change. Promotion names encode creation order, so the strictly-
# greater name comparison is the behind-scoping described in the package doc.
# Deliberately NOT origin-scoped: a rollback is Stage-level recovery, so all
# forward motion defers to it (queue entries carry origin should this ever
# be revisited).
violation contains v if {
	input.promotion.class in forward_classes
	some q in data.queue
	q.class == "rollback"
	q.name > input.promotion.name
	due(q)
	v := {
		"rule": "yield-to-rollback",
		"msg": sprintf("yielding to queued rollback %q", [q.name]),
		"blocked_by": q.name,
		"requeue": 5,
	}
}

# AX5 — auto yields to manual. An auto-forward candidate defers while a
# newer, due manual-forward for the same origin is queued, regardless of the
# manual promote's Freight age: automation must not race ahead of an explicit
# human decision. Origin-scoped so a manual promote for one origin does not
# blanket-hold automation for unrelated origins. Manual-forwards are never
# subject to this rule.
violation contains v if {
	input.promotion.class == "auto-forward"
	some q in data.queue
	q.class == "manual-forward"
	q.name > input.promotion.name
	due(q)
	competes(q)
	v := {
		"rule": "yield-to-manual",
		"msg": sprintf("yielding to queued manual promotion %q", [q.name]),
		"blocked_by": q.name,
		"requeue": 5,
	}
}

# AX6 — auto anti-regression. An auto-forward is fungible "keep the newest
# deployed", so one that does not strictly advance the Stage is stale and must
# not run. The current_freight guard is required: without it the rule would
# vacuously fire on a fresh origin (nothing deployed yet). No requeue — cleared
# by grooming retiring the stale auto, not by the clock.
violation contains v if {
	input.promotion.class == "auto-forward"
	data.kargo.lib.current_freight
	not data.kargo.lib.advances
	v := {
		"rule": "regression",
		"msg": "auto-promotion would not advance the stage; stale",
	}
}

# AX6 — manual anti-regression. A manual-forward whose Freight is strictly
# older than the origin's current Freight would move the Stage backward: hold
# and surface it (re-issue as an explicit rollback if the regression is
# intended). A re-promote of the current Freight (equal, not older) is NOT held
# — kargo.regresses is strict. No requeue — cleared only by operator action.
violation contains v if {
	input.promotion.class == "manual-forward"
	data.kargo.lib.regresses
	v := {
		"rule": "would-regress",
		"msg": sprintf(
			"held: would regress the stage below current Freight %q; re-issue as a rollback if intended",
			[data.kargo.lib.current_freight.name],
		),
	}
}

# AX8 / create-race — respect the per-origin auto-promotion hold. An
# auto-forward for an origin with a committed hold is denied at the gate, the
# dispatch-side complement to the controller's creation-side check. This closes
# the create-race where a rival auto-forward was created before an operator
# promoted older Freight for the same origin.
#
# UNCONDITIONAL by design: unlike the freeze block there is no bypass hook. A
# hold is a controller-owned correctness interlock, not operator policy, so a
# custom policy may READ data.autoPromotionHolds but — since violation sets only
# union — can never suppress this deny. No requeue — cleared by operator resume.
violation contains v if {
	input.promotion.class == "auto-forward"
	data.autoPromotionHolds[input.freight.origin]
	v := {
		"rule": "auto-hold",
		"msg": sprintf("auto-promotion held for origin %q; awaiting resume", [input.freight.origin]),
	}
}

# AX3 — per-Promotion scheduling. Held until the promote-after time, then
# self-resuming via a requeue at that time.
violation contains v if {
	time.parse_rfc3339_ns(input.now) < time.parse_rfc3339_ns(not_before)
	v := {
		"rule": "scheduled",
		"msg": sprintf("scheduled; held until %s", [not_before]),
		"until": not_before,
		"requeue": (time.parse_rfc3339_ns(not_before) - time.parse_rfc3339_ns(input.now)) / 1000000000,
	}
}

not_before := input.promotion.annotations[promote_after_key]

# A queued promotion is due when it carries no promote-after schedule or its
# scheduled time has arrived. Guards the yield rules: a candidate never
# defers to a promotion that cannot itself dispatch yet — a scheduled
# promotion would otherwise embargo the whole queue until its time.
due(q) if not q.notBefore

due(q) if time.parse_rfc3339_ns(input.now) >= time.parse_rfc3339_ns(q.notBefore)

# A queued promotion competes with the candidate when it targets the same
# origin. Either origin may be absent (a queued promotion's Freight can be
# unresolvable; a candidate's Freight may be missing): the comparison then
# fails toward competing, so missing data can only over-yield, never miss a
# yield.
competes(q) if q.origin == input.freight.origin

competes(q) if not q.origin

competes(q) if not input.freight.origin
