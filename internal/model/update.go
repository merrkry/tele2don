package model

import (
	"fmt"
	"time"
)

type BridgeMessageContent struct {
	MDText string
	// TODO: attachments
}

type EndpointID int

// EndpointMessageID is a unique identifier for endpoint messages in the context of a specific endpoint.
// We use string for better compatibility.
type EndpointMessageID string

// UniqueEndpointMessageID is a endpoint-independent unique identifier for messages.
type UniqueEndpointMessageID struct {
	EID EndpointID
	ID  EndpointMessageID
}

func (i UniqueEndpointMessageID) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		fmt.Fprintf(f, "%d:%s", i.EID, i.ID)
	case 'q':
		fmt.Fprintf(f, "%d:%q", i.EID, i.ID)
	default:
		fmt.Fprintf(f, "UniqueEndpointMessageID{EID: %d, ID: %v}", i.EID, i.ID)
	}
}

type EndpointUpdateType int

const (
	UpdateTypeNew EndpointUpdateType = iota + 1 // start from 1 to avoid confusion with zero value
	UpdateTypeEdit
	UpdateTypeDelete
)

// EndpointUpdate is an abstraction of an update received from an endpoint.
// It can be either new message, edition or deletion.
type EndpointUpdate struct {
	Type EndpointUpdateType
	UniqueEndpointMessageID
	Content   *BridgeMessageContent
	Timestamp time.Time
}

type BridgeMessageID int64
