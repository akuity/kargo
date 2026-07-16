# METADATA
# scope: package
# description: |
#   Extension points for an operator-authored custom policy (ClusterConfig
#   spec.customPolicy), composed into every project's dispatch decision.
#   The engine prepends this package declaration and the standard library
#   imports to the custom source, so its rules land here. This shipped
#   module supplies the inert defaults consulted by the standard library
#   when the cluster defines nothing.
package kargo.cluster

import rego.v1

# exclusions_bypass(e) is consulted by kargo.lib.exclusions for each
# exclusion that would otherwise hold the promotion; see kargo.project.
default exclusions_bypass(_) := false
