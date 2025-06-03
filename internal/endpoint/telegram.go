package endpoint

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/state"
)

type TelegramEndpoint struct {
	id    state.EndpointID
	bot   *bot.Bot
	state *state.BridgeState

	channelID int64
}

func (e *TelegramEndpoint) ApplyUpdate(upd bridge.BridgeMessage) (bridge.BridgeMessageID, error) {
	return 0, nil
}

func (e *TelegramEndpoint) StartEndpoint(ctx context.Context, updateChan chan<- *bridge.BridgeMessage) error {
	go e.bot.Start(ctx)
	e.bot.RegisterHandlerMatchFunc(func(u *models.Update) bool {
		return u.ChannelPost != nil && u.ChannelPost.Chat.ID == e.channelID
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		e.handleUpdate(ctx, update)
	})
	return nil
}

func (e *TelegramEndpoint) handleUpdate(_ context.Context, update *models.Update) {
	if update == nil || update.ChannelPost == nil {
		return
	}
	p := update.ChannelPost
	slog.Debug("Received Telegram update", "upd", p)

	// skip unsupported message types
	if p.ReplyToMessage != nil || p.ForwardOrigin != nil {
		return
	}

	bid, err := e.state.QueryPlatformMessage(e.id, state.PlatformMessageID(strconv.Itoa(p.ID)))
	if err == state.ErrNotTracked {
		bm, err := e.convertPost(p)
		if err != nil {
			slog.Error("Failed to convert Telegram post to bridge message", "err", err)
			return
		}
		e.state.WriteBridgeMessage(bm)
		return
	} else if err != nil {
		slog.Error("Failed to query platform message", "err", err)
		return
	}

	// TODO
	slog.Debug("Message edit not supported yet", "bid", bid)
}

func NewTelegramEndpoint(id state.EndpointID, state *state.BridgeState, botToken string, channelID int64) (*TelegramEndpoint, error) {
	ep := &TelegramEndpoint{
		id:        id,
		state:     state,
		channelID: channelID,
	}

	var err error
	ep.bot, err = bot.New(botToken)
	if err != nil {
		slog.Error("Failed to create Telegram bot", "err", err)
		return nil, err
	}

	// TODO: verify if bot token and channel ID are valid,
	// maybe also read permissions to channel posts
	// perhaps use GetMe for authentication check

	return ep, nil
}

func (e *TelegramEndpoint) convertPost(m *models.Message) (*bridge.BridgeMessage, error) {
	return &bridge.BridgeMessage{
		ID: e.state.NextID(),
		Content: &bridge.BridgeMessageContent{
			MDText: m.Text, // TODO: convert entities to Markdown
		},
	}, nil
}
