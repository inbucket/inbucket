package policy

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/stringutil"
)

// Addressing handles email address policy.
type Addressing struct {
	Config *config.Root
}

// ExtractMailbox extracts the mailbox name from a partial email address.
func (a *Addressing) ExtractMailbox(address string) (string, error) {
	if a.Config.MailboxNaming == config.DomainNaming {
		return extractDomainMailbox(address)
	}

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
		return "", fmt.Errorf("unknown MailboxNaming value: %v", a.Config.MailboxNaming)
	}

	if domain == "" {
		return local, nil
	}

	if !ValidateDomainPart(domain) {
		return "", fmt.Errorf("domain part %q in %q failed validation", domain, address)
	}

	return local + "@" + domain, nil
}

// NewRecipient parses an address into a Recipient. This is used for parsing RCPT TO arguments,
// not To headers.
func (a *Addressing) NewRecipient(address string) (*Recipient, error) {
	local, domain, err := ParseEmailAddress(address)
	if err != nil {
		return nil, err
	}
	mailbox, err := a.ExtractMailbox(address)
	if err != nil {
		return nil, err
	}
	return &Recipient{
		Address:    mail.Address{Address: address},
		addrPolicy: a,
		LocalPart:  local,
		Domain:     domain,
		Mailbox:    mailbox,
	}, nil
}

// ParseOrigin parses an address into a Origin. This is used for parsing MAIL FROM argument,
// not To headers.
func (a *Addressing) ParseOrigin(address string) (*Origin, error) {
	local, domain, err := ParseEmailAddress(address)
	if err != nil {
		return nil, err
	}
	return &Origin{
		Address:    mail.Address{Address: address},
		addrPolicy: a,
		LocalPart:  local,
		Domain:     domain,
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

// ShouldAcceptOriginDomain indicates if Inbucket accept mail from the specified domain.
func (a *Addressing) ShouldAcceptOriginDomain(domain string) bool {
	domain = strings.ToLower(domain)
	if len(a.Config.SMTP.RejectOriginDomains) > 0 {
		for _, d := range a.Config.SMTP.RejectOriginDomains {
			if stringutil.MatchWithWildcards(d, domain) {
				return false
			}
		}
	}
	return true
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
		return "", "", errors.New("domain part validation failed")
	}
	return local, domain, nil
}

// ValidateDomainPart returns true if the domain part complies to RFC3696, RFC1035. Used by
// ParseEmailAddress().
func ValidateDomainPart(domain string) bool {
	ln := len(domain)
	if ln == 0 {
		return false
	}
	if ln > 255 {
		return false
	}
	if ln >= 4 && domain[0] == '[' && domain[ln-1] == ']' {
		// Bracketed domains must contain an IP address.
		s := 1
		if strings.HasPrefix(domain[1:], "IPv6:") {
			s = 6
		}
		ip := net.ParseIP(domain[s : ln-1])
		return ip != nil
	}

	if domain[ln-1] != '.' {
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
			if prev == '.' || prev == '-' {
				// Cannot lead with hyphen or double hyphen.
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

// Extracts the mailbox name when domain addressing is enabled.
func extractDomainMailbox(address string) (string, error) {
	var local, domain string
	var err error

	if address != "" && address[0] == '[' && address[len(address)-1] == ']' {
		// Likely an IP address in brackets, treat as domain only.
		domain = address
	} else {
		local, domain, err = parseEmailAddress(address)
		if err != nil {
			return "", err
		}
	}

	if local != "" {
		local, err = parseMailboxName(local)
		if err != nil {
			return "", err
		}
	}

	// If no @domain is specified, assume this is being used for mailbox lookup via the API.
	if domain == "" {
		domain = local
	}

	if !ValidateDomainPart(domain) {
		return "", fmt.Errorf("domain part %q in %q failed validation", domain, address)
	}

	return domain, nil
}

// parseEmailAddress unescapes an email address, and splits the local part from the domain part.  An
// error is returned if the local part fails validation following the guidelines in RFC3696. The
// domain part is optional and not validated.
func parseEmailAddress(address string) (local string, domain string, err error) {
	if address == "" {
		return "", "", errors.New("empty address")
	}
	if len(address) > 320 {
		return "", "", errors.New("address exceeds 320 characters")
	}

	// Remove forward-path routes.
	if address[0] == '@' {
		end := strings.IndexRune(address, ':')
		if end == -1 {
			return "", "", errors.New("missing terminating ':' in route specification")
		}
		address = address[end+1:]
		if address == "" {
			return "", "", errors.New("address empty after removing route specification")
		}
	}

	if address[0] == '.' {
		return "", "", errors.New("address cannot start with a period")
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
		case strings.IndexByte("!#$%&'*+-/=?^_`{|}~", c) >= 0:
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
				return "", "", errors.New("sequence of periods is not permitted")
			}
			err = buf.WriteByte(c)
			if err != nil {
				return
			}
			inCharQuote = false
		case c == '\\':
			inCharQuote = true
		case c == '"':
			switch {
			case inCharQuote:
				err = buf.WriteByte(c)
				if err != nil {
					return
				}
				inCharQuote = false
			case inStringQuote:
				inStringQuote = false
			default:
				if i == 0 {
					inStringQuote = true
				} else {
					return "", "", errors.New("quoted string can only begin at start of address")
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
					return "", "", errors.New("local part must not exceed 128 characters")
				}
				if prev == '.' {
					return "", "", errors.New("local part cannot end with a period")
				}
				domain = address[i+1:]
				break LOOP
			}
		case c > 127:
			return "", "", errors.New("characters outside of US-ASCII range not permitted")
		default:
			if inCharQuote || inStringQuote {
				err = buf.WriteByte(c)
				if err != nil {
					return
				}
				inCharQuote = false
			} else {
				return "", "", fmt.Errorf("character %q must be quoted", c)
			}
		}
		prev = c
	}
	if inCharQuote {
		return "", "", errors.New("cannot end address with unterminated quoted-pair")
	}
	if inStringQuote {
		return "", "", errors.New("cannot end address with unterminated string quote")
	}
	return buf.String(), domain, nil
}

// ParseMailboxName takes a localPart string (ex: "user+ext" without "@domain")
// and returns just the mailbox name (ex: "user").  Returns an error if
// localPart contains invalid characters; it won't accept any that must be
// quoted according to RFC3696.
func parseMailboxName(localPart string) (result string, err error) {
	if localPart == "" {
		return "", errors.New("mailbox name cannot be empty")
	}
	result = strings.ToLower(localPart)
	invalid := make([]byte, 0, 10)
	for i := 0; i < len(result); i++ {
		c := result[i]
		switch {
		case 'a' <= c && c <= 'z':
		case '0' <= c && c <= '9':
		case strings.IndexByte("!#$%&'*+-=/?^_`.{|}~", c) >= 0:
		default:
			invalid = append(invalid, c)
		}
	}
	if len(invalid) > 0 {
		return "", fmt.Errorf("mailbox name contained invalid character(s): %q", invalid)
	}
	if idx := strings.Index(result, "+"); idx > -1 {
		result = result[0:idx]
	}
	return result, nil
}
