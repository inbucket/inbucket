package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// StoreFactory returns a new store for the test suite.
type StoreFactory func() (store storage.Store, destroy func(), err error)

// StoreSuite runs a set of general tests on the provided Store.
func StoreSuite(t *testing.T, factory StoreFactory) {
	testCases := []struct {
		name string
		test func(*testing.T, storage.Store)
	}{
		{"metadata", testMetadata},
		{"content", testContent},
		{"delivery order", testDeliveryOrder},
		{"size", testSize},
		{"delete", testDelete},
		{"purge", testPurge},
		{"visit mailboxes", testVisitMailboxes},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store, destroy, err := factory()
			if err != nil {
				t.Fatal(err)
			}
			tc.test(t, store)
			destroy()
		})
	}
}

// testMetadata verifies message metadata is stored and retrieved correctly.
func testMetadata(t *testing.T, store storage.Store) {
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
		Meta: message.Metadata{
			// ID and Size will be determined by the Store.
			Mailbox: mailbox,
			From:    from,
			To:      to,
			Date:    date,
			Subject: subject,
		},
		Reader: strings.NewReader(content),
	}
	id, err := store.AddMessage(delivery)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("Expected AddMessage() to return non-empty ID string")
	}
	// Retrieve and validate the message.
	sm, err := store.GetMessage(mailbox, id)
	if err != nil {
		t.Fatal(err)
	}
	if sm.Mailbox() != mailbox {
		t.Errorf("got mailbox %q, want: %q", sm.Mailbox(), mailbox)
	}
	if sm.ID() != id {
		t.Errorf("got id %q, want: %q", sm.ID(), id)
	}
	if *sm.From() != *from {
		t.Errorf("got from %v, want: %v", sm.From(), from)
	}
	if len(sm.To()) != len(to) {
		t.Errorf("got len(to) = %v, want: %v", len(sm.To()), len(to))
	} else {
		for i, got := range sm.To() {
			if *to[i] != *got {
				t.Errorf("got to[%v] %v, want: %v", i, got, to[i])
			}
		}
	}
	if !sm.Date().Equal(date) {
		t.Errorf("got date %v, want: %v", sm.Date(), date)
	}
	if sm.Subject() != subject {
		t.Errorf("got subject %q, want: %q", sm.Subject(), subject)
	}
	if sm.Size() != int64(len(content)) {
		t.Errorf("got size %v, want: %v", sm.Size(), len(content))
	}
}

// testContent generates some binary content and makes sure it is correctly retrieved.
func testContent(t *testing.T, store storage.Store) {
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
		Meta: message.Metadata{
			// ID and Size will be determined by the Store.
			Mailbox: mailbox,
			From:    from,
			To:      to,
			Date:    date,
			Subject: subject,
		},
		Reader: bytes.NewReader(content),
	}
	id, err := store.AddMessage(delivery)
	if err != nil {
		t.Fatal(err)
	}
	// Get and check.
	m, err := store.GetMessage(mailbox, id)
	if err != nil {
		t.Fatal(err)
	}
	r, err := m.Source()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(content) {
		t.Errorf("Got len(content) == %v, want: %v", len(got), len(content))
	}
	errors := 0
	for i, b := range got {
		if b != content[i] {
			t.Errorf("Got content[%v] == %v, want: %v", i, b, content[i])
			errors++
		}
		if errors > 5 {
			t.Fatalf("Too many content errors, aborting test.")
			break
		}
	}
}

// testDeliveryOrder delivers several messages to the same mailbox, meanwhile querying its contents
// with a new GetMessages call each cycle.
func testDeliveryOrder(t *testing.T, store storage.Store) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	for i, subj := range subjects {
		// Check mailbox count.
		getAndCountMessages(t, store, mailbox, i)
		deliverMessage(t, store, mailbox, subj, time.Now())
	}
	// Confirm delivery order.
	msgs := getAndCountMessages(t, store, mailbox, 5)
	for i, want := range subjects {
		got := msgs[i].Subject()
		if got != want {
			t.Errorf("Got subject %q, want %q", got, want)
		}
	}
}

// testSize verifies message contnet size metadata values.
func testSize(t *testing.T, store storage.Store) {
	mailbox := "fred"
	subjects := []string{"a", "br", "much longer than the others"}
	sentIds := make([]string, len(subjects))
	sentSizes := make([]int64, len(subjects))
	for i, subj := range subjects {
		id, size := deliverMessage(t, store, mailbox, subj, time.Now())
		sentIds[i] = id
		sentSizes[i] = size
	}
	for i, id := range sentIds {
		msg, err := store.GetMessage(mailbox, id)
		if err != nil {
			t.Fatal(err)
		}
		want := sentSizes[i]
		got := msg.Size()
		if got != want {
			t.Errorf("Got size %v, want: %v", got, want)
		}
	}
}

// testDelete creates and deletes some messages.
func testDelete(t *testing.T, store storage.Store) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	for _, subj := range subjects {
		deliverMessage(t, store, mailbox, subj, time.Now())
	}
	msgs := getAndCountMessages(t, store, mailbox, len(subjects))
	// Delete a couple messages.
	err := store.RemoveMessage(mailbox, msgs[1].ID())
	if err != nil {
		t.Fatal(err)
	}
	err = store.RemoveMessage(mailbox, msgs[3].ID())
	if err != nil {
		t.Fatal(err)
	}
	// Confirm deletion.
	subjects = []string{"alpha", "charlie", "echo"}
	msgs = getAndCountMessages(t, store, mailbox, len(subjects))
	for i, want := range subjects {
		got := msgs[i].Subject()
		if got != want {
			t.Errorf("Got subject %q, want %q", got, want)
		}
	}
	// Try appending one more.
	deliverMessage(t, store, mailbox, "foxtrot", time.Now())
	subjects = []string{"alpha", "charlie", "echo", "foxtrot"}
	msgs = getAndCountMessages(t, store, mailbox, len(subjects))
	for i, want := range subjects {
		got := msgs[i].Subject()
		if got != want {
			t.Errorf("Got subject %q, want %q", got, want)
		}
	}
}

// testPurge makes sure mailboxes can be purged.
func testPurge(t *testing.T, store storage.Store) {
	mailbox := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	for _, subj := range subjects {
		deliverMessage(t, store, mailbox, subj, time.Now())
	}
	getAndCountMessages(t, store, mailbox, len(subjects))
	// Purge and verify.
	err := store.PurgeMessages(mailbox)
	if err != nil {
		t.Fatal(err)
	}
	getAndCountMessages(t, store, mailbox, 0)
}

// testVisitMailboxes creates some mailboxes and confirms the VisitMailboxes method visits all of
// them.
func testVisitMailboxes(t *testing.T, ds storage.Store) {
	boxes := []string{"abby", "bill", "christa", "donald", "evelyn"}
	for _, name := range boxes {
		deliverMessage(t, ds, name, "Old Message", time.Now().Add(-24*time.Hour))
		deliverMessage(t, ds, name, "New Message", time.Now())
	}
	seen := 0
	err := ds.VisitMailboxes(func(messages []storage.Message) bool {
		seen++
		count := len(messages)
		if count != 2 {
			t.Errorf("got: %v messages, want: 2", count)
		}
		return true
	})
	if err != nil {
		t.Error(err)
	}
	if seen != 5 {
		t.Errorf("saw %v messages in total, want: 5", seen)
	}
}

// deliverMessage creates and delivers a message to the specific mailbox, returning the size of the
// generated message.
func deliverMessage(
	t *testing.T,
	store storage.Store,
	mailbox string,
	subject string,
	date time.Time,
) (string, int64) {
	t.Helper()
	meta := message.Metadata{
		Mailbox: mailbox,
		To:      []*mail.Address{{Name: "Some Body", Address: "somebody@host"}},
		From:    &mail.Address{Name: "Some B. Else", Address: "somebodyelse@host"},
		Subject: subject,
		Date:    date,
	}
	testMsg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\nTest Body\r\n",
		meta.To[0].Address, meta.From.Address, subject)
	delivery := &message.Delivery{
		Meta:   meta,
		Reader: ioutil.NopCloser(strings.NewReader(testMsg)),
	}
	id, err := store.AddMessage(delivery)
	if err != nil {
		t.Fatal(err)
	}
	return id, int64(len(testMsg))
}

// getAndCountMessages is a test helper that expects to receive count messages or fails the test, it
// also checks return error.
func getAndCountMessages(t *testing.T, s storage.Store, mailbox string, count int) []storage.Message {
	t.Helper()
	msgs, err := s.GetMessages(mailbox)
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mailbox, err)
	}
	if len(msgs) != count {
		t.Errorf("Got %v messages, want: %v", len(msgs), count)
	}
	return msgs
}
