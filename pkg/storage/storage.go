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
	// ErrNotExist indicates the requested message does not exist.
	ErrNotExist = errors.New("message does not exist")

	// ErrNotWritable indicates the message is closed; no longer writable
	ErrNotWritable = errors.New("Message not writable")
)

// Store is the interface Inbucket uses to interact with storage implementations.
type Store interface {
	GetMessage(mailbox, id string) (StoreMessage, error)
	GetMessages(mailbox string) ([]StoreMessage, error)
	PurgeMessages(mailbox string) error
	RemoveMessage(mailbox, id string) error
	VisitMailboxes(f func([]StoreMessage) (cont bool)) error
	// LockFor is a temporary hack to fix #77 until Datastore revamp
	LockFor(emailAddress string) (*sync.RWMutex, error)
	// NewMessage is temproary until #69 MessageData refactor
	NewMessage(mailbox string) (StoreMessage, error)
}

// StoreMessage represents a message to be stored, or returned from a storage implementation.
type StoreMessage interface {
	Mailbox() string
	ID() string
	From() *mail.Address
	To() []*mail.Address
	Date() time.Time
	Subject() string
	RawReader() (reader io.ReadCloser, err error)
	ReadHeader() (msg *mail.Message, err error)
	ReadBody() (body *enmime.Envelope, err error)
	ReadRaw() (raw *string, err error)
	Append(data []byte) error
	Close() error
	String() string
	Size() int64
}
