package message

import (
	"io"

	"github.com/jhillyerd/enmime"
	"github.com/jhillyerd/inbucket/pkg/policy"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// Manager is the interface controllers use to interact with messages.
type Manager interface {
	GetMetadata(mailbox string) ([]*Metadata, error)
	GetMessage(mailbox, id string) (*Message, error)
	PurgeMessages(mailbox string) error
	RemoveMessage(mailbox, id string) error
	SourceReader(mailbox, id string) (io.ReadCloser, error)
	MailboxForAddress(address string) (string, error)
}

// StoreManager is a message Manager backed by the storage.Store.
type StoreManager struct {
	Store storage.Store
}

// GetMetadata returns a slice of metadata for the specified mailbox.
func (s *StoreManager) GetMetadata(mailbox string) ([]*Metadata, error) {
	messages, err := s.Store.GetMessages(mailbox)
	if err != nil {
		return nil, err
	}
	metas := make([]*Metadata, len(messages))
	for i, sm := range messages {
		metas[i] = makeMetadata(sm)
	}
	return metas, nil
}

// GetMessage returns the specified message.
func (s *StoreManager) GetMessage(mailbox, id string) (*Message, error) {
	sm, err := s.Store.GetMessage(mailbox, id)
	if err != nil {
		return nil, err
	}
	r, err := sm.RawReader()
	if err != nil {
		return nil, err
	}
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, err
	}
	_ = r.Close()
	header := makeMetadata(sm)
	return &Message{Metadata: *header, Envelope: env}, nil
}

// PurgeMessages removes all messages from the specified mailbox.
func (s *StoreManager) PurgeMessages(mailbox string) error {
	return s.Store.PurgeMessages(mailbox)
}

// RemoveMessage deletes the specified message.
func (s *StoreManager) RemoveMessage(mailbox, id string) error {
	return s.Store.RemoveMessage(mailbox, id)
}

// SourceReader allows the stored message source to be read.
func (s *StoreManager) SourceReader(mailbox, id string) (io.ReadCloser, error) {
	sm, err := s.Store.GetMessage(mailbox, id)
	if err != nil {
		return nil, err
	}
	return sm.RawReader()
}

// MailboxForAddress parses an email address to return the canonical mailbox name.
func (s *StoreManager) MailboxForAddress(mailbox string) (string, error) {
	return policy.ParseMailboxName(mailbox)
}

// makeMetadata populates Metadata from a StoreMessage.
func makeMetadata(m storage.StoreMessage) *Metadata {
	return &Metadata{
		Mailbox: m.Mailbox(),
		ID:      m.ID(),
		From:    m.From(),
		To:      m.To(),
		Date:    m.Date(),
		Subject: m.Subject(),
		Size:    m.Size(),
	}
}
