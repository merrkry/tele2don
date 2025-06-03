package service

import (
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strconv"

	"github.com/merrkry/tele2don/internal/state"
)

type PlatformType string

var SupportedPlatforms = []PlatformType{"telegram", "mastodon"}

type BridgeConfig struct {
	DBPath    string
	endpoints []*EndpointConfig
}

type EndpointConfig struct {
	ID   state.EndpointID
	Type PlatformType

	// mastodon
	MastodonInstance *url.URL
	// MastodonUsername     string // Do I need this?
	MastodonClientID     string
	MastodonClientSecret string
	MastodonAccessToken  string

	// telegram
	TelegramChannelID int64
	TelegramBotToken  string
}

func LoadConfig() (*BridgeConfig, error) {
	return nil, nil
}

func (c *BridgeConfig) Validate() error {
	for _, endpoint := range c.endpoints {
		if err := endpoint.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *EndpointConfig) Validate() error {
	if !slices.Contains(SupportedPlatforms, c.Type) {
		return fmt.Errorf("Unsupported platform type: %s", c.Type)
	}
	return nil
}

// LoadTempDebugConfig is a placeholder function for loading temporary debug configurations.
// TODO: remove this
func LoadTempDebugConfig() *BridgeConfig {
	cfg := &BridgeConfig{
		endpoints: []*EndpointConfig{
			{
				ID:                   1,
				Type:                 "mastodon",
				MastodonInstance:     func() *url.URL { u, _ := url.Parse(os.Getenv("MASTODON_SERVER")); return u }(),
				MastodonClientID:     os.Getenv("MASTODON_CLIENT_ID"),
				MastodonClientSecret: os.Getenv("MASTODON_CLIENT_SECRET"),
				MastodonAccessToken:  os.Getenv("MASTODON_ACCESS_TOKEN"),
			},
			{
				ID:                2,
				Type:              "telegram",
				TelegramChannelID: func() int64 { id, _ := strconv.Atoi(os.Getenv("TELEGRAM_CHANNEL_ID")); return int64(id) }(),
				TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
			},
		},
	}

	flag.StringVar(&cfg.DBPath, "db-path", "tele2don.db", "Path to the database file")

	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("Loaded config", "mastodon", *cfg.endpoints[0], "telegram", *cfg.endpoints[1])
	return cfg
}
