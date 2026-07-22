# METADATA
# scope: package
# description: |
#   The promotion dispatch policy. Composes the standard library blocks by
#   unioning their violations, gathers any violations contributed by the
#   project's custom module (kargo.project, from ProjectConfig
#   spec.customPolicy) and the cluster's (kargo.cluster, from ClusterConfig
#   spec.customPolicy), and derives a single decision document from the
#   result.
# schemas:
#   - input: schema.input
# entrypoint: true
package kargo.dispatch

import rego.v1

import data.kargo.lib.freezes
import data.kargo.lib.ordering
import data.kargo.lib.ratelimit
import data.kargo.lib.windows

violation contains v if some v in windows.violation

violation contains v if some v in freezes.violation

violation contains v if some v in ordering.violation

violation contains v if some v in ratelimit.violation

# Custom violations compose into the same set — inert when no custom
# policy defines any. Each violation is an object {rule, msg, requeue?}; a
# numeric requeue (seconds) feeds requeue_after below.
violation contains v if some v in data.kargo.project.violation

violation contains v if some v in data.kargo.cluster.violation

decision := {"allow": true, "message": "within policy", "requeue_after": 0} if {
	count(violation) == 0
}

decision := {
	"allow": false,
	"message": concat("; ", [v.msg | some v in violation]),
	"requeue_after": requeue,
} if {
	count(violation) > 0
}

# The soonest time at which any violation may clear. A violation that
# carries no numeric requeue simply does not contribute.
requeues := [v.requeue | some v in violation; is_number(v.requeue)]

requeue := min(requeues) if count(requeues) > 0

requeue := 0 if count(requeues) == 0
