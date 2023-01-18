package message_test

import (
	"testing"

	"github.com/inbucket/inbucket/pkg/extension"
	"github.com/inbucket/inbucket/pkg/extension/event"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/inbucket/inbucket/pkg/test"
	"github.com/stretchr/testify/assert"
)

func TestManagerEmitsMessageStoredEvent(t *testing.T) {
	extHost := extension.NewHost()
	sm := &message.StoreManager{
		AddrPolicy: &policy.Addressing{},
		Store:      test.NewStore(),
		ExtHost:    extHost,
	}

	// Capture message event.
	var got *event.MessageMetadata
	extHost.Events.MessageStored.AddListener(
		"test",
		func(msg event.MessageMetadata) *extension.Void {
			got = &msg
			return nil
		})

	// Attempt to deliver a message to generate event.
	if _, err := sm.Deliver(
		&policy.Recipient{},
		"from@example.com",
		[]*policy.Recipient{},
		"prefix",
		[]byte("From: from@example.com\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, got, "No event received, or it was nil")
}
