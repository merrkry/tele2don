package endpoint

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	m "github.com/mattn/go-mastodon"
	"github.com/merrkry/tele2don/internal/model"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type EndpointConfigMastodon struct {
	InstanceURL  url.URL
	ClientID     string
	ClientSecret string
	AccessToken  string
}

type EndpointMastodon struct {
	id     model.EndpointID
	client *m.Client
}

func NewEndpointMastodon(id model.EndpointID) *EndpointMastodon {
	return &EndpointMastodon{
		id: id,
	}
}

func (e *EndpointMastodon) ID() model.EndpointID {
	return e.id
}

func (e *EndpointMastodon) Initialize(ctx context.Context, cfg *EndpointConfig) error {
	clientConfig := &m.Config{
		Server:       cfg.Mastodon.InstanceURL.String(),
		ClientID:     cfg.Mastodon.ClientID,
		ClientSecret: cfg.Mastodon.ClientSecret,
		AccessToken:  cfg.Mastodon.AccessToken,
	}

	e.client = m.NewClient(clientConfig)

	// TODO: validate

	return nil
}

func (e *EndpointMastodon) ListenUpdates(ctx context.Context, updatesChan chan<- *model.EndpointUpdate, wg *sync.WaitGroup) {
	defer wg.Done()

	eventChan, err := e.client.StreamingUser(ctx)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-eventChan:
			if event == nil {
				continue
			}
			convertedUpdate, err := e.convertEvent(event)
			if err != nil {
				slog.Error("Failed to convert Mastodon event", "err", err)
				continue
			}
			updatesChan <- convertedUpdate
		}
	}
}

func (e *EndpointMastodon) convertEvent(event m.Event) (*model.EndpointUpdate, error) {
	if event == nil {
		return nil, ErrUnsupportedUpdate
	}

	convertedUpdate := &model.EndpointUpdate{
		UniqueEndpointMessageID: model.UniqueEndpointMessageID{
			EID: e.id,
		},
	}

	switch event := event.(type) {
	case *m.UpdateEvent:
		convertedUpdate.Type = model.UpdateTypeNew
		convertedUpdate.ID = model.EndpointMessageID(event.Status.ID)
		convertedUpdate.Timestamp = event.Status.CreatedAt

		convertedContent, err := htmltomarkdown.ConvertString(event.Status.Content)
		if err != nil {
			return nil, err
		}
		convertedUpdate.Content = &model.BridgeMessageContent{
			MDText: convertedContent,
		}

	case *m.UpdateEditEvent:
		convertedUpdate.Type = model.UpdateTypeEdit
		convertedUpdate.ID = model.EndpointMessageID(event.Status.ID)
		convertedUpdate.Timestamp = event.Status.EditedAt

		convertedContent, err := htmltomarkdown.ConvertString(event.Status.Content)
		if err != nil {
			return nil, err
		}
		convertedUpdate.Content = &model.BridgeMessageContent{
			MDText: convertedContent,
		}

	case *m.DeleteEvent:
		// convertedUpdate.Type = model.UpdateTypeDelete
		return nil, ErrUnsupportedUpdate

	default:
		return nil, ErrUnsupportedUpdate
	}

	return convertedUpdate, nil
}

func (e *EndpointMastodon) ApplyUpdateNew(ctx context.Context, content *model.BridgeMessageContent) (model.EndpointMessageID, time.Time, error) {
	status, err := e.client.PostStatus(ctx, &m.Toot{
		Status: content.MDText,
		// TODO: detect language
	})

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to post status to Mastodon: %w", err)
	}

	slog.Debug("Status posted to Mastodon", "id", status.ID)

	return model.EndpointMessageID(status.ID), status.CreatedAt, nil
}

func (e *EndpointMastodon) ApplyUpdateEdit(ctx context.Context, id model.EndpointMessageID, content *model.BridgeMessageContent) (time.Time, error) {
	status, err := e.client.UpdateStatus(ctx, &m.Toot{
		Status: content.MDText,
	}, m.ID(id))

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to edit status in Mastodon: %w", err)
	}

	slog.Debug("Status edited in Mastodon", "id", status.ID)

	return status.EditedAt, nil
}

func (e *EndpointMastodon) ApplyUpdateDelete(ctx context.Context, id model.EndpointMessageID) error {
	return ErrUnsupportedUpdate
}
