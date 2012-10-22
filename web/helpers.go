package web

import (
	"fmt"
	"github.com/jhillyerd/inbucket/log"
	"html"
	"html/template"
	"strings"
	"time"
)

var TemplateFuncs = template.FuncMap{
	"friendlyTime": friendlyTime,
	"reverse": reverse,
	"textToHtml": textToHtml,
}

// Friendly date & time rendering
func friendlyTime(t time.Time) template.HTML {
	ty, tm, td := t.Date()
	ny, nm, nd := time.Now().Date()
	if (ty == ny) && (tm == nm) && (td == nd) {
		return template.HTML(t.Format("03:04:05 PM"))
	}
	return template.HTML(t.Format("Mon Jan 2, 2006"))
}

// Reversable routing function (shared with templates)
func reverse(name string, things ...interface{}) string {
	// Convert the things to strings
	strs := make([]string, len(things))
	for i, th := range things {
		strs[i] = fmt.Sprint(th)
	}
	// Grab the route
	u, err := Router.Get(name).URL(strs...)
	if err != nil {
		log.Error("Failed to reverse route: %v", err)
		return "/ROUTE-ERROR"
	}
	return u.Path
}

// textToHtml takes plain text, escapes it and tries to pretty it up for
// HTML display
func textToHtml(text string) template.HTML {
	text = html.EscapeString(text)
	replacer := strings.NewReplacer("\r\n", "<br/>\n", "\r", "<br/>\n", "\n", "<br/>\n")
	return template.HTML(replacer.Replace(text))
}
