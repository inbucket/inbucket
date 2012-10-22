package smtpd

import (
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
)

// Take "user+ext@host.com" and return "user", aka the mailbox we'll store it in
func ParseMailboxName(emailAddress string) (result string) {
	result = strings.ToLower(emailAddress)
	if idx := strings.Index(result, "@"); idx > -1 {
		result = result[0:idx]
	}
	if idx := strings.Index(result, "+"); idx > -1 {
		result = result[0:idx]
	}
	return result
}

// Take a mailbox name and hash it into the directory we'll store it in
func HashMailboxName(mailbox string) string {
	h := sha1.New()
	io.WriteString(h, mailbox)
	return fmt.Sprintf("%x", h.Sum(nil))
}
