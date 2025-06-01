package bridge

type MessageIdentifier string

const NilMessageIdentifier MessageIdentifier = ""

type BridgeUpdate interface {
	IsSupportedBy(Platform) bool
	IsOrigin(Platform) bool
	GetIdentifier() MessageIdentifier
}

type NewMessage struct {
	Text       string
	Origin     Platform
	Identifier MessageIdentifier
}

func (m NewMessage) IsSupportedBy(platform Platform) bool {
	return platform.Name() == PlatformTelegram || platform.Name() == PlatformMastodon
}

func (m NewMessage) IsOrigin(other Platform) bool {
	return m.Origin != nil && m.Origin == other
}

func (m NewMessage) GetIdentifier() MessageIdentifier {
	return m.Identifier
}

type MessageEdit struct {
	Text string
}
