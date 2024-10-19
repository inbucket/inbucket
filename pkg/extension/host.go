package extension

import (
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
)

// Host defines extension points for Inbucket.
type Host struct {
	Events *Events
}

// Events defines all the event types supported by the extension host.
//
// Before-events provide an opportunity for extensions to alter how Inbucket responds to that type
// of event.  These events are processed synchronously; expensive operations will reduce the
// perceived performance of Inbucket.  The first listener in the list to respond with a non-nil
// value will determine the response, and the remaining listeners will not be called.
//
// After-events allow extensions to take an action after an event has completed.  These events are
// processed asynchronously with respect to the rest of Inbuckets operation.  However, an event
// listener will not be called until the one before it completes.
type Events struct {
	AfterMessageDeleted    AsyncEventBroker[event.MessageMetadata]
	AfterMessageStored     AsyncEventBroker[event.MessageMetadata]
	BeforeMailFromAccepted EventBroker[event.SMTPSession, event.SMTPResponse]
	BeforeMessageStored    EventBroker[event.InboundMessage, event.InboundMessage]
	BeforeRcptToAccepted   EventBroker[event.SMTPSession, event.SMTPResponse]
}

// Void indicates the event emitter will ignore any value returned by listeners.
type Void struct{}

// NewHost creates a new extension host.
func NewHost() *Host {
	return &Host{Events: &Events{}}
}
