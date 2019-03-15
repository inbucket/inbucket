package policy

import (
	"bytes"
	"fmt"
	"net/mail"
	"strings"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/stringutil"
)

// Addressing handles email address policy.
type Addressing struct {
	Config *config.Root
}

// ExtractMailbox extracts the mailbox name from a partial email address.
func (a *Addressing) ExtractMailbox(address string) (string, error) {
	local, domain, err := parseEmailAddress(address)
	if err != nil {
		return "", err
	}
	local, err = parseMailboxName(local)
	if err != nil {
		return "", err
	}
	if a.Config.MailboxNaming == config.LocalNaming {
		return local, nil
	}
	if a.Config.MailboxNaming != config.FullNaming {
		return "", fmt.Errorf("Unknown MailboxNaming value: %v", a.Config.MailboxNaming)
	}
	if domain == "" {
		return local, nil
	}
	if !ValidateDomainPart(domain) {
		return "", fmt.Errorf("Domain part %q in %q failed validation", domain, address)
	}
	return local + "@" + domain, nil
}

// NewRecipient parses an address into a Recipient.
func (a *Addressing) NewRecipient(address string) (*Recipient, error) {
	local, domain, err := ParseEmailAddress(address)
	if err != nil {
		return nil, err
	}
	mailbox, err := a.ExtractMailbox(address)
	if err != nil {
		return nil, err
	}
	ar, err := mail.ParseAddress(address)
	if err != nil {
		return nil, err
	}
	return &Recipient{
		Address:    *ar,
		addrPolicy: a,
		LocalPart:  local,
		Domain:     domain,
		Mailbox:    mailbox,
	}, nil
}

// ShouldAcceptDomain indicates if Inbucket accepts mail destined for the specified domain.
func (a *Addressing) ShouldAcceptDomain(domain string) bool {
	domain = strings.ToLower(domain)
	if a.Config.SMTP.DefaultAccept &&
		!stringutil.SliceContains(a.Config.SMTP.RejectDomains, domain) {
		return true
	}
	if !a.Config.SMTP.DefaultAccept &&
		stringutil.SliceContains(a.Config.SMTP.AcceptDomains, domain) {
		return true
	}
	return false
}

// ShouldStoreDomain indicates if Inbucket stores mail destined for the specified domain.
func (a *Addressing) ShouldStoreDomain(domain string) bool {
	domain = strings.ToLower(domain)
	if a.Config.SMTP.DefaultStore &&
		!stringutil.SliceContains(a.Config.SMTP.DiscardDomains, domain) {
		return true
	}
	if !a.Config.SMTP.DefaultStore &&
		stringutil.SliceContains(a.Config.SMTP.StoreDomains, domain) {
		return true
	}
	return false
}

// ParseEmailAddress unescapes an email address, and splits the local part from the domain part.
// An error is returned if the local or domain parts fail validation following the guidelines
// in RFC3696.
func ParseEmailAddress(address string) (local string, domain string, err error) {
	local, domain, err = parseEmailAddress(address)
	if err != nil {
		return "", "", err
	}
	if !ValidateDomainPart(domain) {
		return "", "", fmt.Errorf("Domain part validation failed")
	}
	return local, domain, nil
}

// ValidateDomainPart returns true if the domain part complies to RFC3696, RFC1035. Used by
// ParseEmailAddress().
func ValidateDomainPart(domain string) bool {
	if len(domain) == 0 {
		return false
	}
	if len(domain) > 255 {
		return false
	}
	if domain[len(domain)-1] != '.' {
		domain += "."
	}
	prev := '.'
	labelLen := 0
	hasAlphaNum := false
	for _, c := range domain {
		switch {
		case ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') ||
			('0' <= c && c <= '9') || c == '_':
			// Must contain some of these to be a valid label.
			hasAlphaNum = true
			labelLen++
		case c == '-':
			if prev == '.' {
				// Cannot lead with hyphen.
				return false
			}
		case c == '.':
			if prev == '.' || prev == '-' {
				// Cannot end with hyphen or double-dot.
				return false
			}
			if labelLen > 63 {
				return false
			}
			if !hasAlphaNum {
				return false
			}
			labelLen = 0
			hasAlphaNum = false
		default:
			// Unknown character.
			return false
		}
		prev = c
	}
	return true
}

// parseEmailAddress unescapes an email address, and splits the local part from the domain part.  An
// error is returned if the local part fails validation following the guidelines in RFC3696. The
// domain part is optional and not validated.
func parseEmailAddress(address string) (local string, domain string, err error) {
	if address == "" {
		return "", "", fmt.Errorf("empty address")
	}
	if len(address) > 320 {
		return "", "", fmt.Errorf("address exceeds 320 characters")
	}
	if address[0] == '@' {
		return "", "", fmt.Errorf("address cannot start with @ symbol")
	}
	if address[0] == '.' {
		return "", "", fmt.Errorf("address cannot start with a period")
	}
	// Loop over address parsing out local part.
	buf := new(bytes.Buffer)
	prev := byte('.')
	inCharQuote := false
	inStringQuote := false
LOOP:
	for i := 0; i < len(address); i++ {
		c := address[i]
		switch {
		case ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z'):
			// Letters are OK.
			err = buf.WriteByte(c)
			if err != nil {
				return
			}
			inCharQuote = false
		case '0' <= c && c <= '9':
			// Numbers are OK.
			err = buf.WriteByte(c)
			if err != nil {
				return
			}
			inCharQuote = false
		case bytes.IndexByte([]byte("!#$%&'*+-/=?^_`{|}~"), c) >= 0:
			// These specials can be used unquoted.
			err = buf.WriteByte(c)
			if err != nil {
				return
			}
			inCharQuote = false
		case c == '.':
			// A single period is OK.
			if prev == '.' {
				// Sequence of periods is not permitted.
				return "", "", fmt.Errorf("Sequence of periods is not permitted")
			}
			err = buf.WriteByte(c)
			if err != nil {
				return
			}
			inCharQuote = false
		case c == '\\':
			inCharQuote = true
		case c == '"':
			if inCharQuote {
				err = buf.WriteByte(c)
				if err != nil {
					return
				}
				inCharQuote = false
			} else if inStringQuote {
				inStringQuote = false
			} else {
				if i == 0 {
					inStringQuote = true
				} else {
					return "", "", fmt.Errorf("Quoted string can only begin at start of address")
				}
			}
		case c == '@':
			if inCharQuote || inStringQuote {
				err = buf.WriteByte(c)
				if err != nil {
					return
				}
				inCharQuote = false
			} else {
				// End of local-part.
				if i > 128 {
					return "", "", fmt.Errorf("Local part must not exceed 128 characters")
				}
				if prev == '.' {
					return "", "", fmt.Errorf("Local part cannot end with a period")
				}
				domain = address[i+1:]
				break LOOP
			}
		case c > 127:
			return "", "", fmt.Errorf("Characters outside of US-ASCII range not permitted")
		default:
			if inCharQuote || inStringQuote {
				err = buf.WriteByte(c)
				if err != nil {
					return
				}
				inCharQuote = false
			} else {
				return "", "", fmt.Errorf("Character %q must be quoted", c)
			}
		}
		prev = c
	}
	if inCharQuote {
		return "", "", fmt.Errorf("Cannot end address with unterminated quoted-pair")
	}
	if inStringQuote {
		return "", "", fmt.Errorf("Cannot end address with unterminated string quote")
	}
	return buf.String(), domain, nil
}

// ParseMailboxName takes a localPart string (ex: "user+ext" without "@domain")
// and returns just the mailbox name (ex: "user").  Returns an error if
// localPart contains invalid characters; it won't accept any that must be
// quoted according to RFC3696.
func parseMailboxName(localPart string) (result string, err error) {
	if localPart == "" {
		return "", fmt.Errorf("Mailbox name cannot be empty")
	}
	result = strings.ToLower(localPart)
	invalid := make([]byte, 0, 10)
	for i := 0; i < len(result); i++ {
		c := result[i]
		switch {
		case 'a' <= c && c <= 'z':
		case '0' <= c && c <= '9':
		case bytes.IndexByte([]byte("!#$%&'*+-=/?^_`.{|}~"), c) >= 0:
		default:
			invalid = append(invalid, c)
		}
	}
	if len(invalid) > 0 {
		return "", fmt.Errorf("Mailbox name contained invalid character(s): %q", invalid)
	}
	if idx := strings.Index(result, "+"); idx > -1 {
		result = result[0:idx]
	}
	return result, nil
}
