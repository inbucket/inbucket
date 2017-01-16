package msghub

import (
	"container/ring"
	"context"
	"sync"
	"time"
)

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

// Listener receives the contents of the log, followed by new messages
type Listener interface {
	Receive(msg Message) error
}

// Hub relays messages on to its listeners
type Hub struct {
	// log stores history, points next spot to write.  First non-nil entry is oldest Message
	log   *ring.Ring
	logMx sync.RWMutex

	// listeners interested in new messages
	listeners   map[Listener]struct{}
	listenersMx sync.RWMutex

	// broadcast receives new messages
	broadcast chan Message
}

// New constructs a new Hub which will cache logSize messages in memory for playback to future
// listeners.  A goroutine is created to handle incoming messages; it will run until the provided
// context is canceled.
func New(ctx context.Context, logSize int) *Hub {
	h := &Hub{
		log:       ring.New(logSize),
		listeners: make(map[Listener]struct{}),
		broadcast: make(chan Message, 100),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				// Shutdown
				close(h.broadcast)
				h.broadcast = nil
				return
			case msg := <-h.broadcast:
				// Log message
				h.logMx.Lock()
				h.log.Value = msg
				h.log = h.log.Next()
				h.logMx.Unlock()
				// Deliver message to listeners
				h.deliver(msg)
			}
		}
	}()

	return h
}

// Broadcast queues a message for processing by the hub.  The message will be placed into the
// in-memory log and relayed to all registered listeners.
func (h *Hub) Broadcast(msg Message) {
	if h.broadcast != nil {
		h.broadcast <- msg
	}
}

// AddListener registers a listener to receive broadcasted messages.
func (h *Hub) AddListener(l Listener) {
	// Playback log
	h.logMx.RLock()
	h.log.Do(func(v interface{}) {
		if v != nil {
			l.Receive(v.(Message))
		}
	})
	h.logMx.RUnlock()

	// Add to listeners
	h.listenersMx.Lock()
	h.listeners[l] = struct{}{}
	h.listenersMx.Unlock()
}

// RemoveListener deletes a listener registration, it will cease to receive messages.
func (h *Hub) RemoveListener(l Listener) {
	h.listenersMx.Lock()
	defer h.listenersMx.Unlock()
	if _, ok := h.listeners[l]; ok {
		delete(h.listeners, l)
	}
}

// deliver message to all listeners, removing listeners if they return an error
func (h *Hub) deliver(msg Message) {
	h.listenersMx.RLock()
	defer h.listenersMx.RUnlock()
	for l := range h.listeners {
		if err := l.Receive(msg); err != nil {
			h.RemoveListener(l)
		}
	}
}
