# METADATA
# scope: package
# description: |
#   Holds forward promotions outside promotion windows. When one or more
#   windows in data.windows govern the Stage, a forward promotion (class
#   auto-forward or manual-forward) may only be dispatched while at least
#   one window is open. Rollbacks are never held by windows.
# schemas:
#   - input: schema.input
#   - data.windows: schema.windows
package kargo.lib.windows

import rego.v1

forward_classes := {"auto-forward", "manual-forward"}

violation contains v if {
	input.promotion.class in forward_classes
	count(data.windows) > 0
	not in_any_window
	v := {
		"rule": "windows",
		"msg": sprintf(
			"outside all promotion windows; next window opens at %s",
			[next_open],
		),
		"until": next_open,
		"requeue": requeue_seconds,
	}
}

in_any_window if {
	some w in data.windows
	kargo.rrule_active(w.recurrence, w.start, w.end, w.location, input.now)
}

next_open := min({t |
	some w in data.windows
	t := kargo.rrule_next(w.recurrence, w.start, w.location, input.now)
})

requeue_seconds := (time.parse_rfc3339_ns(next_open) - time.parse_rfc3339_ns(input.now)) / 1000000000
