package web

import (
	"testing"
)

func TestTextToHtml(t *testing.T) {
	testCases := []struct {
		input, want string
	}{
		{
			input: "html",
			want:  "html",
		},
		// Check it escapes.
		{
			input: "<html>",
			want:  "&lt;html&gt;",
		},
		// Check for linebreaks.
		{
			input: "line\nbreak",
			want:  "line<br/>\nbreak",
		},
		{
			input: "line\r\nbreak",
			want:  "line<br/>\nbreak",
		},
		{
			input: "line\rbreak",
			want:  "line<br/>\nbreak",
		},
		// Check URL detection.
		{
			input: "http://google.com/",
			want:  "<a href=\"http://google.com/\" target=\"_blank\">http://google.com/</a>",
		},
		{
			input: "http://a.com/?q=a&n=v",
			want:  "<a href=\"http://a.com/?q=a&n=v\" target=\"_blank\">http://a.com/?q=a&amp;n=v</a>",
		},
		{
			input: "(http://a.com/?q=a&n=v)",
			want:  "(<a href=\"http://a.com/?q=a&n=v\" target=\"_blank\">http://a.com/?q=a&amp;n=v</a>)",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got := TextToHTML(tc.input)
			if got != tc.want {
				t.Errorf("TextToHTML(%q)\ngot : %q\nwant: %q", tc.input, got, tc.want)
			}
		})
	}
}
