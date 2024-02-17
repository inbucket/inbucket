package test

import (
	"fmt"
	"io"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/storage"
)

// DeliverToStore creates and delivers a message to the specific mailbox, returning the size of the
// generated message.
func DeliverToStore(
	t *testing.T,
	store storage.Store,
	mailbox string,
	subject string,
	date time.Time,
) (string, int64) {
	t.Helper()
	meta := event.MessageMetadata{
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
		Reader: io.NopCloser(strings.NewReader(testMsg)),
	}

	id, err := store.AddMessage(delivery)
	if err != nil {
		t.Fatal(err)
	}

	return id, int64(len(testMsg))
}

// GetAndCountMessages is a test helper that expects to receive count messages or fails the test, it
// also checks return error.
func GetAndCountMessages(t *testing.T, s storage.Store, mailbox string, count int) []storage.Message {
	t.Helper()
	msgs, err := s.GetMessages(mailbox)
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mailbox, err)
	}
	if len(msgs) != count {
		t.Errorf("Got %v messages for %q, want: %v", len(msgs), mailbox, count)
	}

	return msgs
}
