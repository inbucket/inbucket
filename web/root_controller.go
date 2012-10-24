package web

import (
	"net/http"
)

func RootIndex(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	return RenderTemplate("root/index.html", w, map[string]interface{}{
		"ctx": ctx,
	})
}

func RootStatus(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	return RenderTemplate("root/status.html", w, map[string]interface{}{
		"ctx": ctx,
	})
}
