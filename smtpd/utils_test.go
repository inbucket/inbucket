package smtpd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseMailboxName(t *testing.T) {
	assert.Equal(t, ParseMailboxName("MailBOX"), "mailbox")
	assert.Equal(t, ParseMailboxName("MailBox@Host.Com"), "mailbox")
	assert.Equal(t, ParseMailboxName("Mail+extra@Host.Com"), "mail")
}

func TestHashMailboxName(t *testing.T) {
	assert.Equal(t, HashMailboxName("mail"), "1d6e1cf70ec6f9ab28d3ea4b27a49a77654d370e")
}
