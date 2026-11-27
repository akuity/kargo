package storage

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/event"
)

// Client implement interfaces to access storage, currently K8s resources, to be switched to DB.
// It backs Controlplane components and API Server.
type Client struct {
	client      client.Client
	eventSender event.Sender
	db          *database.Queries
}

func NewStorageClient(
	cl client.Client,
	eventSender event.Sender,
	db *database.Queries,
) *Client {
	return &Client{
		cl,
		eventSender,
		db,
	}
}
