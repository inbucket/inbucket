package extension

import (
	"github.com/inbucket/inbucket/pkg/extension/event"
)

// Host defines extension points for Inbucket.
type Host struct {
	Events *Events
}

// Events defines all the event types supported by the extension host.
type Events struct {
	MessageStored EventBroker[event.MessageMetadata, Void]
}

// Void indicates the event emitter will ignore any value returned by listeners.
type Void struct{}

// NewHost creates a new extension host.
func NewHost() *Host {
	return &Host{Events: &Events{}}
}
