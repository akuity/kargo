package telemetry

import (
	"go.opentelemetry.io/otel/attribute"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FreightDetailAttributes returns a Freight's origin, its Warehouse if that is
// the origin, and its commit ID if it has exactly one.
func FreightDetailAttributes(f *kargoapi.Freight) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if f.Origin.Name != "" {
		attrs = append(attrs, FreightOriginKey.String(string(f.Origin.Kind)+"/"+f.Origin.Name))
		if f.Origin.Kind == kargoapi.FreightOriginKindWarehouse {
			attrs = append(attrs, WarehouseKey.String(f.Origin.Name))
		}
	}
	if len(f.Commits) == 1 {
		attrs = append(attrs, VCSCommitKey.String(f.Commits[0].ID))
	}
	return attrs
}
