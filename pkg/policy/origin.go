package policy

import (
	"net/mail"
)

// Origin represents a potential email origin, allows policies for it to be queried.
type Origin struct {
	mail.Address
	addrPolicy *Addressing
	// LocalPart is the part of the address before @, including +extension.
	LocalPart string
	// Domain is the part of the address after @.
	Domain string
}

// ShouldAccept returns true if Inbucket should accept mail from this origin.
func (o *Origin) ShouldAccept() bool {
	return o.addrPolicy.ShouldAcceptOriginDomain(o.Domain)
}
