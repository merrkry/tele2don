package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	m "github.com/merrkry/tele2don/internal/model"
)

var (
	ErrMessageNotFound      = errors.New("message not found in cache")
	ErrMessageAlreadyExists = errors.New("message already exists in cache")
)

type BridgeCache interface {
	QueryRevision(m.UniqueEndpointMessageID) (time.Time, error)
	QueryBridgeMessageID(m.UniqueEndpointMessageID) (m.BridgeMessageID, error)
	NewBridgeMessage() m.BridgeMessageID
	CreateEndpointMessage(m.UniqueEndpointMessageID, m.BridgeMessageID, time.Time) error
	UpdateEndpointMessage(m.UniqueEndpointMessageID, time.Time) error
	QueryEndpointMessages(m.BridgeMessageID) ([]m.UniqueEndpointMessageID, error)
}

func NewBridgeCache() BridgeCache {
	return &nativeMemoryCache{
		associatedMessages: make(map[m.BridgeMessageID][]m.UniqueEndpointMessageID),
		endpointMessages:   make(map[m.UniqueEndpointMessageID]*cachedEndpointMessage),
	}
}

// TODO: cache expiration
type nativeMemoryCache struct {
	associatedMessages map[m.BridgeMessageID][]m.UniqueEndpointMessageID
	endpointMessages   map[m.UniqueEndpointMessageID]*cachedEndpointMessage

	idCounter int64
	// We might have multiple bridge goroutine in the future.
	mu sync.RWMutex
}

type cachedEndpointMessage struct {
	rev time.Time
	bid m.BridgeMessageID
}

func (c *nativeMemoryCache) QueryRevision(id m.UniqueEndpointMessageID) (time.Time, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msg, ok := c.endpointMessages[id]
	if ok {
		return msg.rev, nil
	} else {
		return time.Time{}, ErrMessageNotFound
	}
}

func (c *nativeMemoryCache) QueryBridgeMessageID(id m.UniqueEndpointMessageID) (m.BridgeMessageID, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msg, ok := c.endpointMessages[id]
	if ok {
		return msg.bid, nil
	} else {
		return m.BridgeMessageID(0), ErrMessageNotFound
	}
}

func (c *nativeMemoryCache) NewBridgeMessage() m.BridgeMessageID {
	bid := m.BridgeMessageID(atomic.AddInt64(&c.idCounter, 1))
	c.associatedMessages[bid] = []m.UniqueEndpointMessageID{}
	return bid
}

func (c *nativeMemoryCache) CreateEndpointMessage(emid m.UniqueEndpointMessageID, bmid m.BridgeMessageID, rev time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TOOD: internal status validation

	c.endpointMessages[emid] = &cachedEndpointMessage{
		rev: rev,
		bid: bmid,
	}

	c.associatedMessages[bmid] = append(c.associatedMessages[bmid], emid)

	return nil
}

func (c *nativeMemoryCache) UpdateEndpointMessage(emid m.UniqueEndpointMessageID, rev time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg, ok := c.endpointMessages[emid]
	if !ok {
		return ErrMessageNotFound
	}

	if msg.rev.After(rev) {
		return nil // No update needed
	}

	msg.rev = rev
	return nil
}

func (c *nativeMemoryCache) QueryEndpointMessages(bid m.BridgeMessageID) ([]m.UniqueEndpointMessageID, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	msgs, ok := c.associatedMessages[bid]
	if !ok {
		return nil, ErrMessageNotFound
	}

	return msgs, nil
}
