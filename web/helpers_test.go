package web

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextToHtml(t *testing.T) {
	// Identity
	assert.Equal(t, textToHTML("html"), template.HTML("html"))

	// Check it escapes
	assert.Equal(t, textToHTML("<html>"), template.HTML("&lt;html&gt;"))

	// Check for linebreaks
	assert.Equal(t, textToHTML("line\nbreak"), template.HTML("line<br/>\nbreak"))
	assert.Equal(t, textToHTML("line\r\nbreak"), template.HTML("line<br/>\nbreak"))
	assert.Equal(t, textToHTML("line\rbreak"), template.HTML("line<br/>\nbreak"))
}

func TestURLDetection(t *testing.T) {
	assert.Equal(t,
		textToHTML("http://google.com/"),
		template.HTML("<a href=\"http://google.com/\" target=\"_blank\">http://google.com/</a>"))
	assert.Equal(t,
		textToHTML("http://a.com/?q=a&n=v"),
		template.HTML("<a href=\"http://a.com/?q=a&n=v\" target=\"_blank\">http://a.com/?q=a&amp;n=v</a>"))
}
