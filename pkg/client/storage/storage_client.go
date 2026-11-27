package storage

import (
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StorageClient implement interfaces to access storage, currently K8s resources, to be switched to DB.
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
