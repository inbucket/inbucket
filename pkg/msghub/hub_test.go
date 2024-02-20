package msghub

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testListener implements the Listener interface, mock for unit tests
type testListener struct {
	messages   []*event.MessageMetadata // received messages
	deletes    []string                 // received deletes
	wantEvents int                      // how many events this listener wants to receive
	errorAfter int                      // when != 0, event count until Receive() begins returning error
	gotEvents  int

	done     chan struct{} // closed once we have received wantMessages
	overflow chan struct{} // closed if we receive wantMessages+1
}

func newTestListener(want int) *testListener {
	l := &testListener{
		messages:   make([]*event.MessageMetadata, 0, want*2),
		deletes:    make([]string, 0, want*2),
		wantEvents: want,
		done:       make(chan struct{}),
		overflow:   make(chan struct{}),
	}
	if want == 0 {
		close(l.done)
	}
	return l
}

// Receive a Message, store it in the messages slice, close applicable channels, and return an error
// if instructed
func (l *testListener) Receive(msg event.MessageMetadata) error {
	l.gotEvents++
	l.messages = append(l.messages, &msg)
	if l.gotEvents == l.wantEvents {
		close(l.done)
	}
	if l.gotEvents == l.wantEvents+1 {
		close(l.overflow)
	}
	if l.errorAfter > 0 && l.gotEvents > l.errorAfter {
		return errors.New("too many messages")
	}
	return nil
}

func (l *testListener) Delete(mailbox string, id string) error {
	l.gotEvents++
	l.deletes = append(l.deletes, mailbox+"/"+id)
	return nil
}

// String formats the got vs wanted message counts
func (l *testListener) String() string {
	return fmt.Sprintf("got %v messages, wanted %v", len(l.messages), l.wantEvents)
}

func TestHubNew(t *testing.T) {
	hub := New(5, extension.NewHost())
	if hub == nil {
		t.Fatal("New() == nil, expected a new Hub")
	}
}

func TestHubZeroLen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(0, extension.NewHost())
	go hub.Start(ctx)
	m := event.MessageMetadata{}
	for i := 0; i < 100; i++ {
		hub.Dispatch(m)
	}
	// Ensures Hub doesn't panic
}

func TestHubZeroListeners(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5, extension.NewHost())
	go hub.Start(ctx)
	m := event.MessageMetadata{}
	for i := 0; i < 100; i++ {
		hub.Dispatch(m)
	}
	// Ensures Hub doesn't panic
}

func TestHubOneListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5, extension.NewHost())
	go hub.Start(ctx)
	m := event.MessageMetadata{}
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
	hub := New(5, extension.NewHost())
	go hub.Start(ctx)
	m := event.MessageMetadata{}
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
	hub := New(5, extension.NewHost())
	go hub.Start(ctx)
	m := event.MessageMetadata{}

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
	hub := New(100, extension.NewHost())
	go hub.Start(ctx)
	l1 := newTestListener(3)
	hub.AddListener(l1)

	// Broadcast 3 messages with no listeners
	msgs := make([]event.MessageMetadata, 3)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = event.MessageMetadata{
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

func TestHubHistoryDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(100, extension.NewHost())
	go hub.Start(ctx)
	l1 := newTestListener(3)
	hub.AddListener(l1)

	// Broadcast 3 messages with no listeners
	msgs := make([]event.MessageMetadata, 3)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = event.MessageMetadata{
			Mailbox: "hub",
			ID:      strconv.Itoa(i),
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

	hub.Delete("hub", "1") // Delete a message
	hub.Delete("zzz", "0") // Attempt to delete non-existent mailbox message

	// Add a new listener, waits for 2 messages
	l2 := newTestListener(2)
	hub.AddListener(l2)

	// Wait for messages (history)
	select {
	case <-l2.done:
	case <-time.After(time.Second):
		t.Fatal("Timeout:", l2)
	}

	want := []string{"subj 0", "subj 2"}
	for i := 0; i < len(want); i++ {
		got := l2.messages[i].Subject
		if got != want[i] {
			t.Errorf("msg[%v].Subject == %q, want %q", i, got, want[i])
		}
	}
}

func TestHubHistoryReplayWrap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(5, extension.NewHost())
	go hub.Start(ctx)
	l1 := newTestListener(20)
	hub.AddListener(l1)

	// Broadcast more messages than the hub can hold
	msgs := make([]event.MessageMetadata, 20)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = event.MessageMetadata{
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

func TestHubHistoryReplayWrapAfterDelete(t *testing.T) {
	bufferSize := 5

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := New(bufferSize, extension.NewHost())
	go hub.Start(ctx)

	waitForMessages := func(n int) {
		l := newTestListener(n)
		hub.AddListener(l)

		select {
		case <-l.done:
		case <-time.After(time.Second):
			t.Fatal("Timeout:", l)
		}
	}

	// Broadcast more messages than the hub can hold.
	msgs := make([]event.MessageMetadata, 10)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = event.MessageMetadata{
			Mailbox: "first",
			ID:      strconv.Itoa(i),
			Subject: fmt.Sprintf("subj %v", i),
		}
		hub.Dispatch(msgs[i])
	}
	waitForMessages(bufferSize)

	// Buffer must be configured size.
	require.Equal(t, bufferSize, hub.history.Len())

	// Delete a message still present in buffer.
	hub.Delete("first", "7")

	// Broadcast another set of messages.
	for i := 0; i < len(msgs); i++ {
		msgs[i] = event.MessageMetadata{
			Mailbox: "second",
			ID:      strconv.Itoa(i),
			Subject: fmt.Sprintf("subj %v", i),
		}
		hub.Dispatch(msgs[i])
	}
	waitForMessages(bufferSize)

	// Ensure the buffer did not shrink after delete.
	got := hub.history.Len()
	assert.Equal(t, bufferSize, got, "got buffer size %d, wanted %d", got, bufferSize)
}

func TestHubContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := New(5, extension.NewHost())
	go hub.Start(ctx)
	m := event.MessageMetadata{}
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
