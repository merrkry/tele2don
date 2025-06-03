package state

import (
	"database/sql"
	"errors"
	"log/slog"
	"sync/atomic"

	"github.com/merrkry/tele2don/internal/bridge"
	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type EndpointID int
type PlatformMessageID string // for better compatibility across platforms

type BridgeState struct {
	db          *sql.DB
	idGenerator int64
	// TODO: map between endpoint id and type
}

func LoadBridgeState(dbPath string) (*BridgeState, error) {
	s := &BridgeState{}

	var err error
	s.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		slog.Error("Failed to validate database path", "err", err)
		return nil, err
	}
	err = s.db.Ping()
	if err != nil {
		slog.Error("Failed to connect to database", "err", err)
		return nil, err
	}

	err = s.initDB()
	if err != nil {
		slog.Error("Failed to initialize database", "err", err)
		return nil, err
	}

	maxRow := s.db.QueryRow("SELECT MAX(id) FROM bridge_messages")
	if maxRow != nil {
		maxRow.Scan(&s.idGenerator)
	} else {
		s.idGenerator = 0
	}

	return s, nil
}

func (s *BridgeState) initDB() error {
	// TODO: consider using gorm
	// TODO: use transactions

	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS bridge_messages (
		id INTEGER PRIMARY KEY,
		content BLOB NOT NULL
	)
	`)
	if err != nil {
		return err
	}

	// _, err = s.db.Exec(`
	// CREATE TABLE IF NOT EXISTS endpoints (
	// 	id INTEGER PRIMARY KEY,
	// 	type TEXT NOT NULL
	// )
	// `)
	// if err != nil {
	// 	return err
	// }

	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS platform_messages (
		eid INTEGER NOT NULL,
		id TEXT NOT NULL, -- use TEXT for compatibility
		bridge_message_id INTEGER NOT NULL,
		-- FOREIGN KEY (eid) REFERENCES endpoints(id),
		FOREIGN KEY (bridge_message_id) REFERENCES bridge_messages(id),
		PRIMARY KEY (eid, id)
	)
	`)
	if err != nil {
		return err
	}

	// TODO: add indexes

	return nil
}

// func (s *BridgeState) QueryPlatformMessage() (uuid.UUID, error) {
// 	return uuid.UUID{}, nil
// }

// maybe we don't need to expose bridge message id

// func (s *BridgeState) WriteBridgeMessage() error {
// 	return nil
// }

var ErrNotTracked = errors.New("Message not tracked")
var ErrInvalidState = errors.New("Inlivad internal state")

func (s *BridgeState) QueryPlatformMessage(eid EndpointID, id PlatformMessageID) (bridge.BridgeMessageID, error) {
	result, err := s.db.Query(`
	SELECT id
	FROM platform_messages
	WHERE eid = ?
	AND id = ?
	`, eid, id)
	if result != nil {
		defer result.Close()
	} else {
		return 0, ErrNotTracked
	}

	if err != nil {
		return 0, err
	}

	var bid bridge.BridgeMessageID
	if ok := result.Next(); ok {
		err = result.Scan(&bid)
		if err != nil || result.Next() {
			return 0, ErrInvalidState
		}
	} else {
		return 0, ErrNotTracked
	}

	return bid, nil
}

func (s *BridgeState) WriteBridgeMessage(message *bridge.BridgeMessage) error {
	if message == nil || message.Content == nil {
		return errors.New("Cannot write nil bridge message")
	}

	_, err := s.db.Exec(`
	INSERT INTO bridge_messages
	VALUES (?, ?)
	`, message.ID, sqlite3.JSON(*message.Content))

	if err != nil {
		slog.Error("Failed to insert bridge message to database", "err", err)
	}

	return nil
}

func (s *BridgeState) LinkPlatformMessage(eid EndpointID, id PlatformMessageID, bridgeMessageID int64) error {
	_, err := s.db.Exec(`
	INSERT INTO platform_messages
	VALUES (?, ?, ?)
	`, eid, id, bridgeMessageID)
	if err != nil {
		slog.Error("Failed to link platform message to bridge message", "err", err)
		return err
	}
	return nil
}

// func (s *BridgeState) RegisterEndpoint(id EndpointID, platform string, rawConfig string) error {
// 	return nil
// }

func (s *BridgeState) Shutdown() {
	if s.db != nil {
		s.db.Close() // should I handle errors here?
	}
}

func (s *BridgeState) NextID() bridge.BridgeMessageID {
	slog.Debug("New message requested", "id", s.idGenerator+1)
	return bridge.BridgeMessageID(atomic.AddInt64(&s.idGenerator, 1))
}
