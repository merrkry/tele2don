package bridge

type PlatformName string

const (
	PlatformMastodon PlatformName = "mastodon"
	PlatformTelegram PlatformName = "telegram"
)

type Platform interface {
	GetPlatformName() PlatformName
	ApplyUpdate(update BridgeUpdate) error
}
