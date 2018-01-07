package sanitize_test

import (
	"testing"

	"github.com/jhillyerd/inbucket/sanitize"
)

// TestHTMLPlainStrings test plain text passthrough
func TestHTMLPlainStrings(t *testing.T) {
	testStrings := []string{
		"",
		"plain string",
		"one &lt; two",
	}
	for _, ts := range testStrings {
		t.Run(ts, func(t *testing.T) {
			got, err := sanitize.HTML(ts)
			if err != nil {
				t.Fatal(err)
			}
			if got != ts {
				t.Errorf("Got: %q, want: %q", got, ts)
			}
		})
	}
}

// TestHTMLSimpleFormatting tests basic tags we should allow
func TestHTMLSimpleFormatting(t *testing.T) {
	testStrings := []string{
		"<p>paragraph</p>",
		"<b>bold</b>",
		"<i>italic</b>",
		"<em>emphasis</em>",
		"<strong>strong</strong>",
		"<div><span>text</span></div>",
	}
	for _, ts := range testStrings {
		t.Run(ts, func(t *testing.T) {
			got, err := sanitize.HTML(ts)
			if err != nil {
				t.Fatal(err)
			}
			if got != ts {
				t.Errorf("Got: %q, want: %q", got, ts)
			}
		})
	}
}

// TestHTMLScriptTags tests some strings with JavaScript
func TestHTMLScriptTags(t *testing.T) {
	testCases := []struct {
		input, want string
	}{
		{
			`safe<script>nope</script>`,
			`safe`,
		},
		{
			`<a onblur="alert(something)" href="http://mysite.com">mysite</a>`,
			`<a href="http://mysite.com" rel="nofollow">mysite</a>`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := sanitize.HTML(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("Got: %q, want: %q", got, tc.want)
			}
		})
	}
}
