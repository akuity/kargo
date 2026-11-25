package dataplane

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/event"
)

// StorageClient implement Dataplane client interfaces on controlplane.
// It backs Controlplane components and API Server.

type StorageClient struct {
	client      client.Client
	eventSender event.Sender
	db          *database.Queries
}

func NewStorageClient(
	cl client.Client,
	eventSender event.Sender,
	db *database.Queries,
) *StorageClient {
	return &StorageClient{
		cl,
		eventSender,
		db,
	}
}
