package web

import (
	"github.com/stretchrcom/testify/assert"
	"testing"
)

func TestTextToHtml(t *testing.T) {
	// Identity
	assert.Equal(t, textToHtml("html"), "html")

	// Check it escapes
	assert.Equal(t, textToHtml("<html>"), "&lt;html&gt;")

	// Check for linebreaks
	assert.Equal(t, textToHtml("line\nbreak"), "line<br/>\nbreak")
	assert.Equal(t, textToHtml("line\r\nbreak"), "line<br/>\nbreak")
	assert.Equal(t, textToHtml("line\rbreak"), "line<br/>\nbreak")
}
