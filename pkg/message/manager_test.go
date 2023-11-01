package message_test

import (
	"net/mail"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliverStoresMessages(t *testing.T) {
	sm, _ := testStoreManager()

	// Attempt to deliver a message to two mailboxes.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	assertMessageCount(t, sm, "u1@example.com", 1)
	assertMessageCount(t, sm, "u2@example.com", 1)
}

func TestDeliverRespectsRecipientPolicy(t *testing.T) {
	sm, _ := testStoreManager()

	// Attempt to deliver a message to two mailboxes.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@nostore.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	assertMessageCount(t, sm, "u1@nostore.com", 0)
	assertMessageCount(t, sm, "u2@example.com", 1)
}

func TestDeliverEmitsBeforeMessageStoredEvent(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive event.
	var got *event.InboundMessage
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			got = &msg
			return nil
		})

	// Deliver a message to trigger event.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	require.NotNil(t, got, "BeforeMessageStored listener did not receive InboundMessage")
	assert.Equal(t, []string{"u1@example.com", "u2@example.com"}, got.Mailboxes, "Mailboxes not equal")
	assert.Equal(t, mail.Address{Name: "", Address: "from@example.com"}, got.From, "From not equal")
	assert.Equal(t, []mail.Address{
		{Name: "", Address: "u1@example.com"},
		{Name: "", Address: "u2@example.com"},
	}, got.To, "To not equal")
	assert.Equal(t, "tsub", got.Subject, "Subject not equal")
	assert.Equal(t, int64(48), got.Size, "Size not equal")
}

func TestDeliverEmitsAfterMessageStoredEvent(t *testing.T) {
	sm, extHost := testStoreManager()

	listener := extHost.Events.AfterMessageStored.AsyncTestListener("manager", 1)

	// Deliver a message to trigger event.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip, _ := sm.AddrPolicy.NewRecipient("to@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip},
		"Received: xyz\n",
		[]byte("From: from@example.com\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	got, err := listener()
	require.NoError(t, err)
	assert.NotNil(t, got, "No event received, or it was nil")
	assertMessageCount(t, sm, "to@example.com", 1)
}

// Returns an empty StoreManager and extension Host pair, configured for testing.
func testStoreManager() (*message.StoreManager, *extension.Host) {
	extHost := extension.NewHost()

	sm := &message.StoreManager{
		AddrPolicy: &policy.Addressing{
			Config: &config.Root{
				MailboxNaming: config.FullNaming,
				SMTP: config.SMTP{
					DefaultAccept:  true,
					DefaultStore:   true,
					RejectDomains:  []string{"noaccept.com"},
					DiscardDomains: []string{"nostore.com"},
				},
			},
		},
		Store:   test.NewStore(),
		ExtHost: extHost,
	}

	return sm, extHost
}

func assertMessageCount(t *testing.T, sm *message.StoreManager, mailbox string, count int) {
	t.Helper()

	metas, err := sm.GetMetadata(mailbox)
	assert.NoError(t, err, "StoreManager GetMetadata failed")

	got := len(metas)
	if got != count {
		t.Errorf("Mailbox %q got %v messages, wanted %v", mailbox, got, count)
	}
}
