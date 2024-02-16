package extension_test

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/extension"
)

func TestBrokerEmitCallsOneListener(t *testing.T) {
	broker := &extension.EventBroker[string, bool]{}

	// Setup listener.
	var got string
	listener := func(s string) *bool {
		got = s
		return nil
	}
	broker.AddListener("x", listener)

	want := "bacon"
	broker.Emit(&want)
	if got != want {
		t.Errorf("Emit got %q, want %q", got, want)
	}
}

func TestBrokerEmitCallsMultipleListeners(t *testing.T) {
	broker := &extension.EventBroker[string, bool]{}

	// Setup listeners.
	var firstGot, secondGot string
	first := func(s string) *bool {
		firstGot = s
		return nil
	}
	second := func(s string) *bool {
		secondGot = s
		return nil
	}

	broker.AddListener("1", first)
	broker.AddListener("2", second)

	want := "hi"
	broker.Emit(&want)
	if firstGot != want {
		t.Errorf("first got %q, want %q", firstGot, want)
	}
	if secondGot != want {
		t.Errorf("second got %q, want %q", secondGot, want)
	}
}

func TestBrokerEmitCapturesFirstResult(t *testing.T) {
	broker := &extension.EventBroker[struct{}, string]{}

	// Setup listeners.
	makeListener := func(result *string) func(struct{}) *string {
		return func(s struct{}) *string { return result }
	}
	first := "first"
	second := "second"
	broker.AddListener("0", makeListener(nil))
	broker.AddListener("1", makeListener(&first))
	broker.AddListener("2", makeListener(&second))

	want := first
	got := broker.Emit(&struct{}{})
	if got == nil {
		t.Errorf("Emit got nil, want %q", want)
	} else if *got != want {
		t.Errorf("Emit got %q, want %q", *got, want)
	}
}

func TestBrokerAddingDuplicateNameReplacesPrevious(t *testing.T) {
	broker := &extension.EventBroker[string, bool]{}

	// Setup listeners.
	var firstGot, secondGot string
	first := func(s string) *bool {
		firstGot = s
		return nil
	}
	second := func(s string) *bool {
		secondGot = s
		return nil
	}

	broker.AddListener("dup", first)
	broker.AddListener("dup", second)

	want := "hi"
	broker.Emit(&want)
	if firstGot != "" {
		t.Errorf("first got %q, want empty string", firstGot)
	}
	if secondGot != want {
		t.Errorf("second got %q, want %q", secondGot, want)
	}
}

func TestBrokerRemovingListenerSuccessful(t *testing.T) {
	broker := &extension.EventBroker[string, bool]{}

	// Setup listeners.
	var firstGot, secondGot string
	first := func(s string) *bool {
		firstGot = s
		return nil
	}
	second := func(s string) *bool {
		secondGot = s
		return nil
	}

	broker.AddListener("1", first)
	broker.AddListener("2", second)
	broker.RemoveListener("1")

	want := "hi"
	broker.Emit(&want)
	if firstGot != "" {
		t.Errorf("first got %q, want empty string", firstGot)
	}
	if secondGot != want {
		t.Errorf("second got %q, want %q", secondGot, want)
	}
}

func TestBrokerRemovingMissingListener(t *testing.T) {
	broker := &extension.EventBroker[string, bool]{}
	broker.RemoveListener("doesn't crash")
}
