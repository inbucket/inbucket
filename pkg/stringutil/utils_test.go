package stringutil_test

import (
	"fmt"
	"net/mail"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/stringutil"
	"github.com/stretchr/testify/assert"
)

func TestHashMailboxName(t *testing.T) {
	want := "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	got := stringutil.HashMailboxName("")
	assert.Equal(t, want, got, "for empty string")

	want = "1d6e1cf70ec6f9ab28d3ea4b27a49a77654d370e"
	got = stringutil.HashMailboxName("mail")
	assert.Equal(t, want, got, "for 'mail'")
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

func TestMakePathPrefixer(t *testing.T) {
	testCases := []struct {
		prefix, path, want string
	}{
		{prefix: "", path: "", want: ""},
		{prefix: "", path: "relative", want: "relative"},
		{prefix: "", path: "/qualified", want: "/qualified"},
		{prefix: "", path: "/many/path/segments", want: "/many/path/segments"},
		{prefix: "pfx", path: "", want: "/pfx"},
		{prefix: "pfx", path: "/", want: "/pfx/"},
		{prefix: "pfx", path: "relative", want: "/pfxrelative"},
		{prefix: "pfx", path: "/qualified", want: "/pfx/qualified"},
		{prefix: "pfx", path: "/many/path/segments", want: "/pfx/many/path/segments"},
		{prefix: "/pfx/", path: "", want: "/pfx"},
		{prefix: "/pfx/", path: "/", want: "/pfx/"},
		{prefix: "/pfx/", path: "relative", want: "/pfxrelative"},
		{prefix: "/pfx/", path: "/qualified", want: "/pfx/qualified"},
		{prefix: "/pfx/", path: "/many/path/segments", want: "/pfx/many/path/segments"},
		{prefix: "a/b/c", path: "", want: "/a/b/c"},
		{prefix: "a/b/c", path: "/", want: "/a/b/c/"},
		{prefix: "a/b/c", path: "relative", want: "/a/b/crelative"},
		{prefix: "a/b/c", path: "/qualified", want: "/a/b/c/qualified"},
		{prefix: "a/b/c", path: "/many/path/segments", want: "/a/b/c/many/path/segments"},
		{prefix: "/a/b/c/", path: "", want: "/a/b/c"},
		{prefix: "/a/b/c/", path: "/", want: "/a/b/c/"},
		{prefix: "/a/b/c/", path: "relative", want: "/a/b/crelative"},
		{prefix: "/a/b/c/", path: "/qualified", want: "/a/b/c/qualified"},
		{prefix: "/a/b/c/", path: "/many/path/segments", want: "/a/b/c/many/path/segments"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("prefix %s for path %s", tc.prefix, tc.path), func(t *testing.T) {
			prefixer := stringutil.MakePathPrefixer(tc.prefix)
			got := prefixer(tc.path)
			if got != tc.want {
				t.Errorf("Got: %q, want: %q", got, tc.want)
			}
		})
	}
}

func TestMatchWithWildcards(t *testing.T) {
	testCases := []struct {
		pattern, input string
		want           bool
	}{
		{pattern: "", input: "", want: true},
		{pattern: "", input: "qwerty", want: false},
		{pattern: "qw*ty", input: "qwerty", want: true},
		{pattern: "qw?ty", input: "qwerty", want: false},
		{pattern: "qwe*ty", input: "qwerty", want: true},
		{pattern: "*erty", input: "qwerty", want: true},
		{pattern: "?erty", input: "qwerty", want: false},
		{pattern: "?werty", input: "qwerty", want: true},
		{pattern: "qwer*", input: "qwerty", want: true},
		{pattern: "qwer?", input: "qwerty", want: false},
		{pattern: "qwert?", input: "qwerty", want: true},
		{pattern: "qw**ty", input: "qwerty", want: true},
		{pattern: "qw??ty", input: "qwerty", want: true},
		{pattern: "qwe??ty", input: "qwerty", want: false},
		{pattern: "**erty", input: "qwerty", want: true},
		{pattern: "??erty", input: "qwerty", want: true},
		{pattern: "??werty", input: "qwerty", want: false},
		{pattern: "qwer**", input: "qwerty", want: true},
		{pattern: "qwer??", input: "qwerty", want: true},
		{pattern: "qwert??", input: "qwerty", want: false},
		{pattern: "q?er?y", input: "qwerty", want: true},
		{pattern: "q?r?y", input: "qwerty", want: false},
		{pattern: "q*er*y", input: "qwerty", want: true},
		{pattern: "q*r*y", input: "qwerty", want: true},
		{pattern: "q*?werty", input: "qwerty", want: false},
		{pattern: "q*?erty", input: "qwerty", want: true},
		{pattern: "q?*werty", input: "qwerty", want: false},
		{pattern: "q?*erty", input: "qwerty", want: true},
		{pattern: "?*rty", input: "qwerty", want: true},
		{pattern: "*?rty", input: "qwerty", want: true},
		{pattern: "qwe?*", input: "qwerty", want: true},
		{pattern: "qwe*?", input: "qwerty", want: true},
	}
	for _, tc := range testCases {
		got := stringutil.MatchWithWildcards(tc.pattern, tc.input)
		if got != tc.want {
			t.Errorf("Test %s with pattern %s, Got: %v, want: %v", tc.input, tc.pattern, got, tc.want)
		}
	}
}
