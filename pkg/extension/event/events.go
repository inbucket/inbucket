package event

import (
	"net/mail"
	"time"
)

// AddressParts contains the local and domain parts of an email address.
type AddressParts struct {
	Local  string
	Domain string
}

// InboundMessage contains the basic header and mailbox data for a message being received.
type InboundMessage struct {
	Mailboxes []string
	From      mail.Address
	To        []mail.Address
	Subject   string
	Size      int64
}

// MessageMetadata contains the basic header data for a message event.
type MessageMetadata struct {
	Mailbox string
	ID      string
	From    *mail.Address
	To      []*mail.Address
	Date    time.Time
	Subject string
	Size    int64
	Seen    bool
}
