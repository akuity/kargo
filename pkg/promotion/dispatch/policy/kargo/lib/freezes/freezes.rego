# METADATA
# scope: package
# description: |
#   Holds promotions during system-wide freeze (blackout) periods, scoped
#   by promotion class and, optionally, by the Argo CD servers targeted by
#   the Stage's referenced Applications.
#
#   A custom policy (project- or cluster-scoped) may bypass freezes by
#   overriding the freeze_bypass predicate, e.g. with a self-defined
#   hotfix rule (see kargo.is_semver_patch for the semver building block):
#
#     freeze_bypass(f) if is_hotfix
# schemas:
#   - input: schema.input
#   - data.freezes: schema.freezes
#   - data.scopes: schema.scopes
package kargo.lib.freezes

import rego.v1

violation contains v if {
	some f in data.freezes
	active(f)
	input.promotion.class in data.scopes[f.scope]
	applies_to_servers(f)
	not bypassed(f)
	v := {
		"rule": "freezes",
		"msg": sprintf(
			"promotion class %q is frozen by freeze %q (scope %q) until %s",
			[input.promotion.class, f.name, f.scope, f.end],
		),
		"requeue": (time.parse_rfc3339_ns(f.end) - time.parse_rfc3339_ns(input.now)) / 1000000000,
	}
}

active(f) if {
	time.parse_rfc3339_ns(f.start) <= time.parse_rfc3339_ns(input.now)
	time.parse_rfc3339_ns(input.now) < time.parse_rfc3339_ns(f.end)
}

# A freeze with no server scoping applies to every Stage; one with
# server scoping applies only when a referenced Application targets one of
# the named destination servers (by URL or name).
applies_to_servers(f) if not f.argocdServers

applies_to_servers(f) if count(f.argocdServers) == 0

applies_to_servers(f) if {
	some s in f.argocdServers
	some app in input.applications
	s in {app.destination.server, app.destination.name}
}

# A freeze is bypassed when the project's or the cluster's custom
# policy says so. Both predicates default to false (see kargo.project and
# kargo.cluster), so this is inert without custom content.
bypassed(f) if data.kargo.project.freeze_bypass(f)

bypassed(f) if data.kargo.cluster.freeze_bypass(f)
