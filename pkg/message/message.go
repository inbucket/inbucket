// Package message contains message handling logic.
package message

import (
	"io"
	"net/mail"
	"net/textproto"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/jhillyerd/enmime/v2"
)

// Message holds both the metadata and content of a message.
type Message struct {
	event.MessageMetadata
	env *enmime.Envelope
}

// New constructs a new Message
func New(m event.MessageMetadata, e *enmime.Envelope) *Message {
	return &Message{
		MessageMetadata: m,
		env:             e,
	}
}

// Attachments returns the MIME attachments for the message.
func (m *Message) Attachments() []*enmime.Part {
	attachments := append([]*enmime.Part{}, m.env.Inlines...)
	attachments = append(attachments, m.env.Attachments...)
	return attachments
}

// Header returns the header map for this message.
func (m *Message) Header() textproto.MIMEHeader {
	return m.env.Root.Header
}

// HTML returns the HTML body of the message.
func (m *Message) HTML() string {
	return m.env.HTML
}

// MIMEErrors returns MIME parsing errors and warnings.
func (m *Message) MIMEErrors() []*enmime.Error {
	return m.env.Errors
}

// Text returns the plain text body of the message.
func (m *Message) Text() string {
	return m.env.Text
}

// Delivery is used to add a message to storage.
type Delivery struct {
	Meta   event.MessageMetadata
	Reader io.Reader
}

var _ storage.Message = &Delivery{}

// Mailbox getter.
func (d *Delivery) Mailbox() string {
	return d.Meta.Mailbox
}

// ID getter.
func (d *Delivery) ID() string {
	return d.Meta.ID
}

// From getter.
func (d *Delivery) From() *mail.Address {
	return d.Meta.From
}

// To getter.
func (d *Delivery) To() []*mail.Address {
	return d.Meta.To
}

// Date getter.
func (d *Delivery) Date() time.Time {
	return d.Meta.Date
}

// Subject getter.
func (d *Delivery) Subject() string {
	return d.Meta.Subject
}

// Size getter.
func (d *Delivery) Size() int64 {
	return d.Meta.Size
}

// Source contains the raw content of the message.
func (d *Delivery) Source() (io.ReadCloser, error) {
	return io.NopCloser(d.Reader), nil
}

// Seen getter.
func (d *Delivery) Seen() bool {
	return d.Meta.Seen
}
