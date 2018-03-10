package web

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextToHtml(t *testing.T) {
	// Identity
	assert.Equal(t, TextToHTML("html"), template.HTML("html"))

	// Check it escapes
	assert.Equal(t, TextToHTML("<html>"), template.HTML("&lt;html&gt;"))

	// Check for linebreaks
	assert.Equal(t, TextToHTML("line\nbreak"), template.HTML("line<br/>\nbreak"))
	assert.Equal(t, TextToHTML("line\r\nbreak"), template.HTML("line<br/>\nbreak"))
	assert.Equal(t, TextToHTML("line\rbreak"), template.HTML("line<br/>\nbreak"))
}

func TestURLDetection(t *testing.T) {
	assert.Equal(t,
		TextToHTML("http://google.com/"),
		template.HTML("<a href=\"http://google.com/\" target=\"_blank\">http://google.com/</a>"))
	assert.Equal(t,
		TextToHTML("http://a.com/?q=a&n=v"),
		template.HTML("<a href=\"http://a.com/?q=a&n=v\" target=\"_blank\">http://a.com/?q=a&amp;n=v</a>"))
}
