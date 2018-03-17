package stringutil_test

import (
	"net/mail"
	"testing"

	"github.com/jhillyerd/inbucket/pkg/stringutil"
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
		{Name: "Fred B. Fish", Address: "fred@fish.org"},
		{Name: "User", Address: "user@domain.org"},
	}
	want := []string{`"Fred B. Fish" <fred@fish.org>`, `"User" <user@domain.org>`}
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
