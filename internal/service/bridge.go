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

			bid, ok := s.queryOrCreateBridgeMessage(update)
			if !ok {
				continue
			}

			slog.Debug("Processing endpoint update", "bid", bid, "uniqueID", update.UniqueEndpointMessageID, "type", update.Type, "timestamp", update.Timestamp)

			switch update.Type {
			case model.UpdateTypeNew:
				s.applyUpdateNew(ctx, update, bid)

			case model.UpdateTypeEdit:
				s.applyUpdateEdit(ctx, update, bid)

			case model.UpdateTypeDelete:
				slog.Error("Message deletion not supported yet.")

			default:
				panic(fmt.Sprintf("Unknown update type %d for %q", update.Type, update.UniqueEndpointMessageID))
			}
		}
	}
}

// queryOrCreateBridgeMessage queries the cache for an existing bridge message ID or creates a new one if it doesn't exist.
// The second return value only indicates if the message should be processed further.
// In case of invalid internal state, query/create will fail, it will panic.
func (s *BridgeService) queryOrCreateBridgeMessage(update *model.EndpointUpdate) (model.BridgeMessageID, bool) {
	var bid model.BridgeMessageID

	time, err := s.Cache.QueryRevision(update.UniqueEndpointMessageID)
	if err == nil {
		if time.Equal(update.Timestamp) { // Already tracked message
			return 0, false
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
			return 0, false
		}
	} else {
		panic(fmt.Sprintf("Failed to query revision for %q: %v", update.UniqueEndpointMessageID, err))
	}

	if bid == 0 {
		panic(fmt.Sprintf("Failed to retrieve or generate bridge message ID for %q", update.UniqueEndpointMessageID))
	}

	return bid, true
}

func (s *BridgeService) applyUpdateNew(ctx context.Context, update *model.EndpointUpdate, bid model.BridgeMessageID) {
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
}

func (s *BridgeService) applyUpdateEdit(ctx context.Context, update *model.EndpointUpdate, bid model.BridgeMessageID) {
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
}
