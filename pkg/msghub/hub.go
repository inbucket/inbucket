package msghub

import (
	"container/ring"
	"context"
	"time"
)

// Length of msghub operation queue
const opChanLen = 100

// Message contains the basic header data for a message
type Message struct {
	Mailbox string
	ID      string
	From    string
	To      []string
	Subject string
	Date    time.Time
	Size    int64
}

// Listener receives the contents of the history buffer, followed by new messages
type Listener interface {
	Receive(msg Message) error
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
func New(historyLen int) *Hub {
	return &Hub{
		history:   ring.New(historyLen),
		listeners: make(map[Listener]struct{}),
		opChan:    make(chan func(h *Hub), opChanLen),
	}
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
			op(hub)
		}
	}
}

// Dispatch queues a message for broadcast by the hub.  The message will be placed into the
// history buffer and then relayed to all registered listeners.
func (hub *Hub) Dispatch(msg Message) {
	hub.opChan <- func(h *Hub) {
		if h.history != nil {
			// Add to history buffer
			h.history.Value = msg
			h.history = h.history.Next()
			// Deliver message to all listeners, removing listeners if they return an error
			for l := range h.listeners {
				if err := l.Receive(msg); err != nil {
					delete(h.listeners, l)
				}
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
				l.Receive(v.(Message))
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
	hub.opChan <- func(h *Hub) {
		close(done)
	}
	<-done
}
