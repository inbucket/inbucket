package test

import (
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// StoreSuite runs a set of general tests on the provided Store
func StoreSuite(t *testing.T, store storage.Store) {
	testCases := []struct {
		name string
		test func(*testing.T, storage.Store)
	}{
		{"metadata", testMetadata},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t, store)
		})
	}
}

func testMetadata(t *testing.T, ds storage.Store) {
	// Store a message
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
			// ID and Size will be determined by the Store
			Mailbox: mailbox,
			From:    from,
			To:      to,
			Date:    date,
			Subject: subject,
		},
		Reader: strings.NewReader(content),
	}
	id, err := ds.AddMessage(delivery)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("Expected AddMessage() to return non-empty ID string")
	}
	// Retrieve and validate the message
	sm, err := ds.GetMessage(mailbox, id)
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
