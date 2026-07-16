# METADATA
# scope: package
# description: |
#   Holds promotions during system-wide exclusion (blackout) periods, scoped
#   by promotion class and, optionally, by the Argo CD servers targeted by
#   the Stage's referenced Applications.
#
#   A custom policy (project- or cluster-scoped) may bypass exclusions by
#   overriding the exclusions_bypass predicate, e.g. with a self-defined
#   hotfix rule (see kargo.lib.helpers for the semver building block):
#
#     exclusions_bypass(e) if is_hotfix
# schemas:
#   - input: schema.input
#   - data.exclusions: schema.exclusions
#   - data.scopes: schema.scopes
package kargo.lib.exclusions

import rego.v1

violation contains v if {
	some e in data.exclusions
	active(e)
	input.promotion.class in data.scopes[e.scope]
	applies_to_servers(e)
	not bypassed(e)
	v := {
		"rule": "exclusions",
		"msg": sprintf(
			"promotion class %q is frozen by exclusion %q (scope %q) until %s",
			[input.promotion.class, e.name, e.scope, e.end],
		),
		"requeue": (time.parse_rfc3339_ns(e.end) - time.parse_rfc3339_ns(input.now)) / 1000000000,
	}
}

active(e) if {
	time.parse_rfc3339_ns(e.start) <= time.parse_rfc3339_ns(input.now)
	time.parse_rfc3339_ns(input.now) < time.parse_rfc3339_ns(e.end)
}

# An exclusion with no server scoping applies to every Stage; one with
# server scoping applies only when a referenced Application targets one of
# the named destination servers (by URL or name).
applies_to_servers(e) if not e.argocdServers

applies_to_servers(e) if count(e.argocdServers) == 0

applies_to_servers(e) if {
	some s in e.argocdServers
	some app in input.applications
	s in {app.destination.server, app.destination.name}
}

# An exclusion is bypassed when the project's or the cluster's custom
# policy says so. Both predicates default to false (see kargo.project and
# kargo.cluster), so this is inert without custom content.
bypassed(e) if data.kargo.project.exclusions_bypass(e)

bypassed(e) if data.kargo.cluster.exclusions_bypass(e)
