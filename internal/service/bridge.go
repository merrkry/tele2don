package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/merrkry/tele2don/internal/endpoint"
	"github.com/merrkry/tele2don/internal/model"
)

type BridgeService struct {
	Cache     BridgeCache
	Config    *BridgeConfig
	Endpoints []Endpoint
}

// LoadBridgeService loads configuration and initializes the BridgeService.
func LoadBridgeService(ctx context.Context) (*BridgeService, error) {
	s := &BridgeService{
		Cache:  NewBridgeCache(),
		Config: LoadDevConfig(),
	}

	for id, endpointConfig := range s.Config.Endpoints {
		id := model.EndpointID(id)
		var ep Endpoint
		switch endpointConfig.Type {
		case endpoint.EndpointTypeMastodon:
			ep = endpoint.NewEndpointMastodon(id)
		case endpoint.EndpointTypeTelegram:
			ep = endpoint.NewEndpointTelegram(id, endpointConfig.Telegram.ChannelID)
		default:
			return nil, fmt.Errorf("unsupported endpoint type %s", endpointConfig.Type)
		}
		err := ep.Initialize(ctx, endpointConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize endpoint %d: %w", id, err)
		}
		s.Endpoints = append(s.Endpoints, ep)
	}

	return s, nil
}

func (s *BridgeService) Start(ctx context.Context) {
	updatesChan := make(chan *model.EndpointUpdate, 128)
	defer close(updatesChan)

	var wg sync.WaitGroup
	wg.Add(len(s.Endpoints))

	for _, endpoint := range s.Endpoints {
		slog.Info("Starting endpoint", "eid", endpoint.ID())
		go endpoint.ListenUpdates(ctx, updatesChan, &wg)
	}

	// This can be further parallelized with multiple workers.
	go s.HandleEndpointUpdates(ctx, updatesChan)

	wg.Wait()
}

func (s *BridgeService) HandleEndpointUpdates(ctx context.Context, updatesChan <-chan *model.EndpointUpdate) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updatesChan:
			if update == nil {
				continue
			}

			slog.Debug("Received update from endpoint", "id", update.UniqueEndpointMessageID, "rev", update.Timestamp, "content", update.Content)

			// Query cache and update, obsolete or loopback messages are ignored.

			var bid model.BridgeMessageID

			time, err := s.Cache.QueryRevision(update.UniqueEndpointMessageID)
			if err == nil {
				if time.Equal(update.Timestamp) { // Already tracked message
					continue
				}
				bid, err = s.Cache.QueryBridgeMessageID(update.UniqueEndpointMessageID)
				if err != nil {
					panic(fmt.Sprintf("Failed to query bridge message ID for %q: %v", update.UniqueEndpointMessageID, err))
				}
				err = s.Cache.UpdateEndpointMessage(update.UniqueEndpointMessageID, update.Timestamp)
				if err != nil {
					panic(fmt.Sprintf("Failed to update endpoint message for %q: %v", update.UniqueEndpointMessageID, err))
				}
			} else if errors.Is(err, ErrMessageNotFound) {
				if update.Type == model.UpdateTypeNew {
					bid = s.Cache.NewBridgeMessage()
					err := s.Cache.CreateEndpointMessage(update.UniqueEndpointMessageID, bid, update.Timestamp)
					if err != nil {
						panic(fmt.Sprintf("Failed to create endpoint message for %q: %v", update.UniqueEndpointMessageID, err))
					}
				} else { // Message is older than our state, ignore it
					continue
				}
			} else {
				panic(fmt.Sprintf("Failed to query revision for %q: %v", update.UniqueEndpointMessageID, err))
			}

			if bid == 0 {
				panic(fmt.Sprintf("Failed to retrieve or generate bridge message ID for %q", update.UniqueEndpointMessageID))
			}

			slog.Debug("Processing update", "type", update.Type, "eid", update.UniqueEndpointMessageID.EID, "bid", bid, "rev", update.Timestamp)

			// Forward update

			switch update.Type {
			case model.UpdateTypeNew:
				for eid, endpoint := range s.Endpoints {
					if endpoint.ID() == update.EID {
						continue
					}
					eid := model.EndpointID(eid)

					updateCtx, cancel := context.WithTimeout(ctx, s.Config.RequestTimeout)
					defer cancel()

					id, rev, err := endpoint.ApplyUpdateNew(updateCtx, update.Content)
					if err != nil {
						slog.Error("Failed to apply update to endpoint", "eid", endpoint.ID(), "err", err)
						continue
					}
					err = s.Cache.CreateEndpointMessage(model.UniqueEndpointMessageID{
						EID: eid,
						ID:  id,
					}, bid, rev)
					if err != nil {
						panic(fmt.Sprintf("Failed to create endpoint message for %q: %v", id, err))
					}
				}

			case model.UpdateTypeEdit:
				associatedMessages, err := s.Cache.QueryEndpointMessages(bid)
				if err != nil {
					panic(fmt.Sprintf("Failed to query associated messages for bridge message ID %d: %v", bid, err))
				}

				for _, uniqueID := range associatedMessages {
					if uniqueID.EID == update.UniqueEndpointMessageID.EID {
						continue
					}

					ctx, cancel := context.WithTimeout(ctx, s.Config.RequestTimeout)
					defer cancel()

					rev, err := s.Endpoints[uniqueID.EID].ApplyUpdateEdit(ctx, uniqueID.ID, update.Content)
					if err != nil {
						slog.Error("Failed to apply update edit to endpoint", "eid", uniqueID.EID, "id", uniqueID.ID, "err", err)
						continue
					}

					err = s.Cache.UpdateEndpointMessage(uniqueID, rev)
					if err != nil {
						panic(fmt.Sprintf("Failed to update endpoint message for %q: %v", uniqueID, err))
					}
				}

			case model.UpdateTypeDelete:
				slog.Error("Message deletion not supported yet.")

			default:
				panic(fmt.Sprintf("Unknown update type %d for %q", update.Type, update.UniqueEndpointMessageID))
			}
		}
	}
}
