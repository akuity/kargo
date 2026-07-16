# METADATA
# scope: package
# description: |
#   Holds promotions during system-wide exclusion (blackout) periods, scoped
#   by promotion class and, optionally, by the Argo CD servers targeted by
#   the Stage's referenced Applications.
#
#   A custom project policy (package kargo.custom) may bypass individual
#   exclusions by contributing their names to an exclusions_bypass set:
#
#     exclusions_bypass contains e.name if {
#         some e in data.exclusions
#         helpers.is_hotfix
#     }
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

# An exclusion is bypassed when the project's custom module names it in
# exclusions_bypass. Inert when the project has no custom module.
bypassed(e) if e.name in data.kargo.custom.exclusions_bypass
