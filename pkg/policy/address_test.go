package policy_test

import (
	"strings"
	"testing"

	"github.com/jhillyerd/inbucket/pkg/policy"
)

func TestParseMailboxName(t *testing.T) {
	var validTable = []struct {
		input  string
		expect string
	}{
		{"mailbox", "mailbox"},
		{"user123", "user123"},
		{"MailBOX", "mailbox"},
		{"First.Last", "first.last"},
		{"user+label", "user"},
		{"chars!#$%", "chars!#$%"},
		{"chars&'*-", "chars&'*-"},
		{"chars=/?^", "chars=/?^"},
		{"chars_`.{", "chars_`.{"},
		{"chars|}~", "chars|}~"},
	}
	for _, tt := range validTable {
		if result, err := policy.ParseMailboxName(tt.input); err != nil {
			t.Errorf("Error while parsing %q: %v", tt.input, err)
		} else {
			if result != tt.expect {
				t.Errorf("Parsing %q, expected %q, got %q", tt.input, tt.expect, result)
			}
		}
	}
	var invalidTable = []struct {
		input, msg string
	}{
		{"", "Empty mailbox name is not permitted"},
		{"user@host", "@ symbol not permitted"},
		{"first last", "Space not permitted"},
		{"first\"last", "Double quote not permitted"},
		{"first\nlast", "Control chars not permitted"},
	}
	for _, tt := range invalidTable {
		if _, err := policy.ParseMailboxName(tt.input); err == nil {
			t.Errorf("Didn't get an error while parsing %q: %v", tt.input, tt.msg)
		}
	}
}

func TestValidateDomain(t *testing.T) {
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
		{"mail.123.com", true, "Number only label valid"},
		{"123.com", true, "Number only label valid"},
		{"google..com", false, "Double dot not valid"},
		{".foo.com", false, "Cannot start with a dot"},
		{"google\r.com", false, "Special chars not allowed"},
		{"foo.-bar.com", false, "Label cannot start with hyphen"},
		{"foo-.bar.com", false, "Label cannot end with hyphen"},
		{strings.Repeat("a", 256), false, "Max domain length is 255"},
		{strings.Repeat("a", 63) + ".com", true, "Should allow 63 char domain label"},
		{strings.Repeat("a", 64) + ".com", false, "Max domain label length is 63"},
	}
	for _, tt := range testTable {
		if policy.ValidateDomainPart(tt.input) != tt.expect {
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
		{strings.Repeat("a", 128), true, "Valid up to 128 characters"},
		{strings.Repeat("a", 129), false, "Only valid up to 128 characters"},
		{"FirstLast", true, "Mixed case permitted"},
		{"user123", true, "Numbers permitted"},
		{"a!#$%&'*+-/=?^_`{|}~", true, "Any of !#$%&'*+-/=?^_`{|}~ are permitted"},
		{"first.last", true, "Embedded period is permitted"},
		{"first..last", false, "Sequence of periods is not allowed"},
		{".user", false, "Cannot lead with a period"},
		{"user.", false, "Cannot end with a period"},
		{"james@mail", false, "Unquoted @ not permitted"},
		{"first last", false, "Unquoted space not permitted"},
		{"tricky\\. ", false, "Unquoted space not permitted"},
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
		_, _, err := policy.ParseEmailAddress(tt.input + "@domain.com")
		if (err != nil) == tt.expect {
			if err != nil {
				t.Logf("Got error: %s", err)
			}
			t.Errorf("Expected %v for %q: %s", tt.expect, tt.input, tt.msg)
		}
	}
}
