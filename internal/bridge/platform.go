package bridge

import "github.com/merrkry/tele2don/internal/config"

type PlatformName string

const (
	PlatformMastodon PlatformName = "mastodon"
	PlatformTelegram PlatformName = "telegram"
)

type Platform interface {
	Name() PlatformName
	ApplyUpdate(*config.Tele2donConfig, BridgeUpdate) (MessageIdentifier, error)
}
