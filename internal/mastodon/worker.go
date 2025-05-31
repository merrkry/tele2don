package mastodon

import (
	"context"
	"log/slog"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/mattn/go-mastodon"
	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/config"
)

type mastodonPlatform struct {
	client *mastodon.Client
}

func (p *mastodonPlatform) Name() bridge.PlatformName {
	return bridge.PlatformTelegram
}

func NewTelegramPlatform(ctx context.Context, cfg *config.Tele2donConfig, bridgeChan chan bridge.BridgeUpdate) (*mastodonPlatform, error) {
	// Ideally, we should use user credentials generated from web for easier setup.
	// However, mastodon only accept "App token" on very few endpoints.
	// We therefore must authenticate via tele2don-setup to retrieve "User token".

	mastodonCfg := &mastodon.Config{
		Server:       cfg.MastodonServer,
		ClientID:     cfg.MastodonClientID,
		ClientSecret: cfg.MastodonClientSecret,
		AccessToken:  cfg.MastodonAccessToken,
	}

	client := mastodon.NewClient(mastodonCfg)

	// It seems that websocket implementation is not using Authorization header,
	// but access_token parameter, "websocket: bad handshake" will be returned
	// eventsChan, err := client.NewWSClient().StreamingWSUser(ctx)
	eventsChan, err := client.StreamingUser(ctx)
	if err != nil {
		slog.Error("Failed to create Mastodon streaming client.", "err", err)
		return nil, err
	}

	go handleStreaming(ctx, cfg, eventsChan, bridgeChan)

	return &mastodonPlatform{
		client: client,
	}, nil
}

func handleStreaming(ctx context.Context, cfg *config.Tele2donConfig, eventsChan chan mastodon.Event, bridgeChan chan bridge.BridgeUpdate) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context done, stopping Mastodon streaming.")
			return
		case event := <-eventsChan:
			if event == nil {
				continue
			}
			slog.Debug("Received event from Mastodon", "event", event)
			switch e := event.(type) {
			case *mastodon.UpdateEvent:
				if s := e.Status; s != nil {
					if s.Content != "" {
						slog.Debug("Converting message from Mastodon.", "text", s.Content, "id", s.ID)
						mdText, err := htmltomarkdown.ConvertString(s.Content)
						if err != nil {
							slog.Error("Failed to convert HTML to Markdown", "err", err, "content", s.Content)
							continue
						}
						bridgeChan <- bridge.NewMessage{
							Text: mdText,
						}
					}
				}
			case *mastodon.UpdateEditEvent:
			}
		}
	}
}

func (p *mastodonPlatform) ApplyUpdate(update bridge.BridgeUpdate) error {
	return nil
}
