package indexer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// mockObject implements client.Object for testing
type mockObject struct {
	client.Object
}

func (m *mockObject) GetName() string      { return "mock" }
func (m *mockObject) GetNamespace() string { return "default" }

// mockFieldIndexer implements client.FieldIndexer for testing.
type mockFieldIndexer struct {
	called int
	err    error
}

func (m *mockFieldIndexer) IndexField(
	context.Context,
	client.Object,
	string,
	client.IndexerFunc,
) error {
	m.called++
	if m.err != nil {
		return m.err
	}
	return nil
}

func TestIndexField(t *testing.T) {
	tests := []struct {
		name          string
		internalIndex *mockFieldIndexer
		key           string
		calls         int
		assertions    func(t *testing.T, index *SharedFieldIndexer, key string, err error)
	}{
		{
			name:          "successful first indexing",
			internalIndex: &mockFieldIndexer{},
			key:           "metadata.name",
			calls:         1,
			assertions: func(t *testing.T, index *SharedFieldIndexer, key string, err error) {
				require.NoError(t, err)

				internal, ok := index.indexer.(*mockFieldIndexer)
				require.True(t, ok)

				assert.Equal(t, 1, internal.called)
				_, ok = index.indices.Load(indexKey(&mockObject{}, key))
				assert.True(t, ok)
			},
		},
		{
			name:          "duplicate indexing attempts",
			internalIndex: &mockFieldIndexer{},
			key:           "metadata.name",
			calls:         2,
			assertions: func(t *testing.T, index *SharedFieldIndexer, key string, err error) {
				require.NoError(t, err)

				internal, ok := index.indexer.(*mockFieldIndexer)
				require.True(t, ok)

				assert.Equal(t, 1, internal.called)
				_, ok = index.indices.Load(indexKey(&mockObject{}, key))
				assert.True(t, ok)
			},
		},
		{
			name: "indexing error",
			internalIndex: &mockFieldIndexer{
				err: errors.New("something went wrong"),
			},
			key:   "metadata.name",
			calls: 1,
			assertions: func(t *testing.T, index *SharedFieldIndexer, key string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "something went wrong")

				internal, ok := index.indexer.(*mockFieldIndexer)
				require.True(t, ok)

				assert.Equal(t, 1, internal.called)
				_, ok = index.indices.Load(indexKey(&mockObject{}, key))
				assert.False(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := NewSharedFieldIndexer(tt.internalIndex)
			err := index.IndexField(context.Background(), &mockObject{}, "metadata.name", nil)
			tt.assertions(t, index, tt.key, err)
		})
	}
}
