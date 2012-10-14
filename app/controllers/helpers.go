package controllers

import (
	"github.com/robfig/revel"
	"html/template"
	"time"
)

func init() {
	rev.TRACE.Println("Registering helpers")
	rev.Funcs["friendlyTime"] = func(t time.Time) template.HTML {
		ty, tm, td := t.Date()
		ny, nm, nd := time.Now().Date()
		if (ty == ny) && (tm == nm) && (td == nd) {
			return template.HTML(t.Format("03:04:05 PM"))
		}
		return template.HTML(t.Format("Mon Jan 2, 2006"))
	}
}
