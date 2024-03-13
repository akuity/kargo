package freight

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (w *webhook) getAvailableFreightAlias(
	ctx context.Context,
) (string, error) {
	for {
		alias := w.freightAliasGenerator.NameSep("-")
		freight := kargoapi.FreightList{}
		if err := w.client.List(
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
