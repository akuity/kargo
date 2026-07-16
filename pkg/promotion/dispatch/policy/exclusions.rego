# kargo.lib.exclusions holds promotions during system-wide exclusion
# (blackout) periods, scoped by promotion class and, optionally, by the
# Argo CD servers targeted by the Stage's referenced Applications.
#
# This block deliberately has no bypass logic. A custom project policy that
# wants one (e.g. for hotfixes) filters this block's violations itself:
#
#	violation contains v if {
#		some v in exclusions.violation
#		not helpers.is_hotfix
#	}
#
# data.exclusions: [{name, start (RFC3339), end (RFC3339), scope,
# argocdServers []}]
# data.scopes: {scope: [frozen classes]}
package kargo.lib.exclusions

import rego.v1

violation contains v if {
	some e in data.exclusions
	active(e)
	input.promotion.class in data.scopes[e.scope]
	applies_to_servers(e)
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
