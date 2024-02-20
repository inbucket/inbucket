package test

import (
	"bytes"
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
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StoreFactory returns a new store for the test suite.
type StoreFactory func(
	config.Storage, *extension.Host) (store storage.Store, destroy func(), err error)

// storeSuite is passed to each test function; embeds `testing.T` to provide testing primitives.
type storeSuite struct {
	*testing.T
	store   storage.Store
	extHost *extension.Host
}

// StoreSuite runs a set of general tests on the provided Store.
func StoreSuite(t *testing.T, factory StoreFactory) {
	t.Helper()
	testCases := []struct {
		name string
		test func(storeSuite)
		conf config.Storage
	}{
		{"metadata", testMetadata, config.Storage{}},
		{"content", testContent, config.Storage{}},
		{"delivery order", testDeliveryOrder, config.Storage{}},
		{"latest", testLatest, config.Storage{}},
		{"naming", testNaming, config.Storage{}},
		{"size", testSize, config.Storage{}},
		{"seen", testSeen, config.Storage{}},
		{"delete", testDelete, config.Storage{}},
		{"purge", testPurge, config.Storage{}},
		{"cap=10", testMsgCap, config.Storage{MailboxMsgCap: 10}},
		{"cap=0", testNoMsgCap, config.Storage{MailboxMsgCap: 0}},
		{"visit mailboxes", testVisitMailboxes, config.Storage{}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extHost := extension.NewHost()
			store, destroy, err := factory(tc.conf, extHost)
			if err != nil {
				t.Fatal(err)
			}
			defer destroy()

			s := storeSuite{
				T:       t,
				store:   store,
				extHost: extHost,
			}
			tc.test(s)
		})
	}
}

// testMetadata verifies message metadata is stored and retrieved correctly.
func testMetadata(s storeSuite) {
	mailbox := "testmailbox"
	from := &mail.Address{Name: "From Person", Address: "from@person.com"}
	to := []*mail.Address{
		{Name: "One Person", Address: "one@a.person.com"},
		{Name: "Two Person", Address: "two@b.person.com"},
	}
	date := time.Now()
	subject := "fantastic test subject line"
	content := "doesn't matter"
	delivery := &message.Delivery{
		Meta: event.MessageMetadata{
			// ID and Size will be determined by the Store.
			Mailbox: mailbox,
			From:    from,
			To:      to,
			Date:    date,
			Subject: subject,
			Seen:    false,
		},
		Reader: strings.NewReader(content),
	}
	id, err := s.store.AddMessage(delivery)
	if err != nil {
		s.Fatal(err)
	}
	if id == "" {
		s.Fatal("Expected AddMessage() to return non-empty ID string")
	}
	// Retrieve and validate the message.
	sm, err := s.store.GetMessage(mailbox, id)
	if err != nil {
		s.Fatal(err)
	}
	if sm.Mailbox() != mailbox {
		s.Errorf("got mailbox %q, want: %q", sm.Mailbox(), mailbox)
	}
	if sm.ID() != id {
		s.Errorf("got id %q, want: %q", sm.ID(), id)
	}
	if *sm.From() != *from {
		s.Errorf("got from %v, want: %v", sm.From(), from)
	}
	if len(sm.To()) != len(to) {
		s.Errorf("got len(to) = %v, want: %v", len(sm.To()), len(to))
	} else {
		for i, got := range sm.To() {
			if *to[i] != *got {
				s.Errorf("got to[%v] %v, want: %v", i, got, to[i])
			}
		}
	}
	if !sm.Date().Equal(date) {
		s.Errorf("got date %v, want: %v", sm.Date(), date)
	}
	if sm.Subject() != subject {
		s.Errorf("got subject %q, want: %q", sm.Subject(), subject)
	}
	if sm.Size() != int64(len(content)) {
		s.Errorf("got size %v, want: %v", sm.Size(), len(content))
	}
	if sm.Seen() {
		s.Errorf("got seen %v, want: false", sm.Seen())
	}
}

// testContent generates some binary content and makes sure it is correctly retrieved.
func testContent(s storeSuite) {
	content := make([]byte, 5000)
	for i := 0; i < len(content); i++ {
		content[i] = byte(i % 256)
	}
	mailbox := "testmailbox"
	from := &mail.Address{Name: "From Person", Address: "from@person.com"}
	to := []*mail.Address{
		{Name: "One Person", Address: "one@a.person.com"},
	}
	date := time.Now()
	subject := "fantastic test subject line"
	delivery := &message.Delivery{
		Meta: event.MessageMetadata{
			// ID and Size will be determined by the Store.
			Mailbox: mailbox,
			From:    from,
			To:      to,
			Date:    date,
			Subject: subject,
		},
		Reader: bytes.NewReader(content),
	}
	id, err := s.store.AddMessage(delivery)
	require.NoError(s, err, "AddMessage() failed")

	// Read stored message source.
	m, err := s.store.GetMessage(mailbox, id)
	require.NoError(s, err, "GetMessage() failed")
	r, err := m.Source()
	require.NoError(s, err, "Source() failed")
	got, err := io.ReadAll(r)
	require.NoError(s, err, "failed to read source")
	err = r.Close()
	require.NoError(s, err, "failed to close source reader")

	// Verify source.
	if len(got) != len(content) {
		s.Errorf("Got len(content) == %v, want: %v", len(got), len(content))
	}
	errors := 0
	for i, b := range got {
		if b != content[i] {
			s.Errorf("Got content[%v] == %v, want: %v", i, b, content[i])
			errors++
		}
		if errors > 5 {
			s.Fatalf("Too many content errors, aborting test.")
		}
	}
}

// testDeliveryOrder delivers several messages to the same mailbox, meanwhile querying its contents
// with a new GetMessages call each cycle.
func testDeliveryOrder(s storeSuite) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	for i, subj := range subjects {
		// Check mailbox count.
		GetAndCountMessages(s.T, s.store, mailbox, i)
		DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
	}
	// Confirm delivery order.
	msgs := GetAndCountMessages(s.T, s.store, mailbox, 5)
	for i, want := range subjects {
		got := msgs[i].Subject()
		if got != want {
			s.Errorf("Got subject %q, want %q", got, want)
		}
	}
}

// testLatest delivers several messages to the same mailbox, and confirms the id `latest` returns
// the last message sent.
func testLatest(s storeSuite) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	for _, subj := range subjects {
		DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
	}
	// Confirm latest.
	latest, err := s.store.GetMessage(mailbox, "latest")
	if err != nil {
		s.Fatal(err)
	}
	if latest == nil {
		s.Fatalf("Got nil message, wanted most recent message for %v.", mailbox)
	}
	got := latest.Subject()
	want := "echo"
	if got != want {
		s.Errorf("Got subject %q, want %q", got, want)
	}
}

// testNaming ensures the store does not enforce local part mailbox naming.
func testNaming(s storeSuite) {
	DeliverToStore(s.T, s.store, "fred@fish.net", "disk #27", time.Now())
	GetAndCountMessages(s.T, s.store, "fred", 0)
	GetAndCountMessages(s.T, s.store, "fred@fish.net", 1)
}

// testSize verifies message content size metadata values.
func testSize(s storeSuite) {
	mailbox := "fred"
	subjects := []string{"a", "br", "much longer than the others"}
	sentIds := make([]string, len(subjects))
	sentSizes := make([]int64, len(subjects))
	for i, subj := range subjects {
		id, size := DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
		sentIds[i] = id
		sentSizes[i] = size
	}
	for i, id := range sentIds {
		msg, err := s.store.GetMessage(mailbox, id)
		if err != nil {
			s.Fatal(err)
		}
		want := sentSizes[i]
		got := msg.Size()
		if got != want {
			s.Errorf("Got size %v, want: %v", got, want)
		}
	}
}

// testSeen verifies a message can be marked as seen.
func testSeen(s storeSuite) {
	mailbox := "lisa"
	id1, _ := DeliverToStore(s.T, s.store, mailbox, "whatever", time.Now())
	id2, _ := DeliverToStore(s.T, s.store, mailbox, "hello?", time.Now())
	// Confirm unseen.
	msg, err := s.store.GetMessage(mailbox, id1)
	if err != nil {
		s.Fatal(err)
	}
	if msg.Seen() {
		s.Errorf("got seen %v, want: false", msg.Seen())
	}
	// Mark id1 seen.
	err = s.store.MarkSeen(mailbox, id1)
	if err != nil {
		s.Fatal(err)
	}
	// Verify id1 seen.
	msg, err = s.store.GetMessage(mailbox, id1)
	if err != nil {
		s.Fatal(err)
	}
	if !msg.Seen() {
		s.Errorf("id1 got seen %v, want: true", msg.Seen())
	}
	// Verify id2 still unseen.
	msg, err = s.store.GetMessage(mailbox, id2)
	if err != nil {
		s.Fatal(err)
	}
	if msg.Seen() {
		s.Errorf("id2 got seen %v, want: false", msg.Seen())
	}
}

// testDelete creates and deletes some messages.
func testDelete(s storeSuite) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	for _, subj := range subjects {
		DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
	}
	msgs := GetAndCountMessages(s.T, s.store, mailbox, len(subjects))

	// Subscribe to events.
	eventListener := s.extHost.Events.AfterMessageDeleted.AsyncTestListener("test", 2)

	// Delete a couple messages.
	deleteIDs := []string{msgs[1].ID(), msgs[3].ID()}
	for _, id := range deleteIDs {
		err := s.store.RemoveMessage(mailbox, id)
		require.NoError(s, err)
	}

	// Confirm deletion.
	subjects = []string{"alpha", "charlie", "echo"}
	msgs = GetAndCountMessages(s.T, s.store, mailbox, len(subjects))
	for i, want := range subjects {
		got := msgs[i].Subject()
		if got != want {
			s.Errorf("Got subject %q, want %q", got, want)
		}
	}

	// Capture events and check correct IDs were emitted.
	ev1, err := eventListener()
	require.NoError(s, err)
	ev2, err := eventListener()
	require.NoError(s, err)
	eventIDs := []string{ev1.ID, ev2.ID}
	for _, id := range deleteIDs {
		assert.Contains(s, eventIDs, id)
	}

	// Try appending one more.
	DeliverToStore(s.T, s.store, mailbox, "foxtrot", time.Now())
	subjects = []string{"alpha", "charlie", "echo", "foxtrot"}
	msgs = GetAndCountMessages(s.T, s.store, mailbox, len(subjects))
	for i, want := range subjects {
		got := msgs[i].Subject()
		if got != want {
			s.Errorf("Got subject %q, want %q", got, want)
		}
	}
}

// testPurge makes sure mailboxes can be purged.
func testPurge(s storeSuite) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	// Subscribe to events.
	eventListener := s.extHost.Events.AfterMessageDeleted.AsyncTestListener("test", len(subjects))

	// Populate mailbox.
	for _, subj := range subjects {
		DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
	}
	GetAndCountMessages(s.T, s.store, mailbox, len(subjects))

	// Purge and verify.
	err := s.store.PurgeMessages(mailbox)
	require.NoError(s, err)
	GetAndCountMessages(s.T, s.store, mailbox, 0)

	// Confirm events emitted.
	gotEvents := []*event.MessageMetadata{}
	for range subjects {
		ev, err := eventListener()
		if err != nil {
			s.Error(err)
			break
		}
		gotEvents = append(gotEvents, ev)
	}
	assert.Equal(s, len(subjects), len(gotEvents),
		"expected delete event for each message in mailbox")
}

// testMsgCap verifies the message cap is enforced.
func testMsgCap(s storeSuite) {
	mbCap := 10
	mailbox := "captain"

	for i := 0; i < 20; i++ {
		subj := fmt.Sprintf("subject %v", i)
		DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
		msgs, err := s.store.GetMessages(mailbox)
		if err != nil {
			s.Fatalf("Failed to GetMessages for %q: %v", mailbox, err)
		}
		if len(msgs) > mbCap {
			s.Errorf("Mailbox has %v messages, should be capped at %v", len(msgs), mbCap)
			break
		}

		// Check that the first (oldest) message is correct.
		first := i - mbCap + 1
		if first < 0 {
			first = 0
		}
		firstSubj := fmt.Sprintf("subject %v", first)
		if firstSubj != msgs[0].Subject() {
			s.Errorf("Got subject %q, wanted first subject: %q", msgs[0].Subject(), firstSubj)
		}
	}
}

// testNoMsgCap verfies a cap of 0 is not enforced.
func testNoMsgCap(s storeSuite) {
	mailbox := "captain"
	for i := 0; i < 20; i++ {
		subj := fmt.Sprintf("subject %v", i)
		DeliverToStore(s.T, s.store, mailbox, subj, time.Now())
		GetAndCountMessages(s.T, s.store, mailbox, i+1)
	}
}

// testVisitMailboxes creates some mailboxes and confirms the VisitMailboxes method visits all of
// them.
func testVisitMailboxes(s storeSuite) {
	// Deliver 2 test messages to each of 5 mailboxes.
	boxes := []string{"abby", "bill", "christa", "donald", "evelyn"}
	for _, name := range boxes {
		DeliverToStore(s.T, s.store, name, "Old Message", time.Now().Add(-24*time.Hour))
		DeliverToStore(s.T, s.store, name, "New Message", time.Now())
	}

	// Verify message and mailbox counts.
	nboxes := 0
	err := s.store.VisitMailboxes(func(messages []storage.Message) bool {
		nboxes++
		name := "unknown"
		if len(messages) > 0 {
			name = messages[0].Mailbox()
		}

		assert.Len(s, messages, 2, "incorrect message count in mailbox %s", name)
		return true
	})
	require.NoError(s, err, "VisitMailboxes() failed")
	assert.Equal(s, 5, nboxes, "visited %v mailboxes, want: 5", nboxes)
}
