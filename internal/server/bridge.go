package server

import (
	"context"
	"fmt"
	"log/slog"

	b "github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/config"
	"github.com/merrkry/tele2don/internal/mastodon"
	"github.com/merrkry/tele2don/internal/telegram"
)

func StartBridge(ctx context.Context, cfg *config.Tele2donConfig) error {
	bridgeUpdates := make(chan b.BridgeUpdate, 128)

	platforms, err := startPlatformWorkers(ctx, cfg, bridgeUpdates)
	if err != nil {
		slog.Error("Failed to start at least one platform workers.", "err", err)
		return err
	}

	go handleUpdates(ctx, cfg, bridgeUpdates, platforms)

	return nil
}

func handleUpdates(ctx context.Context, cfg *config.Tele2donConfig, bridgeUpdates chan b.BridgeUpdate, platforms *[]b.Platform) {
	slog.Debug("Bridge update handler started.")
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context done, stopping bridge.")
		case upd := <-bridgeUpdates:
			slog.Debug("Handling bridge update", "upd", upd)
			for _, platform := range *platforms {
				if !upd.IsSupportedBy(platform) {
					continue
				}
				err := platform.ApplyUpdate(upd)
				if err != nil {
					slog.Error(fmt.Sprintf("Error applying update to %s", platform.Name()), "err", err)
				}
			}
		}
	}
}

func startPlatformWorkers(ctx context.Context, cfg *config.Tele2donConfig, bridgeChan chan b.BridgeUpdate) (*[]b.Platform, error) {
	platforms := make([]b.Platform, 0)

	slog.Info("Starting Telegram platform worker.")
	telegramPlatform, err := telegram.NewTelegramPlatform(ctx, cfg, bridgeChan)
	if err != nil {
		slog.Error("Failed to create Telegram platform worker.", "err", err)
		return nil, err
	} else {
		platforms = append(platforms, telegramPlatform)
		slog.Info("Telegram platform worker started successfully.")
	}

	slog.Info("Starting Mastodon platform worker.")
	mastodonPlatform, err := mastodon.NewTelegramPlatform(ctx, cfg, bridgeChan)
	if err != nil {
		slog.Error("Failed to create Mastodon platform worker.", "err", err)
		return nil, err
	} else {
		platforms = append(platforms, mastodonPlatform)
		slog.Info("Mastodon platform worker started successfully.")
	}

	return &platforms, nil
}
