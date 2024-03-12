package warehouses

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (r *reconciler) getAvailableFreightAlias(
	ctx context.Context,
) (string, error) {
	for {
		alias := r.freightAliasGenerator.NameSep("-")
		freight := kargoapi.FreightList{}
		if err := r.client.List(
			ctx,
			&freight,
			client.MatchingLabels{kargoapi.AliasLabelKey: alias},
		); err != nil {
			return "", fmt.Errorf(
				"error checking for existence of Freight with alias %q: %w",
				alias,
				err,
			)
		}
		if len(freight.Items) == 0 {
			return alias, nil
		}
	}
}
