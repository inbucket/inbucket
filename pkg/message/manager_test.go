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
	err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte(`From: from@example.com
To: u1@example.com, u2@example.com
Subject: tsub

test email`),
	)
	require.NoError(t, err)

	assertMessageCount(t, sm, "u1@example.com", 1)
	assertMessageCount(t, sm, "u2@example.com", 1)
}

func TestDeliverStoresMessageNoFromHeader(t *testing.T) {
	sm, _ := testStoreManager()

	// Attempt to deliver a message to two mailboxes.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte(`To: u1@example.com, u2@example.com
Subject: tsub

test email`),
	)
	require.NoError(t, err)

	assertMessageCount(t, sm, "u1@example.com", 1)
	assertMessageCount(t, sm, "u2@example.com", 1)
}

func TestDeliverStoresMessageNoToHeader(t *testing.T) {
	sm, _ := testStoreManager()

	// Attempt to deliver a message to two mailboxes.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte(`From: from@example.com
Subject: tsub

test email`),
	)
	require.NoError(t, err)

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

	// Expect empty mailbox for nostore domain.
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

func TestDeliverUsesBeforeMessageStoredEventResponseMailboxes(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive event.
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			// Listener rewrites destination mailboxes.
			resp := msg
			resp.Mailboxes = []string{"new1@example.com", "new2@nostore.com"}
			return &resp
		})

	// Deliver a message to trigger event.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\r\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	// Expect messages in only the mailboxes in the event response, and for the DiscardDomains
	// policy to be ignored for nostore.com.
	assertMessageCount(t, sm, "u1@example.com", 0)
	assertMessageCount(t, sm, "u2@example.com", 0)
	assertMessageCount(t, sm, "new1@example.com", 1)
	assertMessageCount(t, sm, "new2@nostore.com", 1)
}

func TestDeliverUsesBeforeMessageStoredEventResponseMailboxesEmpty(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive event.
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			// Listener clears destination mailboxes.
			resp := msg
			resp.Mailboxes = []string{}
			return &resp
		})

	// Deliver a message to trigger event.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\r\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	// Expect no messages the mailboxes.
	assertMessageCount(t, sm, "u1@example.com", 0)
	assertMessageCount(t, sm, "u2@example.com", 0)
}

func TestDeliverUsesBeforeMessageStoredEventResponseFields(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive event.
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			// Listener rewrites destination mailboxes.
			msg.Subject = "event subj"
			msg.From = mail.Address{Address: "from@event.com", Name: "From Event"}

			// Changing To does not affect destination mailbox(es).
			msg.To = []mail.Address{
				{Address: "to@event.com", Name: "To Event"},
				{Address: "to2@event.com", Name: "To 2 Event"},
			}

			// Size is read only, should have no effect.
			msg.Size = 12345

			return &msg
		})

	// Deliver a message to trigger event.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1},
		"Received: xyz\r\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	// Verify single message stored.
	metadata, err := sm.GetMetadata("u1@example.com")
	require.NoError(t, err)
	require.Len(t, metadata, 1, "mailbox has incorrect # of messages")
	got := metadata[0]

	// Verify metadata fields were overridden by event response values.
	assert.Equal(t, "event subj", got.Subject, "Subject didn't match")
	assert.Equal(t, "from@event.com", got.From.Address, "From Address didn't match")
	assert.Equal(t, "From Event", got.From.Name, "From Name didn't match")
	require.Len(t, got.To, 2)
	assert.Equal(t, "to@event.com", got.To[0].Address, "To Address didn't match")
	assert.Equal(t, "To Event", got.To[0].Name, "To Name didn't match")
	assert.Equal(t, "to2@event.com", got.To[1].Address, "To Address didn't match")
	assert.Equal(t, "To 2 Event", got.To[1].Name, "To Name didn't match")
	assert.NotEqual(t, 12345, got.Size, "Size is read only")
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

func TestDeliverBeforeAndAfterMessageStoredEvents(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive Before event.
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			// Listener rewrites destination mailboxes.
			resp := msg
			resp.Mailboxes = []string{"new1@example.com", "new2@example.com"}
			return &resp
		})

	// After event listener.
	listener := extHost.Events.AfterMessageStored.AsyncTestListener("manager", 2)

	// Deliver a message to trigger events.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\r\n",
		[]byte("From: from@example.com\nSubject: tsub\n\ntest email"),
	); err != nil {
		t.Fatal(err)
	}

	// Confirm mailbox names overriden by Before were sent to After event.  Order is
	// not guaranteed.
	got1, err := listener()
	require.NoError(t, err)
	got2, err := listener()
	require.NoError(t, err)
	got := []string{got1.Mailbox, got2.Mailbox}
	assert.Contains(t, got, "new1@example.com")
	assert.Contains(t, got, "new2@example.com")
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
