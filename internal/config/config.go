package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Tele2donConfig struct {
	MastodonClientID     string
	MastodonClientSecret string
	MastodonAccessToken  string
	TelegramBotToken     string
	TelegramChannelID    int64
}

func LoadConfig() (*Tele2donConfig, error) {
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	if err != nil {
		slog.Error("Unknow log level", "err", err)
		return nil, err
	}
	slog.SetLogLoggerLevel(logLevel)

	telegramChannelID, _ := strconv.ParseInt(os.Getenv("TELEGRAM_CHANNEL_ID"), 10, 64)

	return &Tele2donConfig{
		MastodonClientID:     os.Getenv("MASTODON_CLIENT_ID"),
		MastodonClientSecret: os.Getenv("MASTODON_CLIENT_SECRET"),
		MastodonAccessToken:  os.Getenv("MASTODON_ACCESS_TOKEN"),
		TelegramBotToken:     os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChannelID:    telegramChannelID,
	}, nil
}
