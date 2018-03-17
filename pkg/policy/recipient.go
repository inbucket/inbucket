package policy

import "net/mail"

// Recipient represents a potential email recipient, allows policies for it to be queried.
type Recipient struct {
	mail.Address
	apolicy *Addressing
	// LocalPart is the part of the address before @, including +extension.
	LocalPart string
	// Domain is the part of the address after @.
	Domain string
	// Mailbox is the canonical mailbox name for this recipient.
	Mailbox string
}

// ShouldAccept returns true if Inbucket should accept mail for this recipient.
func (r *Recipient) ShouldAccept() bool {
	return true
}

// ShouldStore returns true if Inbucket should store mail for this recipient.
func (r *Recipient) ShouldStore() bool {
	return r.apolicy.ShouldStoreDomain(r.Domain)
}
