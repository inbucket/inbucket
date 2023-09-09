package sanitize_test

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/webui/sanitize"
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
		"<center>text</center>",
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

func TestSanitizeStyleTags(t *testing.T) {
	testCases := []struct {
		name, input, want string
	}{
		{
			"empty",
			``,
			``,
		},
		{
			"open",
			`<div>`,
			`<div>`,
		},
		{
			"open close",
			`<div></div>`,
			`<div></div>`,
		},
		{
			"inner text",
			`<div>foo bar</div>`,
			`<div>foo bar</div>`,
		},
		{
			"self close",
			`<br/>`,
			`<br/>`,
		},
		{
			"open params",
			`<div id="me">`,
			`<div id="me">`,
		},
		{
			"open params squote",
			`<div id="me" title='best'>`,
			`<div id="me" title="best">`,
		},
		{
			"open style",
			`<div id="me" style="color: red;">`,
			`<div id="me" style="color: red;">`,
		},
		{
			"open style squote",
			`<div id="me" style='color: red;'>`,
			`<div id="me" style="color: red;">`,
		},
		{
			"open style mixed case",
			`<div id="me" StYlE="color: red;">`,
			`<div id="me" style="color: red;">`,
		},
		{
			"closed style",
			`<br style="border: 1px solid red;"/>`,
			`<br style="border: 1px solid red;"/>`,
		},
		{
			"mixed case style",
			`<br StYlE="border: 1px solid red;"/>`,
			`<br style="border: 1px solid red;"/>`,
		},
		{
			"mixed case invalid style",
			`<br StYlE="position: fixed;"/>`,
			`<br/>`,
		},
		{
			"mixed",
			`<p id='i' title="cla'zz" style="font-size: 25px;"><b>some text</b></p>`,
			`<p id="i" title="cla&#39;zz" style="font-size: 25px;"><b>some text</b></p>`,
		},
		{
			"invalid styles",
			`<div id="me" style='position: absolute;'>`,
			`<div id="me">`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitize.HTML(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("input: %s\ngot : %s\nwant: %s", tc.input, got, tc.want)
			}
		})
	}
}
