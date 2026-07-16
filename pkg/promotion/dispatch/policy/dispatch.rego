# kargo.dispatch is the default promotion dispatch policy. It composes the
# standard library blocks by unioning their violations and derives a single
# decision document from the result.
#
# A project replaces this module with its own (ProjectConfig
# spec.policy.custom), which must belong to the same package and produce the
# same decision document. The standard blocks remain importable there.
package kargo.dispatch

import rego.v1

import data.kargo.lib.exclusions
import data.kargo.lib.ratelimit
import data.kargo.lib.windows

violation contains v if some v in windows.violation

violation contains v if some v in exclusions.violation

violation contains v if some v in ratelimit.violation

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
