package bridge

type BridgeUpdate interface {
	IsSupportedBy(Platform) bool
}

type NewMessage struct {
	Text string
}

func (m NewMessage) IsSupportedBy(platform Platform) bool {
	return platform.GetPlatformName() == PlatformTelegram
}
