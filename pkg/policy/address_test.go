package policy_test

import (
	"strings"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/policy"
)

func TestShouldAcceptDomain(t *testing.T) {
	// Test with default accept.
	ap := &policy.Addressing{
		Config: &config.Root{
			SMTP: config.SMTP{
				DefaultAccept: true,
				RejectDomains: []string{"a.deny.com", "deny.com"},
			},
		},
	}
	testCases := []struct {
		domain string
		want   bool
	}{
		{domain: "bar.com", want: true},
		{domain: "DENY.com", want: false},
		{domain: "a.deny.com", want: false},
		{domain: "b.deny.com", want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			got := ap.ShouldAcceptDomain(tc.domain)
			if got != tc.want {
				t.Errorf("Got %v for %q, want: %v", got, tc.domain, tc.want)
			}
		})
	}
	// Test with default reject.
	ap = &policy.Addressing{
		Config: &config.Root{
			SMTP: config.SMTP{
				DefaultAccept: false,
				AcceptDomains: []string{"a.allow.com", "allow.com"},
			},
		},
	}
	testCases = []struct {
		domain string
		want   bool
	}{
		{domain: "bar.com", want: false},
		{domain: "ALLOW.com", want: true},
		{domain: "a.allow.com", want: true},
		{domain: "b.allow.com", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			got := ap.ShouldAcceptDomain(tc.domain)
			if got != tc.want {
				t.Errorf("Got %v for %q, want: %v", got, tc.domain, tc.want)
			}
		})
	}
}

func TestShouldStoreDomain(t *testing.T) {
	// Test with storage enabled.
	ap := &policy.Addressing{
		Config: &config.Root{
			SMTP: config.SMTP{
				DefaultStore: false,
				StoreDomains: []string{"store.com", "a.store.com"},
			},
		},
	}
	testCases := []struct {
		domain string
		want   bool
	}{
		{domain: "foo.com", want: false},
		{domain: "STORE.com", want: true},
		{domain: "a.store.com", want: true},
		{domain: "b.store.com", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			got := ap.ShouldStoreDomain(tc.domain)
			if got != tc.want {
				t.Errorf("Got store %v for %q, want: %v", got, tc.domain, tc.want)
			}
		})
	}
	// Test with storage disabled.
	ap = &policy.Addressing{
		Config: &config.Root{
			SMTP: config.SMTP{
				DefaultStore:   true,
				DiscardDomains: []string{"discard.com", "a.discard.com"},
			},
		},
	}
	testCases = []struct {
		domain string
		want   bool
	}{
		{domain: "foo.com", want: true},
		{domain: "DISCARD.com", want: false},
		{domain: "a.discard.com", want: false},
		{domain: "b.discard.com", want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			got := ap.ShouldStoreDomain(tc.domain)
			if got != tc.want {
				t.Errorf("Got store %v for %q, want: %v", got, tc.domain, tc.want)
			}
		})
	}
}

func TestExtractMailboxValid(t *testing.T) {
	localPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.LocalNaming}}
	fullPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.FullNaming}}
	domainPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.DomainNaming}}

	testTable := []struct {
		input  string // Input to test
		local  string // Expected output when mailbox naming = local
		full   string // Expected output when mailbox naming = full
		domain string // Expected output when mailbox naming = domain
	}{
		{
			input:  "mailbox",
			local:  "mailbox",
			full:   "mailbox",
			domain: "mailbox",
		},
		{
			input:  "user123",
			local:  "user123",
			full:   "user123",
			domain: "user123",
		},
		{
			input:  "MailBOX",
			local:  "mailbox",
			full:   "mailbox",
			domain: "mailbox",
		},
		{
			input:  "First.Last",
			local:  "first.last",
			full:   "first.last",
			domain: "first.last",
		},
		{
			input:  "user+label",
			local:  "user",
			full:   "user",
			domain: "user",
		},
		{
			input:  "chars!#$%",
			local:  "chars!#$%",
			full:   "chars!#$%",
			domain: "",
		},
		{
			input:  "chars&'*-",
			local:  "chars&'*-",
			full:   "chars&'*-",
			domain: "",
		},
		{
			input:  "chars=/?^",
			local:  "chars=/?^",
			full:   "chars=/?^",
			domain: "",
		},
		{
			input:  "chars_`.{",
			local:  "chars_`.{",
			full:   "chars_`.{",
			domain: "",
		},
		{
			input:  "chars|}~",
			local:  "chars|}~",
			full:   "chars|}~",
			domain: "",
		},
		{
			input:  "mailbox@domain.com",
			local:  "mailbox",
			full:   "mailbox@domain.com",
			domain: "domain.com",
		},
		{
			input:  "user123@domain.com",
			local:  "user123",
			full:   "user123@domain.com",
			domain: "domain.com",
		},
		{
			input:  "MailBOX@domain.com",
			local:  "mailbox",
			full:   "mailbox@domain.com",
			domain: "domain.com",
		},
		{
			input:  "First.Last@domain.com",
			local:  "first.last",
			full:   "first.last@domain.com",
			domain: "domain.com",
		},
		{
			input:  "user+label@domain.com",
			local:  "user",
			full:   "user@domain.com",
			domain: "domain.com",
		},
		{
			input:  "chars!#$%@domain.com",
			local:  "chars!#$%",
			full:   "chars!#$%@domain.com",
			domain: "domain.com",
		},
		{
			input:  "chars&'*-@domain.com",
			local:  "chars&'*-",
			full:   "chars&'*-@domain.com",
			domain: "domain.com",
		},
		{
			input:  "chars=/?^@domain.com",
			local:  "chars=/?^",
			full:   "chars=/?^@domain.com",
			domain: "domain.com",
		},
		{
			input:  "chars_`.{@domain.com",
			local:  "chars_`.{",
			full:   "chars_`.{@domain.com",
			domain: "domain.com",
		},
		{
			input:  "chars|}~@domain.com",
			local:  "chars|}~",
			full:   "chars|}~@domain.com",
			domain: "domain.com",
		},
		{
			input:  "chars|}~@example.co.uk",
			local:  "chars|}~",
			full:   "chars|}~@example.co.uk",
			domain: "example.co.uk",
		},
		{
			input:  "@host:user+label@domain.com",
			local:  "user",
			full:   "user@domain.com",
			domain: "domain.com",
		},
		{
			input:  "@a.com,@b.com:user+label@domain.com",
			local:  "user",
			full:   "user@domain.com",
			domain: "domain.com",
		},
		{
			input:  "u@[127.0.0.1]",
			local:  "u",
			full:   "u@[127.0.0.1]",
			domain: "[127.0.0.1]",
		},
		{
			input:  "u@[IPv6:2001:db8:aaaa:1::100]",
			local:  "u",
			full:   "u@[IPv6:2001:db8:aaaa:1::100]",
			domain: "[IPv6:2001:db8:aaaa:1::100]",
		},
	}
	for _, tc := range testTable {
		if result, err := localPolicy.ExtractMailbox(tc.input); err != nil {
			t.Errorf("Error while parsing with local naming %q: %v", tc.input, err)
		} else if result != tc.local {
			t.Errorf("Parsing %q, expected %q, got %q", tc.input, tc.local, result)
		}
		if result, err := fullPolicy.ExtractMailbox(tc.input); err != nil {
			t.Errorf("Error while parsing with full naming %q: %v", tc.input, err)
		} else if result != tc.full {
			t.Errorf("Parsing %q, expected %q, got %q", tc.input, tc.full, result)
		}
		if result, err := domainPolicy.ExtractMailbox(tc.input); tc.domain != "" && err != nil {
			t.Errorf("Error while parsing with domain naming %q: %v", tc.input, err)
		} else if result != tc.domain {
			t.Errorf("Parsing %q, expected %q, got %q", tc.input, tc.domain, result)
		}
	}
}

// Test special cases with domain addressing mode.
func TestExtractDomainMailboxValid(t *testing.T) {
	domainPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.DomainNaming}}

	tests := map[string]struct {
		input  string // Input to test
		domain string // Expected output when mailbox naming = domain
	}{
		"ipv4": {
			input:  "[127.0.0.1]",
			domain: "[127.0.0.1]",
		},
		"medium ipv6": {
			input:  "[IPv6:2001:db8:aaaa:1::100]",
			domain: "[IPv6:2001:db8:aaaa:1::100]",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if result, err := domainPolicy.ExtractMailbox(tc.input); tc.domain != "" && err != nil {
				t.Errorf("Error while parsing with domain naming %q: %v", tc.input, err)
			} else if result != tc.domain {
				t.Errorf("Parsing %q, expected %q, got %q", tc.input, tc.domain, result)
			}
		})
	}
}

func TestExtractMailboxInvalid(t *testing.T) {
	localPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.LocalNaming}}
	fullPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.FullNaming}}
	domainPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.DomainNaming}}

	// Test local mailbox naming policy.
	localInvalidTable := []struct {
		input, msg string
	}{
		{"", "Empty mailbox name is not permitted"},
		{"first last", "Space not permitted"},
		{"first\"last", "Double quote not permitted"},
		{"first\nlast", "Control chars not permitted"},
	}
	for _, tt := range localInvalidTable {
		if _, err := localPolicy.ExtractMailbox(tt.input); err == nil {
			t.Errorf("Didn't get an error while parsing in local mode %q: %v", tt.input, tt.msg)
		}
	}

	// Test full mailbox naming policy.
	fullInvalidTable := []struct {
		input, msg string
	}{
		{"", "Empty mailbox name is not permitted"},
		{"user@host@domain.com", "@ symbol not permitted"},
		{"first last@domain.com", "Space not permitted"},
		{"first\"last@domain.com", "Double quote not permitted"},
		{"first\nlast@domain.com", "Control chars not permitted"},
	}
	for _, tt := range fullInvalidTable {
		if _, err := fullPolicy.ExtractMailbox(tt.input); err == nil {
			t.Errorf("Didn't get an error while parsing in full mode %q: %v", tt.input, tt.msg)
		}
	}

	// Test domain mailbox naming policy.
	domainInvalidTable := []struct {
		input, msg string
	}{
		{"", "Empty mailbox name is not permitted"},
		{"user@host@domain.com", "@ symbol not permitted"},
		{"first.last@dom ain.com", "Space not permitted"},
		{"first\"last@domain.com", "Double quote not permitted"},
		{"first\nlast@domain.com", "Control chars not permitted"},
		{"first.last@chars!#$%.com", "Invalid domain name"},
		{"first.last@.example.com", "Domain cannot start with dot"},
		{"first.last@-example.com", "Domain canont start with dash"},
		{"first.last@example.com-", "Domain cannot end with dash"},
		{"first.last@example..com", "Domain cannot contain double dots"},
		{"first.last@example--com", "Domain cannot contain double dashes"},
		{"first.last@example.-com", "Domain cannot contain concecutive symbols"},
	}
	for _, tt := range domainInvalidTable {
		if _, err := domainPolicy.ExtractMailbox(tt.input); err == nil {
			t.Errorf("Didn't get an error while parsing in domain mode %q: %v", tt.input, tt.msg)
		}
	}
}

func TestValidateDomain(t *testing.T) {
	testTable := []struct {
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
		{"[0.0.0.0]", true, "Single digit octet IP addr is valid"},
		{"[123.123.123.123]", true, "Multiple digit octet IP addr is valid"},
		{"[IPv6:2001:0db8:aaaa:0001:0000:0000:0000:0200]", true, "Full IPv6 addr is valid"},
		{"[IPv6:::1]", true, "Abbr IPv6 addr is valid"},
	}
	for _, tt := range testTable {
		if policy.ValidateDomainPart(tt.input) != tt.expect {
			t.Errorf("Expected %v for %q: %s", tt.expect, tt.input, tt.msg)
		}
	}
}

func TestValidateLocal(t *testing.T) {
	testTable := []struct {
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
		// {"james@mail", false, "Unquoted @ not permitted"},
		{"first last", false, "Unquoted space not permitted"},
		{"tricky\\. ", false, "Unquoted space not permitted"},
		{"no,commas", false, "Unquoted comma not allowed"},
		{"t[es]t", false, "Unquoted square brackets not allowed"},
		// {"james\\", false, "Cannot end with backslash quote"},
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
		{"@host:mailbox", true, "Forward-path routes are valid"},
		{"@a.com,@b.com:mailbox", true, "Multi-hop forward-path routes are valid"},
		{"@a.com,mailbox", false, "Unterminated forward-path routes are invalid"},
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

// TestRecipientAddress verifies the Recipient.Address values returned by Addressing.NewRecipient.
// This function parses a RCPT TO path, not a To header. See rfc5321#section-4.1.2
func TestRecipientAddress(t *testing.T) {
	localPolicy := policy.Addressing{Config: &config.Root{MailboxNaming: config.LocalNaming}}

	tests := map[string]string{
		"common":          "user@example.com",
		"with label":      "user+mailbox@example.com",
		"special chars":   "a!#$%&'*+-/=?^_`{|}~@example.com",
		"ipv4":            "user@[127.0.0.1]",
		"ipv6":            "user@[IPv6:::1]",
		"route host":      "@host:user@example.com",
		"route domain":    "@route.com:user@example.com",
		"multi-hop route": "@first.com,@second.com:user@example.com",
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := localPolicy.NewRecipient(tc)
			if err != nil {
				t.Fatalf("Parse of %q failed: %v", tc, err)
			}

			if got, want := r.Address.Address, tc; got != want {
				t.Errorf("Got Address: %q, want: %q", got, want)
			}
		})
	}
}
