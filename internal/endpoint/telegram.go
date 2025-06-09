package endpoint

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/merrkry/tele2don/internal/model"
)

type EndpointConfigTelegram struct {
	BotToken  string
	ChannelID int64
}

// TODO: track recent messages in order to detect message deletion
type EndpointTelegram struct {
	id        model.EndpointID
	bot       *tg.Bot
	channelID int64
}

func NewEndpointTelegram(id model.EndpointID, channelID int64) *EndpointTelegram {
	return &EndpointTelegram{
		id:        id,
		channelID: channelID,
	}
}

func (e *EndpointTelegram) ID() model.EndpointID {
	return e.id
}

func (e *EndpointTelegram) Initialize(ctx context.Context, cfg *EndpointConfig) error {
	var err error
	e.bot, err = tg.New(cfg.Telegram.BotToken)
	if err != nil {
		return fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}
	return nil
}

func (e *EndpointTelegram) ListenUpdates(ctx context.Context, updatesChan chan<- *model.EndpointUpdate, wg *sync.WaitGroup) {
	defer wg.Done()

	e.bot.RegisterHandlerMatchFunc(e.isSupportedUpdate, func(ctx context.Context, bot *tg.Bot, update *models.Update) {
		convertedUpdate, err := e.convertUpdate(update)
		if err != nil {
			slog.Error("Failed to convert update", "err", err)
			return
		}
		updatesChan <- convertedUpdate
	})

	e.bot.Start(ctx)
}

func (e *EndpointTelegram) isSupportedUpdate(update *models.Update) bool {
	if update == nil {
		return false
	}

	return update.ChannelPost != nil || update.EditedChannelPost != nil
}

func (e *EndpointTelegram) convertUpdate(update *models.Update) (*model.EndpointUpdate, error) {
	if !e.isSupportedUpdate(update) {
		return nil, ErrUnsupportedUpdate
	}

	convertedUpdate := &model.EndpointUpdate{
		UniqueEndpointMessageID: model.UniqueEndpointMessageID{
			EID: e.id,
		},
	}

	if update.ChannelPost != nil { // new message
		convertedUpdate.Type = model.UpdateTypeNew
		convertedUpdate.Timestamp = time.Unix(int64(update.ChannelPost.Date), 0)
		convertedUpdate.ID = model.EndpointMessageID(strconv.FormatInt(int64(update.ChannelPost.ID), 10))
		convertedUpdate.Content = &model.BridgeMessageContent{
			MDText: update.ChannelPost.Text, // TODO: convert tg entities
		}
	} else if update.EditedChannelPost != nil { // edited message
		convertedUpdate.Type = model.UpdateTypeEdit
		convertedUpdate.Timestamp = time.Unix(int64(update.EditedChannelPost.EditDate), 0)
		convertedUpdate.ID = model.EndpointMessageID(strconv.FormatInt(int64(update.EditedChannelPost.ID), 10))
		convertedUpdate.Content = &model.BridgeMessageContent{
			MDText: update.EditedChannelPost.Text, // TODO: convert tg entities
		}
	}

	return convertedUpdate, nil
}

func (e *EndpointTelegram) ApplyUpdateNew(ctx context.Context, content *model.BridgeMessageContent) (model.EndpointMessageID, time.Time, error) {
	msg, err := e.bot.SendMessage(ctx, &tg.SendMessageParams{
		ChatID: e.channelID,
		Text:   content.MDText,
	})

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to send message to Telegram: %w", err)
	}

	slog.Debug("Message sent to Telegram", "id", msg.ID)

	return model.EndpointMessageID(strconv.FormatInt(int64(msg.ID), 10)), time.Unix(int64(msg.Date), 0), nil
}

func (e *EndpointTelegram) ApplyUpdateEdit(ctx context.Context, id model.EndpointMessageID, content *model.BridgeMessageContent) (time.Time, error) {
	msgID, _ := strconv.ParseInt(string(id), 10, 32)
	msg, err := e.bot.EditMessageText(ctx, &tg.EditMessageTextParams{
		ChatID:    e.channelID,
		MessageID: int(msgID),
		Text:      content.MDText,
	})

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to edit message in Telegram: %w", err)
	}

	slog.Debug("Message edited in Telegram", "id", msg.ID)

	return time.Unix(int64(msg.EditDate), 0), nil
}

func (e *EndpointTelegram) ApplyUpdateDelete(ctx context.Context, id model.EndpointMessageID) error {
	return ErrUnsupportedUpdate
}
