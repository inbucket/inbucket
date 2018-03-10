// Package storage contains implementation independent datastore logic
package storage

import (
	"errors"
	"io"
	"net/mail"
	"sync"
	"time"

	"github.com/jhillyerd/enmime"
)

var (
	// ErrNotExist indicates the requested message does not exist
	ErrNotExist = errors.New("Message does not exist")

	// ErrNotWritable indicates the message is closed; no longer writable
	ErrNotWritable = errors.New("Message not writable")
)

// Store is an interface to get Mailboxes stored in Inbucket
type Store interface {
	MailboxFor(emailAddress string) (Mailbox, error)
	AllMailboxes() ([]Mailbox, error)
	// LockFor is a temporary hack to fix #77 until Datastore revamp
	LockFor(emailAddress string) (*sync.RWMutex, error)
}

// Mailbox is an interface to get and manipulate messages in a DataStore
type Mailbox interface {
	GetMessages() ([]Message, error)
	GetMessage(id string) (Message, error)
	Purge() error
	NewMessage() (Message, error)
	Name() string
	String() string
}

// Message is an interface for a single message in a Mailbox
type Message interface {
	ID() string
	From() string
	To() []string
	Date() time.Time
	Subject() string
	RawReader() (reader io.ReadCloser, err error)
	ReadHeader() (msg *mail.Message, err error)
	ReadBody() (body *enmime.Envelope, err error)
	ReadRaw() (raw *string, err error)
	Append(data []byte) error
	Close() error
	Delete() error
	String() string
	Size() int64
}
