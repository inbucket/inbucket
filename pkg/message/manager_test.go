package message_test

import (
	"fmt"
	"io"
	"net/mail"
	"strings"
	"testing"
	"time"

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

func TestDeliverEmitsBeforeMessageStoredEventToHeader(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive event.
	var got *event.InboundMessage
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			got = &msg
			return nil
		})

	// Deliver a message to trigger event, To header differs from RCPT TO.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	recip2, _ := sm.AddrPolicy.NewRecipient("u2@example.com")
	if err := sm.Deliver(
		origin,
		[]*policy.Recipient{recip1, recip2},
		"Received: xyz\n",
		[]byte(`From: from@example.com
To: u1@example.com, u3@external.com
Subject: tsub

test email`),
	); err != nil {
		t.Fatal(err)
	}

	require.NotNil(t, got, "BeforeMessageStored listener did not receive InboundMessage")
	assert.Equal(t, []string{"u1@example.com", "u2@example.com"}, got.Mailboxes, "Mailboxes not equal")
	assert.Equal(t, &mail.Address{Name: "", Address: "from@example.com"}, got.From, "From not equal")
	assert.Equal(t, []*mail.Address{
		{Name: "", Address: "u1@example.com"},
		{Name: "", Address: "u3@external.com"},
	}, got.To, "To not equal")
	assert.Equal(t, "tsub", got.Subject, "Subject not equal")
	assert.Equal(t, int64(84), got.Size, "Size not equal")
}

func TestDeliverEmitsBeforeMessageStoredEventRcptTo(t *testing.T) {
	sm, extHost := testStoreManager()

	// Register function to receive event.
	var got *event.InboundMessage
	extHost.Events.BeforeMessageStored.AddListener(
		"test",
		func(msg event.InboundMessage) *event.InboundMessage {
			got = &msg
			return nil
		})

	// Deliver a message to trigger event, lacks To header.
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
	assert.Equal(t, &mail.Address{Name: "", Address: "from@example.com"}, got.From, "From not equal")
	assert.Equal(t, []*mail.Address{
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
			msg.From = &mail.Address{Address: "from@event.com", Name: "From Event"}

			// Changing To does not affect destination mailbox(es).
			msg.To = []*mail.Address{
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
		[]byte("From: from@example.com\nSubject: events\n\ntest email."),
	); err != nil {
		t.Fatal(err)
	}

	got, err := listener()
	require.NoError(t, err)
	assert.NotNil(t, got, "No event received, or it was nil")
	assertMessageCount(t, sm, "to@example.com", 1)

	// Verify event content.
	assert.Equal(t, "to@example.com", got.Mailbox)
	assert.Equal(t, "from@example.com", got.From.Address)

	assert.WithinDuration(t, time.Now(), got.Date, 5*time.Second)
	assert.Equal(t, "events", got.Subject, nil)
	assert.Equal(t, int64(51), got.Size)

	require.Len(t, got.To, 1)
	assert.Equal(t, "to@example.com", got.To[0].Address)
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

	// Confirm mailbox names overridden by `Before` were sent to `After` event.  Order is
	// not guaranteed.
	got1, err := listener()
	require.NoError(t, err)
	got2, err := listener()
	require.NoError(t, err)
	got := []string{got1.Mailbox, got2.Mailbox}
	assert.Contains(t, got, "new1@example.com")
	assert.Contains(t, got, "new2@example.com")
}

func TestGetMessage(t *testing.T) {
	sm, _ := testStoreManager()

	// Add a test message.
	subject := "getMessage1"
	id := addTestMessage(sm, "get-box", subject)

	// Verify retrieval of the test message.
	msg, err := sm.GetMessage("get-box", id)
	require.NoError(t, err, "GetMessage must succeed")
	require.NotNil(t, msg, "GetMessage must return a result")
	assert.Equal(t, subject, msg.Subject)
	assert.Contains(t, msg.Text(), fmt.Sprintf("about %q", subject))
}

func TestMarkSeen(t *testing.T) {
	sm, _ := testStoreManager()

	// Add a test message.
	subject := "getMessage1"
	id := addTestMessage(sm, "seen-box", subject)

	// Verify test message unseen.
	msg, err := sm.GetMessage("seen-box", id)
	require.NoError(t, err, "GetMessage must succeed")
	require.NotNil(t, msg, "GetMessage must return a result")
	assert.False(t, msg.Seen, "msg should be unseen")

	err = sm.MarkSeen("seen-box", id)
	require.NoError(t, err, "MarkSeen should succeed")

	// Verify test message seen.
	msg, err = sm.GetMessage("seen-box", id)
	require.NoError(t, err, "GetMessage must succeed")
	require.NotNil(t, msg, "GetMessage must return a result")
	assert.True(t, msg.Seen, "msg should have been seen")
}

func TestRemoveMessage(t *testing.T) {
	sm, _ := testStoreManager()

	// Add test messages.
	id1 := addTestMessage(sm, "rm-box", "subject 1")
	id2 := addTestMessage(sm, "rm-box", "subject 2")
	id3 := addTestMessage(sm, "rm-box", "subject 3")
	got, err := sm.GetMetadata("rm-box")
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Delete message 2 and verify.
	err = sm.RemoveMessage("rm-box", id2)
	require.NoError(t, err)
	got, err = sm.GetMetadata("rm-box")
	require.NoError(t, err)
	require.Len(t, got, 2, "Should be 2 messages remaining")

	gotIDs := make([]string, 0, 3)
	for _, msg := range got {
		gotIDs = append(gotIDs, msg.ID)
	}
	assert.Contains(t, gotIDs, id1)
	assert.Contains(t, gotIDs, id3)
}

func TestPurgeMessages(t *testing.T) {
	sm, _ := testStoreManager()

	// Add test messages.
	_ = addTestMessage(sm, "purge-box", "subject 1")
	_ = addTestMessage(sm, "purge-box", "subject 2")
	_ = addTestMessage(sm, "purge-box", "subject 3")
	got, err := sm.GetMetadata("purge-box")
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Purge and verify.
	err = sm.PurgeMessages("purge-box")
	require.NoError(t, err)
	got, err = sm.GetMetadata("purge-box")
	require.NoError(t, err)
	assert.Empty(t, got, "Purge should remove all mailbox messages")
}

func TestSourceReader(t *testing.T) {
	sm, _ := testStoreManager()

	recvdHeader := "Received: xyz\n"
	msgSource := `From: from@example.com
To: u1@example.com, u2@example.com
Subject: tsub

test email`

	// Deliver mesage.
	origin, _ := sm.AddrPolicy.ParseOrigin("from@example.com")
	recip1, _ := sm.AddrPolicy.NewRecipient("u1@example.com")
	err := sm.Deliver(origin, []*policy.Recipient{recip1}, recvdHeader, []byte(msgSource))
	require.NoError(t, err)

	// Find message ID.
	msgs, err := sm.GetMetadata("u1@example.com")
	require.NoError(t, err, "Failed to read mailbox")
	require.Len(t, msgs, 1, "Unexpected mailbox len")
	id := msgs[0].ID

	// Read back and verify source.
	r, err := sm.SourceReader("u1@example.com", id)
	require.NoError(t, err, "SourceReader must succeed")
	gotBytes, err := io.ReadAll(r)
	require.NoError(t, err, "Failed to read source")

	got := string(gotBytes)
	assert.Contains(t, got, recvdHeader, "Source should contain received header")
	assert.Contains(t, got, msgSource, "Source should contain original message source")
}

func TestMailboxForAddress(t *testing.T) {
	// Configured for FullNaming.
	sm, _ := testStoreManager()

	addr := "u1@example.com"
	got, err := sm.MailboxForAddress(addr)
	require.NoError(t, err)

	assert.Equal(t, addr, got, "FullNaming mode should return a full address for mailbox")
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

// Adds a test message to the provided store, returning the new message ID.
func addTestMessage(sm *message.StoreManager, mailbox string, subject string) string {
	from := mail.Address{Name: "From Test", Address: "from@example.com"}
	to := mail.Address{Name: "To Test", Address: "to@example.com"}
	delivery := &message.Delivery{
		Meta: event.MessageMetadata{
			Mailbox: mailbox,
			From:    &from,
			To:      []*mail.Address{&to},
			Date:    time.Now(),
			Subject: subject,
		},
		Reader: strings.NewReader(fmt.Sprintf(
			"From: %s\nTo: %s\nSubject: %s\n\nTest message about %q\n",
			from, to, subject, subject,
		)),
	}

	id, err := sm.Store.AddMessage(delivery)
	if err != nil {
		panic(err)
	}

	return id
}

func assertMessageCount(t *testing.T, sm *message.StoreManager, mailbox string, count int) {
	t.Helper()

	metas, err := sm.GetMetadata(mailbox)
	require.NoError(t, err, "StoreManager GetMetadata failed")

	got := len(metas)
	if got != count {
		t.Errorf("Mailbox %q got %v messages, wanted %v", mailbox, got, count)
	}
}
