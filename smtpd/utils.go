package smtpd

import (
	"crypto/sha1"
	"fmt"
	"html"
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

// TextToHtml takes plain text, escapes it and tries to pretty it up for
// HTML display
func TextToHtml(text string) string {
	text = html.EscapeString(text)
	replacer := strings.NewReplacer("\r\n", "<br/>\n", "\r", "<br/>\n", "\n", "<br/>\n")
	return replacer.Replace(text)
}
