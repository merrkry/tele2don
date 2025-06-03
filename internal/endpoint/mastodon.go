package endpoint

import (
	"context"
	"log/slog"

	"github.com/mattn/go-mastodon"
	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/state"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type MastodonEndpoint struct {
	id    state.EndpointID
	state *state.BridgeState

	client *mastodon.Client

	// userName string
}

func NewMastodonEndpoint(id state.EndpointID, state *state.BridgeState, cfg *mastodon.Config) (*MastodonEndpoint, error) {
	ep := &MastodonEndpoint{
		id:     id,
		state:  state,
		client: mastodon.NewClient(cfg),
	}

	// TODO: verify authentication

	return ep, nil
}

func (e *MastodonEndpoint) StartEndpoint(ctx context.Context, updateChan chan<- *bridge.BridgeMessage) error {
	// theoretically, we should use websocket
	// but it seems that go-mastodon doesn't implement it correctly
	eventChan, err := e.client.StreamingUser(ctx)
	if err != nil {
		slog.Error("Failed to start Mastodon streaming", "err", err)
		return err
	}

	// idk why go-mastodon doesn't use pointers
	go e.HandleUpdates(ctx, eventChan)

	return nil
}

func (e *MastodonEndpoint) HandleUpdates(ctx context.Context, eventChan <-chan mastodon.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventChan:
			slog.Debug("Received Mastodon event", "event", event)
			switch event := event.(type) {
			case *mastodon.UpdateEvent:
				if event.Status != nil {
					s := event.Status
					bid, err := e.state.QueryPlatformMessage(e.id, state.PlatformMessageID(s.ID))
					if err == state.ErrNotTracked {
						bm, err := e.convertStatus(s)
						if err != nil {
							slog.Error("Failed to convert Mastodon status to bridge message", "err", err)
							return
						}
						e.state.WriteBridgeMessage(bm)
						return
					} else if err != nil {
						slog.Error("Failed to query platform message", "err", err)
						return
					}

					slog.Error("Message edit not supported yet", "bid", bid)
				}
			case *mastodon.UpdateEditEvent:
				slog.Debug("Received Mastodon update edit event", "event", event)
			default: // unsupported event type
				continue
			}
		}
	}
}

func (e *MastodonEndpoint) convertStatus(status *mastodon.Status) (*bridge.BridgeMessage, error) {
	mdText, err := htmltomarkdown.ConvertString(status.Content)
	if err != nil {
		slog.Error("Failed to convert HTML to Markdown", "text", status.Content, "err", err)
		return nil, err
	}
	return &bridge.BridgeMessage{
		ID: e.state.NextID(),
		Content: &bridge.BridgeMessageContent{
			MDText: mdText,
		},
	}, nil
}
