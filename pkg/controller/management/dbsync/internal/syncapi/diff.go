package syncapi

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Diff compares a database snapshot with a complete, current Kubernetes list.
// Callers must load rows before calling Diff, using an uncached reader. Matching
// is resource-specific; missing rows and stale IDs are handled here. Neither
// the snapshot nor either data source is mutated.
func Diff[Object client.Object, Row any](
	ctx context.Context,
	reader client.Reader,
	list client.ObjectList,
	rows map[string]Row,
	matches func(Object, Row) (bool, error),
) (Changes, error) {
	if err := reader.List(ctx, list); err != nil {
		return Changes{}, fmt.Errorf("error listing objects for database resync: %w", err)
	}
	if list.GetContinue() != "" {
		return Changes{}, errors.New("incomplete Kubernetes list for database resync")
	}
	changes := Changes{}
	live := make(map[string]struct{})
	err := meta.EachListItem(list, func(item runtime.Object) error {
		obj, ok := item.(Object)
		if !ok {
			return fmt.Errorf("unexpected object %T in database resync list", item)
		}
		id := string(obj.GetUID())
		key := client.ObjectKeyFromObject(obj)
		if id == "" {
			return fmt.Errorf("object %q has no UID", key)
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
		changes.ToSync = append(changes.ToSync, key)
		return nil
	})
	if err != nil {
		return Changes{}, err
	}
	for id := range rows {
		if _, exists := live[id]; !exists {
			changes.ToDelete = append(changes.ToDelete, id)
		}
	}
	slices.Sort(changes.ToDelete)
	return changes, nil
}
