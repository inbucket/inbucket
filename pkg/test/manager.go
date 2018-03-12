package test

import (
	"errors"

	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// ManagerStub is a test stub for message.Manager
type ManagerStub struct {
	message.Manager
	mailboxes map[string][]*message.Message
}

// NewManager creates a new ManagerStub.
func NewManager() *ManagerStub {
	return &ManagerStub{
		mailboxes: make(map[string][]*message.Message),
	}
}

// AddMessage adds a message to the specified mailbox.
func (m *ManagerStub) AddMessage(mailbox string, msg *message.Message) {
	messages := m.mailboxes[mailbox]
	m.mailboxes[mailbox] = append(messages, msg)
}

// GetMessage gets a message by ID from the specified mailbox.
func (m *ManagerStub) GetMessage(mailbox, id string) (*message.Message, error) {
	if mailbox == "messageerr" {
		return nil, errors.New("internal error")
	}
	for _, msg := range m.mailboxes[mailbox] {
		if msg.ID == id {
			return msg, nil
		}
	}
	return nil, storage.ErrNotExist
}

// GetMetadata gets all the metadata for the specified mailbox.
func (m *ManagerStub) GetMetadata(mailbox string) ([]*message.Metadata, error) {
	if mailbox == "messageserr" {
		return nil, errors.New("internal error")
	}
	messages := m.mailboxes[mailbox]
	metas := make([]*message.Metadata, len(messages))
	for i, msg := range messages {
		metas[i] = &msg.Metadata
	}
	return metas, nil
}
