package event

import (
	"net/mail"
	"net/textproto"
	"time"
)

const (
	// ActionDefer defers decision to built-in Inbucket logic.
	ActionDefer = iota
	// ActionAllow explicitly allows this event.
	ActionAllow
	// ActionDeny explicitly deny this event, typically with specified SMTP error.
	ActionDeny
)

// AddressParts contains the local and domain parts of an email address.
type AddressParts struct {
	Local  string
	Domain string
}

// InboundMessage contains the basic header and mailbox data for a message being received.
type InboundMessage struct {
	Mailboxes []string
	From      *mail.Address
	To        []*mail.Address
	Subject   string
	Size      int64
	// Headers contains the decoded message headers: the default set (Date,
	// Subject, Sender, From, To, CC, BCC) plus any additional names registered
	// by a Lua script via inbucket.config.decode_headers.
	//
	// The same map is shared with every BeforeMessageStored listener; mutations
	// made by a Go listener are visible to listeners that run after it.  Sender
	// supplied values are untrusted: headers such as Authentication-Results and
	// X-Spam-Status should only drive decisions when the deployment controls
	// their provenance.
	Headers textproto.MIMEHeader
}

// MessageMetadata contains the basic header data for a message event.
type MessageMetadata struct {
	Mailbox string
	ID      string
	From    *mail.Address
	To      []*mail.Address
	Date    time.Time
	Subject string
	Size    int64
	Seen    bool
}

// SMTPResponse describes the response to an SMTP policy check.
type SMTPResponse struct {
	Action    int    // ActionDefer, ActionAllow, etc.
	ErrorCode int    // SMTP error code to respond with on deny.
	ErrorMsg  string // SMTP error message to respond with on deny.
}

// SMTPSession captures SMTP `MAIL FROM` & `RCPT TO` values prior to mail DATA being received.
type SMTPSession struct {
	From       *mail.Address
	To         []*mail.Address
	RemoteAddr string
}
