# METADATA
# scope: package
# description: |
#   Building blocks for custom policies. The header prepended to custom
#   sources imports this package as `kargo`, so custom rules address them
#   as kargo.is_forward, kargo.is_semver_patch, etc. Contributes no
#   violations of its own.
# schemas:
#   - input: schema.input
#   - data.currentFreight: schema.currentFreight
package kargo.lib

import rego.v1

# is_forward is true when the promotion moves new Freight forward (as
# opposed to a rollback).
is_forward if input.promotion.class in {"auto-forward", "manual-forward"}

# freight_newer(a, b) is true when Freight a was discovered strictly after
# Freight b. It mirrors the Stage controller's auto-promotion ordering key
# (EffectiveDiscoveredAt, then name as a deterministic tiebreak), so the gate
# and auto-selection never disagree on "newer". Undefined -- hence false --
# when either discoveredAt is absent, so callers need no presence guard.
freight_newer(a, b) if time.parse_rfc3339_ns(a.discoveredAt) > time.parse_rfc3339_ns(b.discoveredAt)

freight_newer(a, b) if {
	a.discoveredAt == b.discoveredAt
	a.name > b.name
}

# current_freight is the Stage's current Freight for the candidate's origin,
# as {name, discoveredAt}. Undefined on a fresh origin (nothing deployed yet)
# or when the current Freight could not be resolved, in which case advances
# and regresses are both undefined (false).
current_freight := data.currentFreight[input.freight.origin]

# advances is true when the candidate Freight is strictly newer than the
# current Freight of its origin -- the promotion moves the Stage forward.
advances if freight_newer(input.freight, current_freight)

# regresses is true when the candidate Freight is strictly older than the
# current Freight of its origin -- the promotion would move the Stage
# backward. A re-promote of the current Freight is neither (freight_newer is
# strict), so it is not a regression.
regresses if freight_newer(current_freight, input.freight)

# is_semver_patch is true when new is a semver patch-only increment over
# old: same major.minor, strictly greater patch. A leading "v" is
# tolerated. The building block for hotfix semantics, which are otherwise
# a custom-policy concern -- e.g. a cluster policy might define:
#
#	freeze_bypass(f) if is_hotfix
#
#	is_hotfix if {
#		count(shared_images) > 0
#		every pair in shared_images {
#			kargo.is_semver_patch(pair.old, pair.new)
#		}
#	}
#
#	shared_images := [pair |
#		some img in input.freight.images
#		some last in input.stage.lastPromotion.freight.images
#		img.repoURL == last.repoURL
#		pair := {"old": last.tag, "new": img.tag}
#	]
is_semver_patch(old, new) if {
	o := trim_prefix(old, "v")
	semver.is_valid(o)
	n := trim_prefix(new, "v")
	semver.is_valid(n)
	semver.compare(n, o) == 1
	split(o, ".")[0] == split(n, ".")[0]
	split(o, ".")[1] == split(n, ".")[1]
}
