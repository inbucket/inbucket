package file

import (
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

// Message implements Message and contains a little bit of data about a
// particular email message, and methods to retrieve the rest of it from disk.
type Message struct {
	mailbox *mbox
	// Stored in GOB
	Fid      string
	Fdate    time.Time
	Ffrom    *mail.Address
	Fto      []*mail.Address
	Fsubject string
	Fsize    int64
	Fseen    bool
}

// newMessage creates a new FileMessage object and sets the Date and ID fields.
// It will also delete messages over messageCap if configured.
func (mb *mbox) newMessage() (*Message, error) {
	// Load index
	if !mb.indexLoaded {
		if err := mb.readIndex(); err != nil {
			return nil, err
		}
	}
	// Delete old messages over messageCap
	if mb.store.messageCap > 0 {
		for len(mb.messages) >= mb.store.messageCap {
			log.Info().Str("module", "storage").Str("mailbox", mb.name).
				Msg("Mailbox over message cap")
			id := mb.messages[0].ID()
			if err := mb.removeMessage(id); err != nil {
				log.Error().Str("module", "storage").Str("mailbox", mb.name).Str("id", id).
					Err(err).Msg("Unable to delete message")
			}
		}
	}
	date := time.Now()
	id := generateID(date)
	return &Message{mailbox: mb, Fid: id, Fdate: date}, nil
}

// Mailbox returns the name of the mailbox this message resides in.
func (m *Message) Mailbox() string {
	return m.mailbox.name
}

// ID gets the ID of the Message
func (m *Message) ID() string {
	return m.Fid
}

// Date returns the date/time this Message was received by Inbucket
func (m *Message) Date() time.Time {
	return m.Fdate
}

// From returns the value of the Message From header
func (m *Message) From() *mail.Address {
	return m.Ffrom
}

// To returns the value of the Message To header
func (m *Message) To() []*mail.Address {
	return m.Fto
}

// Subject returns the value of the Message Subject header
func (m *Message) Subject() string {
	return m.Fsubject
}

// Size returns the size of the Message on disk in bytes
func (m *Message) Size() int64 {
	return m.Fsize
}

func (m *Message) rawPath() string {
	return filepath.Join(m.mailbox.path, m.Fid+".raw")
}

// Source opens the .raw portion of a Message as an io.ReadCloser
func (m *Message) Source() (reader io.ReadCloser, err error) {
	file, err := os.Open(m.rawPath())
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Seen returns the seen flag value.
func (m *Message) Seen() bool {
	return m.Fseen
}
