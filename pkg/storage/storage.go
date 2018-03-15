// Package storage contains implementation independent datastore logic
package storage

import (
	"errors"
	"io"
	"net/mail"
	"sync"
	"time"
)

var (
	// ErrNotExist indicates the requested message does not exist.
	ErrNotExist = errors.New("message does not exist")

	// ErrNotWritable indicates the message is closed; no longer writable
	ErrNotWritable = errors.New("Message not writable")
)

// Store is the interface Inbucket uses to interact with storage implementations.
type Store interface {
	// AddMessage stores the message, message ID and Size will be ignored.
	AddMessage(message StoreMessage) (id string, err error)
	GetMessage(mailbox, id string) (StoreMessage, error)
	GetMessages(mailbox string) ([]StoreMessage, error)
	PurgeMessages(mailbox string) error
	RemoveMessage(mailbox, id string) error
	VisitMailboxes(f func([]StoreMessage) (cont bool)) error
	// LockFor is a temporary hack to fix #77 until Datastore revamp
	LockFor(emailAddress string) (*sync.RWMutex, error)
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
	Size() int64
}
