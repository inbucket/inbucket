package message

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/jhillyerd/enmime/v2"
	"github.com/rs/zerolog/log"
)

// recvdTimeFmt to use in generated Received header.
const recvdTimeFmt = "Mon, 02 Jan 2006 15:04:05 -0700 (MST)"

// Manager is the interface controllers use to interact with messages.
type Manager interface {
	Deliver(
		from *policy.Origin,
		recipients []*policy.Recipient,
		recvdHeader string,
		content []byte,
	) error
	GetMetadata(mailbox string) ([]*event.MessageMetadata, error)
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
	ExtHost    *extension.Host
}

// Deliver submits a new message to the store.
func (s *StoreManager) Deliver(
	from *policy.Origin,
	recipients []*policy.Recipient,
	recvdHeader string,
	source []byte,
) error {
	logger := log.With().Str("module", "message").Logger()

	// Parse envelope headers.
	header, err := enmime.DecodeHeaders(source)
	if err != nil {
		return err
	}

	fromAddrs, err := enmime.ParseAddressList(header.Get("From"))
	if err != nil || len(fromAddrs) == 0 {
		// Failed to parse From header, use SMTP MAIL FROM instead.
		fromAddrs = make([]*mail.Address, 1)
		fromAddrs[0] = &from.Address
	}

	toAddrs, err := enmime.ParseAddressList(header.Get("To"))
	if err != nil {
		// Failed to parse To header, use SMTP RCPT TO instead.
		toAddrs = make([]*mail.Address, len(recipients))
		for i, torecip := range recipients {
			toAddrs[i] = &torecip.Address
		}
	}

	subject := header.Get("Subject")
	now := time.Now()
	tstamp := now.UTC().Format(recvdTimeFmt)

	// Process inbound message through extensions.
	mailboxes := make([]string, 0, len(recipients))
	for _, recip := range recipients {
		mailboxes = append(mailboxes, recip.Mailbox)
	}

	// Construct InboundMessage event and process through extensions.
	inbound := &event.InboundMessage{
		Mailboxes: mailboxes,
		From:      fromAddrs[0],
		To:        toAddrs,
		Subject:   subject,
		Size:      int64(len(source)),
	}

	extResult := s.ExtHost.Events.BeforeMessageStored.Emit(inbound)
	if extResult == nil {
		// Use address policy to determine deliverable mailboxes.
		mailboxes = mailboxes[:0]
		for _, recip := range recipients {
			if recip.ShouldStore() {
				mailboxes = append(mailboxes, recip.Mailbox)
			}
		}
		inbound.Mailboxes = mailboxes
	} else {
		// Event response overrides destination mailboxes and address policy.
		inbound = extResult
	}

	// Deliver to each mailbox.
	for _, mb := range inbound.Mailboxes {
		// Append recipient and timestamp to generated Received header.
		recvd := fmt.Sprintf("%s  for <%s>; %s\r\n", recvdHeader, mb, tstamp)

		// Deliver message.
		logger.Debug().Str("mailbox", mb).Msg("Delivering message")
		delivery := &Delivery{
			Meta: event.MessageMetadata{
				Mailbox: mb,
				From:    inbound.From,
				To:      inbound.To,
				Date:    now,
				Subject: inbound.Subject,
				Size:    inbound.Size,
			},
			Reader: io.MultiReader(strings.NewReader(recvd), bytes.NewReader(source)),
		}
		id, err := s.Store.AddMessage(delivery)
		if err != nil {
			logger.Error().Str("mailbox", mb).Err(err).Msg("Delivery failed")
			return err
		}

		// Emit message stored event.
		event := delivery.Meta
		event.ID = id
		s.ExtHost.Events.AfterMessageStored.Emit(&event)
	}

	return nil
}

// GetMetadata returns a slice of metadata for the specified mailbox.
func (s *StoreManager) GetMetadata(mailbox string) ([]*event.MessageMetadata, error) {
	messages, err := s.Store.GetMessages(mailbox)
	if err != nil {
		return nil, err
	}
	metas := make([]*event.MessageMetadata, len(messages))
	for i, sm := range messages {
		metas[i] = MakeMetadata(sm)
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
	header := MakeMetadata(sm)
	return &Message{MessageMetadata: *header, env: env}, nil
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

// MakeMetadata populates Metadata from a storage.Message.
func MakeMetadata(m storage.Message) *event.MessageMetadata {
	return &event.MessageMetadata{
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
