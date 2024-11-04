package indexer

import (
	"context"
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SharedFieldIndexer is a wrapper around a client.FieldIndexer that ensures that
// the same field is not indexed multiple times.
//
// This is useful when multiple reconcilers require the same index, but want to
// continue to manage their own indices.
type SharedFieldIndexer struct {
	indexer client.FieldIndexer
	indices sync.Map
}

// NewSharedFieldIndexer returns a new SharedFieldIndexer.
func NewSharedFieldIndexer(indexer client.FieldIndexer) *SharedFieldIndexer {
	return &SharedFieldIndexer{
		indexer: indexer,
	}
}

// IndexField indexes the given field on the given object using the underlying
// client.FieldIndexer. If the field has already been indexed for the given
// object type, it will not be indexed again.
func (i *SharedFieldIndexer) IndexField(
	ctx context.Context,
	obj client.Object,
	field string,
	extractValue client.IndexerFunc,
) error {
	key := indexKey(obj, field)
	if _, loaded := i.indices.LoadOrStore(key, struct{}{}); loaded {
		return nil
	}

	if err := i.indexer.IndexField(ctx, obj, field, extractValue); err != nil {
		i.indices.Delete(key)
		return err
	}
	return nil
}

// indexKey returns a unique key for the given object and field.
func indexKey(obj client.Object, field string) string {
	return fmt.Sprintf("%T:%s", obj, field)
}
