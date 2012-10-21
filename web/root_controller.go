package web

import (
	"net/http"
)

func RootIndex(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	return T("root-index.html").Execute(w, nil)
}
