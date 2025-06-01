package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/config"
)

type telegramPlatform struct {
	bot *bot.Bot
}

func (p *telegramPlatform) Name() bridge.PlatformName {
	return bridge.PlatformTelegram
}

func NewTelegramPlatform(ctx context.Context, cfg *config.Tele2donConfig, bridgeChan chan bridge.BridgeUpdate) (*telegramPlatform, error) {
	p := &telegramPlatform{}

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			handleUpdate(ctx, cfg, bridgeChan, b, update, p)
		}),
	}
	b, err := bot.New(cfg.TelegramBotToken, opts...)
	p.bot = b
	if err != nil {
		slog.Error("Failed to create Telegram bot.", "err", err)
		return nil, err
	}
	go b.Start(ctx)
	return p, nil
}

func handleUpdate(ctx context.Context, cfg *config.Tele2donConfig, targetChan chan bridge.BridgeUpdate, b *bot.Bot, update *models.Update, platform *telegramPlatform) {
	if update == nil {
		return
	}
	slog.Debug("Received update from Telegram", "update", update)

	if update.ChannelPost != nil && update.ChannelPost.Chat.ID == cfg.TelegramChannelID {
		p := update.ChannelPost
		if p.Text != "" {
			slog.Debug("Converting message from Telegram", "text", p.Text, "id", p.ID)
			targetChan <- bridge.NewMessage{
				Text:       p.Text,
				Origin:     platform,
				Identifier: bridge.MessageIdentifier("tg:" + strconv.Itoa(p.ID)),
			}
		}
	}

	// if update.EditedChannelPost != nil && update.EditedChannelPost.Chat.ID == cfg.TelegramChannelID {
	// 	p := update.EditedChannelPost
	// 	if p.Text != "" {
	// 		slog.Debug("Converting edit from Telegram", "text", p.Text, "id", p.ID)
	// 		targetChan <- bridge.NewMessage{
	// 			Text:   p.Text,
	// 			Origin: platform,
	// 		}
	// 	}
	// }
}

func (p *telegramPlatform) ApplyUpdate(cfg *config.Tele2donConfig, update bridge.BridgeUpdate) (bridge.MessageIdentifier, error) {
	if update == nil {
		return bridge.NilMessageIdentifier, fmt.Errorf("update is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	switch u := update.(type) {
	case bridge.NewMessage:
		tgMsg, err := p.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: cfg.TelegramChannelID,
			Text:   u.Text,
		})
		if err != nil {
			slog.Error("Failed to send message to Telegram channel", "err", err)
			return bridge.NilMessageIdentifier, err
		}
		return bridge.MessageIdentifier("tg:" + strconv.Itoa(tgMsg.ID)), nil
	default:
		return bridge.NilMessageIdentifier, fmt.Errorf("unsupported update type: %T", update)
	}
}
