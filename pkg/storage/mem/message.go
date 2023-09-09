package mem

import (
	"bytes"
	"container/list"
	"io"
	"net/mail"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/storage"
)

// Message is a memory store message.
type Message struct {
	index   int
	mailbox string
	id      string
	from    *mail.Address
	to      []*mail.Address
	date    time.Time
	subject string
	source  []byte
	seen    bool
	el      *list.Element // This message in Store.messages
}

var _ storage.Message = &Message{}

// Mailbox returns the mailbox name.
func (m *Message) Mailbox() string { return m.mailbox }

// ID the message ID.
func (m *Message) ID() string { return m.id }

// From returns the from address.
func (m *Message) From() *mail.Address { return m.from }

// To returns the to address list.
func (m *Message) To() []*mail.Address { return m.to }

// Date returns the date received.
func (m *Message) Date() time.Time { return m.date }

// Subject returns the subject line.
func (m *Message) Subject() string { return m.subject }

// Source returns a reader for the message source.
func (m *Message) Source() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.source)), nil
}

// Size returns the message size in bytes.
func (m *Message) Size() int64 { return int64(len(m.source)) }

// Seen returns the message seen flag.
func (m *Message) Seen() bool { return m.seen }
