package event

import (
	"net/mail"
	"time"
)

// MessageMetadata contains the basic header data for a message event.
type MessageMetadata struct {
	Mailbox string
	ID      string
	From    *mail.Address
	To      []*mail.Address
	Date    time.Time
	Subject string
	Size    int64
}
