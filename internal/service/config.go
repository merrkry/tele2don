package service

import (
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/merrkry/tele2don/internal/endpoint"
)

type BridgeConfig struct {
	Endpoints      []*endpoint.EndpointConfig `json:"endpoints"`
	RequestTimeout time.Duration
}

// LoadDevConfig returns a temporary config for debug only
func LoadDevConfig() *BridgeConfig {
	mastodonURL, _ := url.Parse(os.Getenv("MASTODON_SERVER"))
	telegramChannelID, _ := strconv.ParseInt(os.Getenv("TELEGRAM_CHANNEL_ID"), 10, 64)
	return &BridgeConfig{
		Endpoints: []*endpoint.EndpointConfig{
			{
				Type: endpoint.EndpointTypeMastodon,
				Mastodon: &endpoint.EndpointConfigMastodon{
					InstanceURL:  *mastodonURL,
					ClientID:     os.Getenv("MASTODON_CLIENT_ID"),
					ClientSecret: os.Getenv("MASTODON_CLIENT_SECRET"),
					AccessToken:  os.Getenv("MASTODON_ACCESS_TOKEN"),
				},
			},
			{
				Type: endpoint.EndpointTypeTelegram,
				Telegram: &endpoint.EndpointConfigTelegram{
					BotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
					ChannelID: telegramChannelID,
				},
			},
		},
		RequestTimeout: 10 * time.Second,
	}
}
