package smtpd

import (
	"github.com/stretchr/testify/assert"
	"strings"
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

func TestValidateDomain(t *testing.T) {
	assert.True(t, ValidateDomainPart("jhillyerd.github.com"),
		"Simple domain failed")
	assert.False(t, ValidateDomainPart(""), "Empty domain is not valid")
	assert.False(t, ValidateDomainPart(strings.Repeat("a", 256)),
		"Max domain length is 255")
	assert.False(t, ValidateDomainPart(strings.Repeat("a", 64)+".com"),
		"Max label length is 63")
	assert.True(t, ValidateDomainPart(strings.Repeat("a", 63)+".com"),
		"Should allow 63 char label")

	var testTable = []struct {
		input  string
		expect bool
		msg    string
	}{
		{"hostname", true, "Just a hostname is valid"},
		{"github.com", true, "Two labels should be just fine"},
		{"my-domain.com", true, "Hyphen is allowed mid-label"},
		{"_domainkey.foo.com", true, "Underscores are allowed"},
		{"bar.com.", true, "Must be able to end with a dot"},
		{"ABC.6DBS.com", true, "Mixed case is OK"},
		{"google..com", false, "Double dot not valid"},
		{".foo.com", false, "Cannot start with a dot"},
		{"mail.123.com", false, "Number only label not valid"},
		{"google\r.com", false, "Special chars not allowed"},
		{"foo.-bar.com", false, "Label cannot start with hyphen"},
		{"foo-.bar.com", false, "Label cannot end with hyphen"},
	}

	for _, tt := range testTable {
		if ValidateDomainPart(tt.input) != tt.expect {
			t.Errorf("Expected %v for %q: %s", tt.expect, tt.input, tt.msg)
		}
	}
}
