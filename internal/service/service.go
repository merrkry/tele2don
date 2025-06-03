package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattn/go-mastodon"
	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/merrkry/tele2don/internal/endpoint"
	"github.com/merrkry/tele2don/internal/state"
)

type BridgeService struct {
	config     *BridgeConfig
	state      *state.BridgeState
	endpoints  []endpoint.Endpoint
	updateChan chan *bridge.BridgeMessage
}

func NewBridgeService(cfg *BridgeConfig) (*BridgeService, error) {
	s := &BridgeService{
		config: cfg,
	}

	var err error
	s.state, err = state.LoadBridgeState(cfg.DBPath)
	if err != nil {
		slog.Error("Failed to load bridge state", "err", err)
		return nil, err
	}

	s.endpoints = []endpoint.Endpoint{}

	for _, epCfg := range cfg.endpoints {
		var newEp endpoint.Endpoint
		switch epCfg.Type {
		case "mastodon":
			newEp, err = endpoint.NewMastodonEndpoint(epCfg.ID, s.state, &mastodon.Config{
				Server:       epCfg.MastodonInstance.String(),
				ClientID:     epCfg.MastodonClientID,
				ClientSecret: epCfg.MastodonClientSecret,
				AccessToken:  epCfg.MastodonAccessToken,
			})
		case "telegram":
			newEp, err = endpoint.NewTelegramEndpoint(epCfg.ID, s.state, epCfg.TelegramBotToken, epCfg.TelegramChannelID)
		default:
			err = fmt.Errorf("unsupported endpoint type: %s", epCfg.Type)
			slog.Error(err.Error())
			return nil, err
		}
		s.endpoints = append(s.endpoints, newEp)
	}

	s.updateChan = make(chan *bridge.BridgeMessage, 128)
	return s, nil
}

func (s *BridgeService) Start(ctx context.Context) error {
	slog.Debug("Starting platform endpoints")
	for _, ep := range s.endpoints {
		if ep == nil {
			continue
		}
		if err := ep.StartEndpoint(ctx, s.updateChan); err != nil {
			slog.Error("Failed to start endpoint", "err", err)
			return err
		}
	}

	slog.Debug("All endpoints started.")

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Context done, stopping update channel.")
				close(s.updateChan)
				s.state.Shutdown()
				return
			case msg := <-s.updateChan:
				slog.Debug("Received update", "msg", msg)
			}
		}
	}()

	return nil
}
