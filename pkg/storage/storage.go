// Package storage contains implementation independent datastore logic
package storage

import (
	"errors"
	"fmt"
	"io"
	"net/mail"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
)

var (
	// ErrNotExist indicates the requested message does not exist.
	ErrNotExist = errors.New("message does not exist")

	// ErrNotWritable indicates the message is closed; no longer writable
	ErrNotWritable = errors.New("Message not writable")

	// Constructors tracks registered storage constructors
	Constructors = make(map[string]func(config.Storage, *extension.Host) (Store, error))
)

// Store is the interface Inbucket uses to interact with storage implementations.
type Store interface {
	// AddMessage stores the message, message ID and Size will be ignored.
	AddMessage(message Message) (id string, err error)
	GetMessage(mailbox, id string) (Message, error)
	GetMessages(mailbox string) ([]Message, error)
	MarkSeen(mailbox, id string) error
	PurgeMessages(mailbox string) error
	RemoveMessage(mailbox, id string) error
	VisitMailboxes(f func([]Message) (cont bool)) error
}

// Message represents a message to be stored, or returned from a storage implementation.
type Message interface {
	Mailbox() string
	ID() string
	From() *mail.Address
	To() []*mail.Address
	Date() time.Time
	Subject() string
	Source() (io.ReadCloser, error)
	Size() int64
	Seen() bool
}

// FromConfig creates an instance of the Store based on the provided configuration.
func FromConfig(c config.Storage, extHost *extension.Host) (store Store, err error) {
	if cf := Constructors[c.Type]; cf != nil {
		return cf(c, extHost)
	}
	return nil, fmt.Errorf("unknown storage type configured: %q", c.Type)
}
