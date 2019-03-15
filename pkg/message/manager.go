package message

import (
	"bytes"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/inbucket/inbucket/pkg/msghub"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/stringutil"
	"github.com/jhillyerd/enmime"
	"github.com/rs/zerolog/log"
)

// Manager is the interface controllers use to interact with messages.
type Manager interface {
	Deliver(
		to *policy.Recipient,
		from string,
		recipients []*policy.Recipient,
		prefix string,
		content []byte,
	) (id string, err error)
	GetMetadata(mailbox string) ([]*Metadata, error)
	GetMessage(mailbox, id string) (*Message, error)
	MarkSeen(mailbox, id string) error
	PurgeMessages(mailbox string) error
	RemoveMessage(mailbox, id string) error
	SourceReader(mailbox, id string) (io.ReadCloser, error)
	MailboxForAddress(address string) (string, error)
}

// StoreManager is a message Manager backed by the storage.Store.
type StoreManager struct {
	AddrPolicy *policy.Addressing
	Store      storage.Store
	Hub        *msghub.Hub
}

// Deliver submits a new message to the store.
func (s *StoreManager) Deliver(
	to *policy.Recipient,
	from string,
	recipients []*policy.Recipient,
	prefix string,
	source []byte,
) (string, error) {
	// TODO enmime is too heavy for this step, only need header.
	// Go's header parsing isn't good enough, so this is blocked on enmime issue #64.
	env, err := enmime.ReadEnvelope(bytes.NewReader(source))
	if err != nil {
		return "", err
	}
	fromaddr, err := env.AddressList("From")
	if err != nil || len(fromaddr) == 0 {
		fromaddr = []*mail.Address{{Address: from}}
	}
	toaddr, err := env.AddressList("To")
	if err != nil {
		toaddr = make([]*mail.Address, len(recipients))
		for i, torecip := range recipients {
			toaddr[i] = &torecip.Address
		}
	}
	log.Debug().Str("module", "message").Str("mailbox", to.Mailbox).Msg("Delivering message")
	delivery := &Delivery{
		Meta: Metadata{
			Mailbox: to.Mailbox,
			From:    fromaddr[0],
			To:      toaddr,
			Date:    time.Now(),
			Subject: env.GetHeader("Subject"),
		},
		Reader: io.MultiReader(strings.NewReader(prefix), bytes.NewReader(source)),
	}
	id, err := s.Store.AddMessage(delivery)
	if err != nil {
		return "", err
	}
	if s.Hub != nil {
		// Broadcast message information.
		broadcast := msghub.Message{
			Mailbox: to.Mailbox,
			ID:      id,
			From:    stringutil.StringAddress(delivery.From()),
			To:      stringutil.StringAddressList(delivery.To()),
			Subject: delivery.Subject(),
			Date:    delivery.Date(),
			Size:    delivery.Size(),
		}
		s.Hub.Dispatch(broadcast)
	}
	return id, nil
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
	if err != nil || sm == nil {
		return nil, err
	}
	r, err := sm.Source()
	if err != nil {
		return nil, err
	}
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, err
	}
	_ = r.Close()
	header := makeMetadata(sm)
	return &Message{Metadata: *header, env: env}, nil
}

// MarkSeen marks the message as having been read.
func (s *StoreManager) MarkSeen(mailbox, id string) error {
	log.Debug().Str("module", "manager").Str("mailbox", mailbox).Str("id", id).
		Msg("Marking as seen")
	return s.Store.MarkSeen(mailbox, id)
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
	if err != nil || sm == nil {
		return nil, err
	}
	return sm.Source()
}

// MailboxForAddress parses an email address to return the canonical mailbox name.
func (s *StoreManager) MailboxForAddress(mailbox string) (string, error) {
	return s.AddrPolicy.ExtractMailbox(mailbox)
}

// makeMetadata populates Metadata from a storage.Message.
func makeMetadata(m storage.Message) *Metadata {
	return &Metadata{
		Mailbox: m.Mailbox(),
		ID:      m.ID(),
		From:    m.From(),
		To:      m.To(),
		Date:    m.Date(),
		Subject: m.Subject(),
		Size:    m.Size(),
		Seen:    m.Seen(),
	}
}
