package extension

import (
	"sync"
)

// EventBroker maintains a list of listeners interested in a specific type
// of event.
type EventBroker[E any, R interface{}] struct {
	sync.RWMutex
	listenerNames []string     // Ordered listener names.
	listenerFuncs []func(E) *R // Ordered listener functions.
}

// Emit sends the provided event to each registered listener in order, until
// one returns a non-nil result.  That result will be returned to the caller.
func (eb *EventBroker[E, R]) Emit(event *E) *R {
	eb.RLock()
	defer eb.RUnlock()

	for _, l := range eb.listenerFuncs {
		// Events are copied to minimize the risk of mutation.
		if result := l(*event); result != nil {
			return result
		}
	}

	return nil
}

// AddListener registers the named listener, replacing one with a duplicate
// name if present.  Listeners should be added in order of priority, most
// significant first.
func (eb *EventBroker[E, R]) AddListener(name string, listener func(E) *R) {
	eb.Lock()
	defer eb.Unlock()

	eb.lockedRemoveListener(name)
	eb.listenerNames = append(eb.listenerNames, name)
	eb.listenerFuncs = append(eb.listenerFuncs, listener)
}

// RemoveListener unregisters the named listener.
func (eb *EventBroker[E, R]) RemoveListener(name string) {
	eb.Lock()
	defer eb.Unlock()

	eb.lockedRemoveListener(name)
}

func (eb *EventBroker[E, R]) lockedRemoveListener(name string) {
	for i, entry := range eb.listenerNames {
		if entry == name {
			eb.listenerNames = append(eb.listenerNames[:i], eb.listenerNames[i+1:]...)
			eb.listenerFuncs = append(eb.listenerFuncs[:i], eb.listenerFuncs[i+1:]...)
			break
		}
	}
}
