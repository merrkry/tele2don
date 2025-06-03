package bridge

type BridgeMessageID int64
type BridgeMessageRevision int64

type BridgeMessage struct {
	ID BridgeMessageID
	// Revision BridgeMessageRevision
	Content *BridgeMessageContent
}

type BridgeMessageContent struct {
	MDText string `json:"md_text,omitempty"`
}
