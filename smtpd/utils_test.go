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
		{"", false, "Empty domain is not valid"},
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

func TestValidateLocal(t *testing.T) {
	var testTable = []struct {
		input  string
		expect bool
		msg    string
	}{
		{"", false, "Empty local is not valid"},
		{"a", true, "Single letter should be fine"},
		{strings.Repeat("a", 65), false, "Only valid up to 64 characters"},
		{"FirstLast", true, "Mixed case permitted"},
		{"user123", true, "Numbers permitted"},
		{"a!#$%&'*+-/=?^_`{|}~", true, "Any of !#$%&'*+-/=?^_`{|}~ are permitted"},
		{"first.last", true, "Embedded period is permitted"},
		{"first..last", false, "Sequence of periods is not allowed"},
		{".user", false, "Cannot lead with a period"},
		{"user.", false, "Cannot end with a period"},
		{"james@mail", false, "Unquoted @ not permitted"},
		{"first last", false, "Unquoted space not permitted"},
		{"no,commas", false, "Unquoted comma not allowed"},
		{"t[es]t", false, "Unquoted square brackets not allowed"},
		{"james\\", false, "Cannot end with backslash quote"},
		{"james\\@mail", true, "Quoted @ permitted"},
		{"quoted\\ space", true, "Quoted space permitted"},
		{"no\\,commas", true, "Quoted comma is OK"},
		{"t\\[es\\]t", true, "Quoted brackets are OK"},
		{"user\\name", true, "Should be able to quote a-z"},
		{"USER\\NAME", true, "Should be able to quote A-Z"},
		{"user\\1", true, "Should be able to quote a digit"},
		{"one\\$\\|", true, "Should be able to quote plain specials"},
		{"return\\\r", true, "Should be able to quote ASCII control chars"},
		{"high\\\x80", false, "Should not accept > 7-bit quoted chars"},
		{"quote\\\"", true, "Quoted double quote is permitted"},
		{"\"james\"", true, "Quoted a-z is permitted"},
		{"\"first last\"", true, "Quoted space is permitted"},
		{"\"quoted@sign\"", true, "Quoted @ is allowed"},
		{"\"qp\\\"quote\"", true, "Quoted quote within quoted string is OK"},
		{"\"unterminated", false, "Quoted string must be terminated"},
		{"\"unterminated\\\"", false, "Quoted string must be terminated"},
		{"embed\"quote\"string", false, "Embedded quoted string is illegal"},

		{"user+mailbox", true, "RFC3696 test case should be valid"},
		{"customer/department=shipping", true, "RFC3696 test case should be valid"},
		{"$A12345", true, "RFC3696 test case should be valid"},
		{"!def!xyz%abc", true, "RFC3696 test case should be valid"},
		{"_somename", true, "RFC3696 test case should be valid"},
	}

	for _, tt := range testTable {
		_, _, err := ParseEmailAddress(tt.input + "@domain.com")
		if (err != nil) == tt.expect {
			if err != nil {
				t.Logf("Got error: %s", err)
			}
			t.Errorf("Expected %v for %q: %s", tt.expect, tt.input, tt.msg)
		}
	}
}

func TestParseEmailAddress(t *testing.T) {
	var testTable = []struct {
		input, local, domain string
	}{
		{"root@localhost", "root", "localhost"},
	}

	for _, tt := range testTable {
		local, domain, err := ParseEmailAddress(tt.input)
		if err != nil {
			t.Errorf("Error when parsing %q: %s", tt.input, err)
		} else {
			if tt.local != local {
				t.Errorf("When parsing %q, expected local %q, got %q instead",
					tt.input, tt.local, local)
			}
			if tt.domain != domain {
				t.Errorf("When parsing %q, expected domain %q, got %q instead",
					tt.input, tt.domain, domain)
			}
		}
	}
}
