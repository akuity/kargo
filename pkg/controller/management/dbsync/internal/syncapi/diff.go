package syncapi

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Diff compares a database snapshot with a Kubernetes list and returns the
// keys that disagree. Callers must load rows before calling Diff. The list is
// never paginated: no limit is passed, so the API server returns every object,
// and the continue token the cache reader stamps on its lists carries no
// meaning and is ignored. The reader may be cached, since every returned key
// is re-read live before it is acted on. Matching is resource-specific;
// missing rows and orphaned rows are handled here. Live objects come first,
// in list order, followed by the keys of orphaned rows sorted by name, with
// no duplicates. A kind whose rows outlive their objects passes a nil key:
// orphaned rows are then not orphans at all and are left out. Neither the
// snapshot nor either data source is mutated.
func Diff[Object client.Object, Row any](
	ctx context.Context,
	reader client.Reader,
	list client.ObjectList,
	rows map[string]Row,
	matches func(Object, Row) (bool, error),
	key func(Row) client.ObjectKey,
) ([]reconcile.Request, error) {
	if err := reader.List(ctx, list); err != nil {
		return nil, fmt.Errorf("error listing objects for database resync: %w", err)
	}
	var requests []reconcile.Request
	seen := make(map[client.ObjectKey]struct{})
	live := make(map[string]struct{})
	err := meta.EachListItem(list, func(item runtime.Object) error {
		obj, ok := item.(Object)
		if !ok {
			return fmt.Errorf("unexpected object %T in database resync list", item)
		}
		id := string(obj.GetUID())
		objKey := client.ObjectKeyFromObject(obj)
		if id == "" {
			return fmt.Errorf("object %q has no UID", objKey)
		}
		live[id] = struct{}{}
		row, exists := rows[id]
		if exists {
			match, matchErr := matches(obj, row)
			if matchErr != nil {
				return matchErr
			}
			if match {
				return nil
			}
		}
		seen[objKey] = struct{}{}
		requests = append(requests, reconcile.Request{NamespacedName: objKey})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if key == nil {
		return requests, nil
	}
	var orphaned []reconcile.Request
	for id, row := range rows {
		if _, exists := live[id]; exists {
			continue
		}
		rowKey := key(row)
		if _, dup := seen[rowKey]; dup {
			continue
		}
		seen[rowKey] = struct{}{}
		orphaned = append(orphaned, reconcile.Request{NamespacedName: rowKey})
	}
	slices.SortFunc(orphaned, func(a, b reconcile.Request) int {
		return strings.Compare(a.String(), b.String())
	})
	return append(requests, orphaned...), nil
}
