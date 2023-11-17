package test

import (
	"errors"

	"github.com/inbucket/inbucket/v3/pkg/storage"
)

// StoreStub stubs storage.Store for testing.
type StoreStub struct {
	storage.Store
	mailboxes map[string][]*MessageStub    // Stored messages, by mailbox.
	deleted   map[storage.Message]struct{} // Deleted message references.
}

// NewStore creates a new StoreStub.
func NewStore() *StoreStub {
	return &StoreStub{
		mailboxes: make(map[string][]*MessageStub),
		deleted:   make(map[storage.Message]struct{}),
	}
}

// AddMessage adds a message to the specified mailbox.
func (s *StoreStub) AddMessage(m storage.Message) (id string, err error) {
	mb := m.Mailbox()
	msgs := s.mailboxes[mb]
	s.mailboxes[mb] = append(msgs, &MessageStub{Message: m})
	return m.ID(), nil
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

	stubs := s.mailboxes[mailbox]
	msgs := make([]storage.Message, len(stubs))
	for i, stub := range stubs {
		msgs[i] = stub
	}

	return msgs, nil
}

// MarkSeen marks the message as having been seen.
func (s *StoreStub) MarkSeen(mailbox, id string) error {
	if mailbox == "messageerr" {
		return errors.New("internal error")
	}
	for _, m := range s.mailboxes[mailbox] {
		if m.ID() == id {
			m.seen = true
			return nil
		}
	}
	return storage.ErrNotExist
}

// RemoveMessage deletes a message by ID from the specified mailbox.
func (s *StoreStub) RemoveMessage(mailbox, id string) error {
	if mb, ok := s.mailboxes[mailbox]; ok {
		var removed *MessageStub
		for i, m := range mb {
			if m.ID() == id {
				removed = m
				s.mailboxes[mailbox] = append(mb[:i], mb[i+1:]...)
				break
			}
		}

		if removed != nil {
			// Clients will be looking for the original storage.Message, not our stub.
			s.deleted[removed.Message] = struct{}{}
			return nil
		}
	}

	return storage.ErrNotExist
}

// VisitMailboxes accepts a function that will be called with the messages in each mailbox while it
// continues to return true.
func (s *StoreStub) VisitMailboxes(f func([]storage.Message) (cont bool)) error {
	for _, stubs := range s.mailboxes {
		msgs := make([]storage.Message, len(stubs))
		for i, stub := range stubs {
			msgs[i] = stub
		}

		if !f(msgs) {
			return nil
		}
	}

	return nil
}

// MessageDeleted returns true if the specified message was deleted
func (s *StoreStub) MessageDeleted(m storage.Message) bool {
	_, ok := s.deleted[m]
	return ok
}

// MessageStub wraps a storage.Message with "seen" functionality.
type MessageStub struct {
	storage.Message
	seen bool
}

// Seen returns true if the message has been marked as seen previously.
func (m *MessageStub) Seen() bool {
	return m.seen
}
