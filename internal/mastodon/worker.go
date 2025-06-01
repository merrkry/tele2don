package mastodon

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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

	p := &mastodonPlatform{
		client: client,
	}

	go handleStreaming(ctx, cfg, eventsChan, bridgeChan, p)

	return p, nil
}

func handleStreaming(ctx context.Context, cfg *config.Tele2donConfig, eventsChan chan mastodon.Event, bridgeChan chan bridge.BridgeUpdate, platform *mastodonPlatform) {
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
							Text:       mdText,
							Origin:     platform,
							Identifier: bridge.MessageIdentifier("m:" + s.ID),
						}
					}
				}
			case *mastodon.UpdateEditEvent:
			}
		}
	}
}

func (p *mastodonPlatform) ApplyUpdate(cfg *config.Tele2donConfig, update bridge.BridgeUpdate) (bridge.MessageIdentifier, error) {
	if update == nil {
		return bridge.NilMessageIdentifier, fmt.Errorf("update is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	switch u := update.(type) {
	case bridge.NewMessage:
		var convertedText string = u.Text
		status, err := p.client.PostStatus(ctx, &mastodon.Toot{
			Status:     convertedText,
			Visibility: "public",
		})
		if err != nil {
			slog.Error("Failed to post status to Mastodon.", "err", err, "text", u.Text)
			return bridge.NilMessageIdentifier, err
		}
		return bridge.MessageIdentifier("m:" + status.ID), nil
	default:
		return bridge.NilMessageIdentifier, fmt.Errorf("unsupported update type: %T", u)
	}
}
