package web

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// From http://daringfireball.net/2010/07/improved_regex_for_matching_urls
var urlRE = regexp.MustCompile("(?i)\\b((?:[a-z][\\w-]+:(?:/{1,3}|[a-z0-9%])|www\\d{0,3}[.]|[a-z0-9.\\-]+[.][a-z]{2,4}/)(?:[^\\s()<>]+|\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\))+(?:\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\)|[^\\s`!()\\[\\]{};:'\".,<>?«»“”‘’]))")

// TextToHTML takes plain text, escapes it and tries to pretty it up for
// HTML display
func TextToHTML(text string) string {
	text = html.EscapeString(text)
	text = urlRE.ReplaceAllStringFunc(text, WrapURL)
	replacer := strings.NewReplacer("\r\n", "<br/>\n", "\r", "<br/>\n", "\n", "<br/>\n")
	return replacer.Replace(text)
}

// WrapURL wraps a <a href> tag around the provided URL
func WrapURL(url string) string {
	unescaped := strings.ReplaceAll(url, "&amp;", "&")
	return fmt.Sprintf("<a href=\"%s\" target=\"_blank\">%s</a>", unescaped, url)
}
