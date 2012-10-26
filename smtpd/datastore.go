package smtpd

import (
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
	NewMessage() Message
	String() string
}

type Message interface {
	Id() string
	From() string
	Date() time.Time
	Subject() string
	ReadHeader() (msg *mail.Message, err error)
	ReadBody() (msg *mail.Message, body *MIMEBody, err error)
	ReadRaw() (raw *string, err error)
	Append(data []byte) error
	Close() error
	Delete() error
	String() string
}
