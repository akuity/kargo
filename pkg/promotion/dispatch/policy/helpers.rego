# kargo.lib.helpers offers building blocks for custom project policies. It
# contributes no violations of its own.
package kargo.lib.helpers

import rego.v1

# forward is true when the promotion moves new Freight forward (as opposed
# to a rollback).
forward if input.promotion.class in {"auto-forward", "manual-forward"}

# is_hotfix is true when the candidate Freight is a patch-only increment
# over the Freight the Stage last promoted: at least one image repository is
# shared with the last promoted Freight, and every shared repository's tag
# has the same major.minor and a strictly greater patch. Typically used by
# custom policies to let hotfixes bypass exclusions:
#
#	violation contains v if {
#		some v in exclusions.violation
#		not helpers.is_hotfix
#	}
is_hotfix if {
	count(shared_images) > 0
	every pair in shared_images {
		patch_increment(pair.old, pair.new)
	}
}

shared_images := [pair |
	some img in input.freight.images
	some last in input.stage.lastPromotion.freight.images
	img.repoURL == last.repoURL
	pair := {"old": last.tag, "new": img.tag}
]

patch_increment(old, new) if {
	o := trim_prefix(old, "v")
	n := trim_prefix(new, "v")
	semver.is_valid(o)
	semver.is_valid(n)
	semver.compare(n, o) == 1
	split(o, ".")[0] == split(n, ".")[0]
	split(o, ".")[1] == split(n, ".")[1]
}
