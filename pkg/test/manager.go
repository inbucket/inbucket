package test

import (
	"errors"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/storage"
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
func (m *ManagerStub) GetMetadata(mailbox string) ([]*event.MessageMetadata, error) {
	if mailbox == "messageserr" {
		return nil, errors.New("internal error")
	}
	messages := m.mailboxes[mailbox]
	metas := make([]*event.MessageMetadata, len(messages))
	for i, msg := range messages {
		metas[i] = &msg.MessageMetadata
	}
	return metas, nil
}

// MailboxForAddress invokes policy.ParseMailboxName.
func (m *ManagerStub) MailboxForAddress(address string) (string, error) {
	addrPolicy := &policy.Addressing{Config: &config.Root{
		MailboxNaming: config.FullNaming,
	}}
	return addrPolicy.ExtractMailbox(address)
}

// MarkSeen marks a message as having been read.
func (m *ManagerStub) MarkSeen(mailbox, id string) error {
	if mailbox == "messageerr" {
		return errors.New("internal error")
	}
	for _, msg := range m.mailboxes[mailbox] {
		if msg.ID == id {
			msg.MessageMetadata.Seen = true
			return nil
		}
	}
	return storage.ErrNotExist
}
