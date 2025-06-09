package service

import (
	"context"
	"sync"
	"time"

	"github.com/merrkry/tele2don/internal/endpoint"
	"github.com/merrkry/tele2don/internal/model"
)

type Endpoint interface {
	ID() model.EndpointID

	// Initialize validates the configuration, and initializes platform-specific APIs.
	Initialize(ctx context.Context, cfg *endpoint.EndpointConfig) error

	// ListenUpdates starts endpoint worker to listen for platform updates.
	// Endpoint should convert platform-specific updates to model.EndpointUpdate.
	ListenUpdates(ctx context.Context, updatesChan chan<- *model.EndpointUpdate, wg *sync.WaitGroup)

	// ApplyUpdate sends new message to the endpoint, and returns the timestamp responded by platform API.
	ApplyUpdateNew(ctx context.Context, content *model.BridgeMessageContent) (model.EndpointMessageID, time.Time, error)

	// ApplyUpdateEdit applies message deletion to the endpoint, and returns the timestamp responded by platform API.
	ApplyUpdateEdit(ctx context.Context, id model.EndpointMessageID, content *model.BridgeMessageContent) (time.Time, error)

	// ApplyUpdateDelete applies message deletion to the endpoint, and returns the timestamp responded by platform API.
	ApplyUpdateDelete(ctx context.Context, id model.EndpointMessageID) error
}
