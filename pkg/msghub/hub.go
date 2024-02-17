package msghub

import (
	"container/ring"
	"context"

	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/rs/zerolog/log"
)

// Length of msghub operation queue
const opChanLen = 100

// Listener receives the contents of the history buffer, followed by new messages
type Listener interface {
	Receive(msg event.MessageMetadata) error
	Delete(mailbox string, id string) error
}

// Hub relays messages on to its listeners
type Hub struct {
	// history buffer, points next Message to write.  Proceeding non-nil entry is oldest Message
	history   *ring.Ring
	listeners map[Listener]struct{} // listeners interested in new messages
	opChan    chan func(h *Hub)     // operations queued for this actor
}

// New constructs a new Hub which will cache historyLen messages in memory for playback to future
// listeners.  A goroutine is created to handle incoming messages; it will run until the provided
// context is canceled.
func New(historyLen int, extHost *extension.Host) *Hub {
	hub := &Hub{
		history:   ring.New(historyLen),
		listeners: make(map[Listener]struct{}),
		opChan:    make(chan func(h *Hub), opChanLen),
	}

	// Register an extension event listener for MessageStored.
	extHost.Events.AfterMessageStored.AddListener("msghub",
		func(msg event.MessageMetadata) {
			hub.Dispatch(msg)
		})

	extHost.Events.AfterMessageDeleted.AddListener("msghub",
		func(msg event.MessageMetadata) {
			hub.Delete(msg.Mailbox, msg.ID)
		})

	return hub
}

// Start Hub processing loop.
func (hub *Hub) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown
			close(hub.opChan)
			return
		case op := <-hub.opChan:
			hub.runOp(op)
		}
	}
}

// Dispatch queues a message for broadcast by the hub.  The message will be placed into the
// history buffer and then relayed to all registered listeners.
func (hub *Hub) Dispatch(msg event.MessageMetadata) {
	hub.opChan <- func(h *Hub) {
		if h.history != nil {
			// Add to history buffer
			h.history.Value = msg
			h.history = h.history.Next()

			// Relay event to all listeners, removing listeners if they return an error.
			for l := range h.listeners {
				if err := l.Receive(msg); err != nil {
					delete(h.listeners, l)
				}
			}
		}
	}
}

// Delete removes the message from the history buffer and instructs listeners to do the same.
func (hub *Hub) Delete(mailbox string, id string) {
	hub.opChan <- func(h *Hub) {
		if h.history == nil {
			return
		}

		// Locate and remove history entry.
		p := h.history
		end := p
		for {
			if next, ok := p.Next().Value.(event.MessageMetadata); ok {
				if mailbox == next.Mailbox && id == next.ID {
					p.Next().Value = nil
					break
				}
			}
			if p = p.Next(); p == end {
				break
			}
		}

		// Relay event to all listeners, removing listeners if they return an error.
		for l := range h.listeners {
			if err := l.Delete(mailbox, id); err != nil {
				delete(h.listeners, l)
			}
		}
	}
}

// AddListener registers a listener to receive broadcasted messages.
func (hub *Hub) AddListener(l Listener) {
	hub.opChan <- func(h *Hub) {
		// Playback log
		h.history.Do(func(v interface{}) {
			if v != nil {
				_ = l.Receive(v.(event.MessageMetadata))
			}
		})

		// Add to listeners
		h.listeners[l] = struct{}{}
	}
}

// RemoveListener deletes a listener registration, it will cease to receive messages.
func (hub *Hub) RemoveListener(l Listener) {
	hub.opChan <- func(h *Hub) {
		delete(h.listeners, l)
	}
}

// Sync blocks until the msghub has processed its queue up to this point, useful
// for unit tests.
func (hub *Hub) Sync() {
	done := make(chan struct{})
	hub.opChan <- func(_ *Hub) {
		close(done)
	}
	<-done
}

func (hub *Hub) runOp(op func(*Hub)) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				log.Error().Str("module", "msghub").Err(err).Msg("Operation panicked")
			} else {
				log.Error().Str("module", "msghub").Err(err).Msgf("Operation panicked: %s", r)
			}
		}
	}()

	op(hub)
}
