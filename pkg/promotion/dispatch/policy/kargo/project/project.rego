# METADATA
# scope: package
# description: |
#   Extension points for a project-authored custom policy (ProjectConfig
#   spec.customPolicy). The engine prepends this package declaration and the
#   standard library imports to the custom source, so its rules land here.
#   This shipped module supplies the inert defaults consulted by the
#   standard library when a project defines nothing.
package kargo.project

import rego.v1

# exclusions_bypass(e) is consulted by kargo.lib.exclusions for each
# exclusion that would otherwise hold the promotion. A custom policy
# overrides it, e.g. with a self-defined hotfix rule (see
# kargo.is_semver_patch in ../lib/lib.rego):
#
#	exclusions_bypass(e) if is_hotfix
default exclusions_bypass(_) := false
