package extension_test

import (
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple smoke test without using AsyncTestListener.
func TestAsyncBrokerEmitCallsOneListener(t *testing.T) {
	broker := &extension.AsyncEventBroker[string]{}

	// Setup listener.
	events := make(chan string, 1)
	listener := func(s string) {
		events <- s
	}
	broker.AddListener("x", listener)

	want := "bacon"
	broker.Emit(&want)

	var got string
	select {
	case event := <-events:
		got = event

	case <-time.After(time.Second * 2):
		t.Fatal("Timeout waiting for event")
	}

	if got != want {
		t.Errorf("Emit got %q, want %q", got, want)
	}
}

func TestAsyncBrokerEmitCallsMultipleListeners(t *testing.T) {
	broker := &extension.AsyncEventBroker[string]{}

	// Setup listeners.
	first := broker.AsyncTestListener("first", 1)
	second := broker.AsyncTestListener("second", 1)

	want := "hi"
	broker.Emit(&want)

	first_got, err := first()
	require.NoError(t, err)
	assert.Equal(t, want, *first_got)

	second_got, err := second()
	require.NoError(t, err)
	assert.Equal(t, want, *second_got)
}

func TestAsyncBrokerAddingDuplicateNameReplacesPrevious(t *testing.T) {
	broker := &extension.AsyncEventBroker[string]{}

	// Setup listeners.
	first := broker.AsyncTestListener("dup", 1)
	second := broker.AsyncTestListener("dup", 1)

	want := "hi"
	broker.Emit(&want)

	first_got, err := first()
	require.Error(t, err)
	assert.Nil(t, first_got)

	second_got, err := second()
	require.NoError(t, err)
	assert.Equal(t, want, *second_got)
}

func TestAsyncBrokerRemovingListenerSuccessful(t *testing.T) {
	broker := &extension.AsyncEventBroker[string]{}

	// Setup listeners.
	first := broker.AsyncTestListener("1", 1)
	second := broker.AsyncTestListener("2", 1)
	broker.RemoveListener("1")

	want := "hi"
	broker.Emit(&want)

	first_got, err := first()
	require.Error(t, err)
	assert.Nil(t, first_got)

	second_got, err := second()
	require.NoError(t, err)
	assert.Equal(t, want, *second_got)
}

func TestAsyncBrokerRemovingMissingListener(t *testing.T) {
	broker := &extension.AsyncEventBroker[string]{}
	broker.RemoveListener("doesn't crash")
}
