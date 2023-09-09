package message_test

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerEmitsMessageStoredEvent(t *testing.T) {
	extHost := extension.NewHost()
	sm := &message.StoreManager{
		AddrPolicy: &policy.Addressing{},
		Store:      test.NewStore(),
		ExtHost:    extHost,
	}

	listener := extHost.Events.AfterMessageStored.AsyncTestListener("manager", 1)

	// Attempt to deliver a message to generate event.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	if _, err := sm.Deliver(
		&policy.Recipient{},
		origin,
		[]*policy.Recipient{},
		"prefix",
		[]byte("From: from@example.com\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	got, err := listener()
	require.NoError(t, err)
	assert.NotNil(t, got, "No event received, or it was nil")
}
