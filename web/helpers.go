package web

import (
	"fmt"
	"github.com/jhillyerd/inbucket"
	"html/template"
	"time"
)

var TemplateFuncs = template.FuncMap{
	// Reversable routing function (shared with templates)
	"reverse": reverse,
	// Friendly date & time rendering
	"friendlyTime": func(t time.Time) template.HTML {
		ty, tm, td := t.Date()
		ny, nm, nd := time.Now().Date()
		if (ty == ny) && (tm == nm) && (td == nd) {
			return template.HTML(t.Format("03:04:05 PM"))
		}
		return template.HTML(t.Format("Mon Jan 2, 2006"))
	},
}

func reverse(name string, things ...interface{}) string {
	// Convert the things to strings
	strs := make([]string, len(things))
	for i, th := range things {
		strs[i] = fmt.Sprint(th)
	}
	// Grab the route
	u, err := Router.Get(name).URL(strs...)
	if err != nil {
		inbucket.Error("Failed to reverse route: %v", err)
		return "/ROUTE-ERROR"
	}
	return u.Path
}
