package endpoint

import "fmt"

type EndpointType string

const (
	EndpointTypeMastodon EndpointType = "mastodon"
	EndpointTypeTelegram EndpointType = "telegram"
)

type EndpointConfig struct {
	Type EndpointType `json:"type"`

	// As we don't have ADT in Golang, we simply combine all endpoint-specific fields together.
	Mastodon *EndpointConfigMastodon
	Telegram *EndpointConfigTelegram
}

var (
	ErrUnsupportedUpdate = fmt.Errorf("Update or message not supported")
)
