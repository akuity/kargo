# METADATA
# scope: package
# description: |
#   Building blocks for custom policies. The header prepended to custom
#   sources imports this package as `kargo`, so custom rules address them
#   as kargo.is_forward, kargo.is_semver_patch, etc. Contributes no
#   violations of its own.
# schemas:
#   - input: schema.input
package kargo.lib

import rego.v1

# is_forward is true when the promotion moves new Freight forward (as
# opposed to a rollback).
is_forward if input.promotion.class in {"auto-forward", "manual-forward"}

# is_semver_patch is true when new is a semver patch-only increment over
# old: same major.minor, strictly greater patch. A leading "v" is
# tolerated. The building block for hotfix semantics, which are otherwise
# a custom-policy concern -- e.g. a cluster policy might define:
#
#	exclusions_bypass(e) if is_hotfix
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
