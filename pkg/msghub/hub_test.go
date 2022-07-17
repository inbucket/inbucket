package msghub

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// testListener implements the Listener interface, mock for unit tests
type testListener struct {
	messages     []*Message // received messages
	wantMessages int        // how many messages this listener wants to receive
	errorAfter   int        // when != 0, messages until Receive() begins returning error

	done     chan struct{} // closed once we have received wantMessages
	overflow chan struct{} // closed if we receive wantMessages+1
}

func newTestListener(want int) *testListener {
	l := &testListener{
		messages:     make([]*Message, 0, want*2),
		wantMessages: want,
		done:         make(chan struct{}),
		overflow:     make(chan struct{}),
	}
	if want == 0 {
		close(l.done)
	}
	return l
}

// Receive a Message, store it in the messages slice, close applicable channels, and return an error
// if instructed
func (l *testListener) Receive(msg Message) error {
	l.messages = append(l.messages, &msg)
	if len(l.messages) == l.wantMessages {
		close(l.done)
	}
	if len(l.messages) == l.wantMessages+1 {
		close(l.overflow)
	}
	if l.errorAfter > 0 && len(l.messages) > l.errorAfter {
		return fmt.Errorf("Too many messages")
	}
	return nil
}

// String formats the got vs wanted message counts
func (l *testListener) String() string {
	return fmt.Sprintf("got %v messages, wanted %v", len(l.messages), l.wantMessages)
}

func TestHubNew(t *testing.T) {
	hub := New(5)
	if hub == nil {
		t.Fatal("New() == nil, expected a new Hub")
	}
}

func TestHubZeroLen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(0)
	go hub.Start(ctx)
	m := Message{}
	for i := 0; i < 100; i++ {
		hub.Dispatch(m)
	}
	// Ensures Hub doesn't panic
}

func TestHubZeroListeners(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5)
	go hub.Start(ctx)
	m := Message{}
	for i := 0; i < 100; i++ {
		hub.Dispatch(m)
	}
	// Ensures Hub doesn't panic
}

func TestHubOneListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5)
	go hub.Start(ctx)
	m := Message{}
	l := newTestListener(1)

	hub.AddListener(l)
	hub.Dispatch(m)

	// Wait for messages
	select {
	case <-l.done:
	case <-time.After(time.Second):
		t.Error("Timeout:", l)
	}
}

func TestHubRemoveListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5)
	go hub.Start(ctx)
	m := Message{}
	l := newTestListener(1)

	hub.AddListener(l)
	hub.Dispatch(m)
	hub.RemoveListener(l)
	hub.Dispatch(m)
	hub.Sync()

	// Wait for messages
	select {
	case <-l.overflow:
		t.Error(l)
	case <-time.After(50 * time.Millisecond):
		// Expected result, no overflow
	}
}

func TestHubRemoveListenerOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5)
	go hub.Start(ctx)
	m := Message{}

	// error after 1 means listener should receive 2 messages before being removed
	l := newTestListener(2)
	l.errorAfter = 1

	hub.AddListener(l)
	hub.Dispatch(m)
	hub.Dispatch(m)
	hub.Dispatch(m)
	hub.Dispatch(m)
	hub.Sync()

	// Wait for messages
	select {
	case <-l.overflow:
		t.Error(l)
	case <-time.After(50 * time.Millisecond):
		// Expected result, no overflow
	}
}

func TestHubHistoryReplay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(100)
	go hub.Start(ctx)
	l1 := newTestListener(3)
	hub.AddListener(l1)

	// Broadcast 3 messages with no listeners
	msgs := make([]Message, 3)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = Message{
			Subject: fmt.Sprintf("subj %v", i),
		}
		hub.Dispatch(msgs[i])
	}

	// Wait for messages (live)
	select {
	case <-l1.done:
	case <-time.After(time.Second):
		t.Fatal("Timeout:", l1)
	}

	// Add a new listener
	l2 := newTestListener(3)
	hub.AddListener(l2)

	// Wait for messages (history)
	select {
	case <-l2.done:
	case <-time.After(time.Second):
		t.Fatal("Timeout:", l2)
	}

	for i := 0; i < len(msgs); i++ {
		got := l2.messages[i].Subject
		want := msgs[i].Subject
		if got != want {
			t.Errorf("msg[%v].Subject == %q, want %q", i, got, want)
		}
	}
}

func TestHubHistoryReplayWrap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5)
	go hub.Start(ctx)
	l1 := newTestListener(20)
	hub.AddListener(l1)

	// Broadcast more messages than the hub can hold
	msgs := make([]Message, 20)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = Message{
			Subject: fmt.Sprintf("subj %v", i),
		}
		hub.Dispatch(msgs[i])
	}

	// Wait for messages (live)
	select {
	case <-l1.done:
	case <-time.After(time.Second):
		t.Fatal("Timeout:", l1)
	}

	// Add a new listener
	l2 := newTestListener(5)
	hub.AddListener(l2)

	// Wait for messages (history)
	select {
	case <-l2.done:
	case <-time.After(time.Second):
		t.Fatal("Timeout:", l2)
	}

	for i := 0; i < 5; i++ {
		got := l2.messages[i].Subject
		want := msgs[i+15].Subject
		if got != want {
			t.Errorf("msg[%v].Subject == %q, want %q", i, got, want)
		}
	}
}

func TestHubContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := New(5)
	go hub.Start(ctx)
	m := Message{}
	l := newTestListener(1)

	hub.AddListener(l)
	hub.Dispatch(m)
	hub.Sync()
	cancel()

	// Wait for messages
	select {
	case <-l.overflow:
		t.Error(l)
	case <-time.After(50 * time.Millisecond):
		// Expected result, no overflow
	}
}
