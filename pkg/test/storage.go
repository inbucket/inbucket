package test

import (
	"errors"

	"github.com/jhillyerd/inbucket/pkg/storage"
)

// StoreStub stubs storage.Store for testing.
type StoreStub struct {
	storage.Store
	mailboxes map[string][]storage.Message
}

// NewStore creates a new StoreStub.
func NewStore() *StoreStub {
	return &StoreStub{
		mailboxes: make(map[string][]storage.Message),
	}
}

// AddMessage adds a message to the specified mailbox.
func (s *StoreStub) AddMessage(mailbox string, m storage.Message) {
	msgs := s.mailboxes[mailbox]
	s.mailboxes[mailbox] = append(msgs, m)
}

// GetMessage gets a message by ID from the specified mailbox.
func (s *StoreStub) GetMessage(mailbox, id string) (storage.Message, error) {
	if mailbox == "messageerr" {
		return nil, errors.New("internal error")
	}
	for _, m := range s.mailboxes[mailbox] {
		if m.ID() == id {
			return m, nil
		}
	}
	return nil, storage.ErrNotExist
}

// GetMessages gets all the messages for the specified mailbox.
func (s *StoreStub) GetMessages(mailbox string) ([]storage.Message, error) {
	if mailbox == "messageserr" {
		return nil, errors.New("internal error")
	}
	return s.mailboxes[mailbox], nil
}
