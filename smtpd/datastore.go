package smtpd

import (
	"github.com/jhillyerd/go.enmime"
	"io"
	"net/mail"
	"time"
)

type DataStore interface {
	MailboxFor(emailAddress string) (Mailbox, error)
	AllMailboxes() ([]Mailbox, error)
}

type Mailbox interface {
	GetMessages() ([]Message, error)
	GetMessage(id string) (Message, error)
	Purge() error
	NewMessage() (Message, error)
	String() string
}

type Message interface {
	Id() string
	From() string
	Date() time.Time
	Subject() string
	RawReader() (reader io.ReadCloser, err error)
	ReadHeader() (msg *mail.Message, err error)
	ReadBody() (body *enmime.MIMEBody, err error)
	ReadRaw() (raw *string, err error)
	Append(data []byte) error
	Close() error
	Delete() error
	String() string
	Size() int64
}
