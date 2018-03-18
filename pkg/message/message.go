// Package message contains message handling logic.
package message

import (
	"io"
	"io/ioutil"
	"net/mail"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// Metadata holds information about a message, but not the content.
type Metadata struct {
	Mailbox string
	ID      string
	From    *mail.Address
	To      []*mail.Address
	Date    time.Time
	Subject string
	Size    int64
}

// Message holds both the metadata and content of a message.
type Message struct {
	Metadata
	Envelope *enmime.Envelope
}

// Delivery is used to add a message to storage.
type Delivery struct {
	Meta   Metadata
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
	return ioutil.NopCloser(d.Reader), nil
}
