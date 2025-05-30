package telegram

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/config"
)

type telegramPlatform struct {
	bot *bot.Bot
}

func (p *telegramPlatform) GetPlatformName() bridge.PlatformName {
	return bridge.PlatformTelegram
}

func NewTelegramPlatform(ctx context.Context, cfg *config.Tele2donConfig, bridgeChan chan bridge.BridgeUpdate) (*telegramPlatform, error) {
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			handleUpdate(ctx, cfg, bridgeChan, b, update)
		}),
	}
	bot, err := bot.New(cfg.TelegramBotToken, opts...)
	if err != nil {
		slog.Error("Failed to create Telegram bot.", "err", err)
		return nil, err
	}
	go bot.Start(ctx)
	return &telegramPlatform{
		bot: bot,
	}, nil
}

func handleUpdate(ctx context.Context, cfg *config.Tele2donConfig, targetChan chan bridge.BridgeUpdate, b *bot.Bot, update *models.Update) {
	slog.Debug("Received update from Telegram", "update", update)
	if update != nil && update.ChannelPost != nil && update.ChannelPost.Chat.ID == cfg.TelegramChannelID {
		p := update.ChannelPost
		if p.Text != "" {
			slog.Debug("Converting message from Telegram", "text", p.Text)
			slog.Debug("Channel status", "size", len(targetChan), "capacity", cap(targetChan))
			targetChan <- bridge.NewMessage{
				Text: p.Text,
			}
			slog.Debug("Message sent to bridge channel", "size", len(targetChan))
		}
	}
}

func (p *telegramPlatform) ApplyUpdate(update bridge.BridgeUpdate) error {
	return nil
}
