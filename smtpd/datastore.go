package smtpd

import (
	"errors"
	"io"
	"net/mail"
	"time"

	"github.com/jhillyerd/go.enmime"
)

var (
	// ErrNotExist indicates the requested message does not exist
	ErrNotExist = errors.New("Message does not exist")

	// ErrNotWritable indicates the message is closed; no longer writable
	ErrNotWritable = errors.New("Message not writable")
)

// DataStore is an interface to get Mailboxes stored in Inbucket
type DataStore interface {
	MailboxFor(emailAddress string) (Mailbox, error)
	AllMailboxes() ([]Mailbox, error)
}

// Mailbox is an interface to get and manipulate messages in a DataStore
type Mailbox interface {
	GetMessages() ([]Message, error)
	GetMessage(id string) (Message, error)
	Purge() error
	NewMessage() (Message, error)
	String() string
}

// Message is an interface for a single message in a Mailbox
type Message interface {
	ID() string
	From() string
	To() string
	Date() time.Time
	Subject() string
	RawReader() (reader io.ReadCloser, err error)
	ReadHeader() (msg *mail.Message, err error)
	ReadBody() (body *enmime.MIMEBody, err error)
	ReadRaw() (raw *string, err error)
	Append(data []byte) error
	Close() error
	Delete() error
	String() string
	Size() int64
}
