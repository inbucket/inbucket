package web

import (
	"github.com/stretchrcom/testify/assert"
	"html/template"
	"testing"
)

func TestTextToHtml(t *testing.T) {
	// Identity
	assert.Equal(t, textToHtml("html"), template.HTML("html"))

	// Check it escapes
	assert.Equal(t, textToHtml("<html>"), template.HTML("&lt;html&gt;"))

	// Check for linebreaks
	assert.Equal(t, textToHtml("line\nbreak"), template.HTML("line<br/>\nbreak"))
	assert.Equal(t, textToHtml("line\r\nbreak"), template.HTML("line<br/>\nbreak"))
	assert.Equal(t, textToHtml("line\rbreak"), template.HTML("line<br/>\nbreak"))
}

func TestURLDetection(t *testing.T) {
	assert.Equal(t,
		textToHtml("http://google.com/"),
		template.HTML("<a href=\"http://google.com/\" target=\"_blank\">http://google.com/</a>"))
	assert.Equal(t,
		textToHtml("http://a.com/?q=a&n=v"),
		template.HTML("<a href=\"http://a.com/?q=a&n=v\" target=\"_blank\">http://a.com/?q=a&amp;n=v</a>"))
}
