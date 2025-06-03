package endpoint

import (
	"context"

	"github.com/merrkry/tele2don/internal/bridge"
)

type Endpoint interface {
	// ApplyUpdate create/updates a platform message based on the provided bridge message
	// ApplyUpdate(update *bridge.BridgeMessage) (Brid, error)
	// StartEndpoint starts the endpoint and pushes updates to the provided channel
	StartEndpoint(ctx context.Context, updateChan chan<- *bridge.BridgeMessage) error
}
