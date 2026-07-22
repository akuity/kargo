# METADATA
# scope: package
# description: |
#   Rate-limits automatic promotion dispatch using a rolling window: at
#   most `max` dispatches within any trailing `window`. Manual promotions
#   and rollbacks are never rate-limited.
# schemas:
#   - input: schema.input
#   - data.rateLimit: schema.ratelimit
package kargo.lib.ratelimit

import rego.v1

violation contains v if {
	input.promotion.class == "auto-forward"
	rl := data.rateLimit[input.stage.name]
	count(recent) >= rl.max
	v := {
		"rule": "ratelimit",
		"msg": sprintf(
			"dispatch rate limit for stage %q is in effect (max %d per %s)",
			[input.stage.name, rl.max, format_ns(rl.window)],
		),
		"until": slot_free,
		"requeue": requeue_seconds,
	}
}

now_ns := time.parse_rfc3339_ns(input.now)

# The RFC3339 time the next dispatch slot frees (the oldest in-window
# dispatch aging out).
slot_free := time.format([min(recent) + data.rateLimit[input.stage.name].window, "UTC"])

# Dispatches still inside the rolling window. An array, not a set, so that
# same-second dispatches are not collapsed.
recent := [d |
	rl := data.rateLimit[input.stage.name]
	some d in rl.dispatches
	d > now_ns - rl.window
]

# The next dispatch slot frees when the oldest in-window dispatch ages out.
requeue_seconds := ((min(recent) + data.rateLimit[input.stage.name].window) - now_ns) / 1000000000

format_ns(ns) := sprintf("%ds", [round(ns / 1000000000)])
