// Package message contains message handling logic.
package message

import (
	"net/mail"
	"time"

	"github.com/jhillyerd/enmime"
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
