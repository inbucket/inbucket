package extension

import (
	"errors"
	"sync"
	"time"
)

// AsyncEventBroker maintains a list of listeners interested in a specific type
// of event.  Events are sent in parallel to all listeners, and no result is
// returned.
type AsyncEventBroker[E any] struct {
	sync.RWMutex
	listenerNames []string  // Ordered listener names.
	listenerFuncs []func(E) // Ordered listener functions.
}

// Emit sends the provided event to each registered listener in parallel.
func (eb *AsyncEventBroker[E]) Emit(event *E) {
	eb.RLock()
	defer eb.RUnlock()

	for _, l := range eb.listenerFuncs {
		// Events are copied to minimize the risk of mutation.
		go l(*event)
	}
}

// AddListener registers the named listener, replacing one with a duplicate
// name if present.  Listeners should be added in order of priority, most
// significant first.
func (eb *AsyncEventBroker[E]) AddListener(name string, listener func(E)) {
	eb.Lock()
	defer eb.Unlock()

	eb.lockedRemoveListener(name)
	eb.listenerNames = append(eb.listenerNames, name)
	eb.listenerFuncs = append(eb.listenerFuncs, listener)
}

// RemoveListener unregisters the named listener.
func (eb *AsyncEventBroker[E]) RemoveListener(name string) {
	eb.Lock()
	defer eb.Unlock()

	eb.lockedRemoveListener(name)
}

func (eb *AsyncEventBroker[E]) lockedRemoveListener(name string) {
	for i, entry := range eb.listenerNames {
		if entry == name {
			eb.listenerNames = append(eb.listenerNames[:i], eb.listenerNames[i+1:]...)
			eb.listenerFuncs = append(eb.listenerFuncs[:i], eb.listenerFuncs[i+1:]...)
			break
		}
	}
}

// AsyncTestListener returns a func that will wait for an event and return it, or timeout
// with an error.
func (eb *AsyncEventBroker[E]) AsyncTestListener(name string, capacity int) func() (*E, error) {
	// Send event down channel.
	events := make(chan E, capacity)
	eb.AddListener(name,
		func(msg E) {
			events <- msg
		})

	count := 0

	return func() (*E, error) {
		count++

		defer func() {
			if count >= capacity {
				eb.RemoveListener(name)
				close(events)
			}
		}()

		select {
		case event := <-events:
			return &event, nil

		case <-time.After(time.Second * 2):
			return nil, errors.New("timeout waiting for event")
		}
	}
}
