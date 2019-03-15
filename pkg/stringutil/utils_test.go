package stringutil_test

import (
	"net/mail"
	"testing"

	"github.com/inbucket/inbucket/pkg/stringutil"
)

func TestHashMailboxName(t *testing.T) {
	want := "1d6e1cf70ec6f9ab28d3ea4b27a49a77654d370e"
	got := stringutil.HashMailboxName("mail")
	if got != want {
		t.Errorf("Got %q, want %q", got, want)
	}
}

func TestStringAddressList(t *testing.T) {
	input := []*mail.Address{
		{Name: "Fred ß. Fish", Address: "fred@fish.org"},
		{Name: "User", Address: "user@domain.org"},
		{Address: "a@b.com"},
	}
	want := []string{
		`Fred ß. Fish <fred@fish.org>`,
		`User <user@domain.org>`,
		`<a@b.com>`}
	output := stringutil.StringAddressList(input)
	if len(output) != len(want) {
		t.Fatalf("Got %v strings, want: %v", len(output), len(want))
	}
	for i, got := range output {
		if got != want[i] {
			t.Errorf("Got %q, want: %q", got, want[i])
		}
	}
}
