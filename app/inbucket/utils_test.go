package inbucket

import (
	"github.com/stretchrcom/testify/assert"
	"testing"
)

func TestParseMailboxName(t *testing.T) {
	in, out := "MailBOX", "mailbox"
	if x := ParseMailboxName(in); x != out {
		t.Errorf("ParseMailboxName(%v) = %v, want %v", in, x, out)
	}

	in, out = "MailBox@Host.Com", "mailbox"
	if x := ParseMailboxName(in); x != out {
		t.Errorf("ParseMailboxName(%v) = %v, want %v", in, x, out)
	}

	in, out = "Mail+extra@Host.Com", "mail"
	if x := ParseMailboxName(in); x != out {
		t.Errorf("ParseMailboxName(%v) = %v, want %v", in, x, out)
	}
}

func TestHashMailboxName(t *testing.T) {
	in, out := "mail", "1d6e1cf70ec6f9ab28d3ea4b27a49a77654d370e"
	if x := HashMailboxName(in); x != out {
		t.Errorf("HashMailboxName(%v) = %v, want %v", in, x, out)
	}
}

func TestTextToHtml(t *testing.T) {
	// Identity
	assert.Equal(t, TextToHtml("html"), "html")

	// Check it escapes
	assert.Equal(t, TextToHtml("<html>"), "&lt;html&gt;")

	// Check for linebreaks
	assert.Equal(t, TextToHtml("line\nbreak"), "line<br/>\nbreak")
	assert.Equal(t, TextToHtml("line\r\nbreak"), "line<br/>\nbreak")
	assert.Equal(t, TextToHtml("line\rbreak"), "line<br/>\nbreak")
}

