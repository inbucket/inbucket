package web

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jhillyerd/inbucket/smtpd"
	"net/http"
)

type Context struct {
	Vars      map[string]string
	Session   *sessions.Session
	DataStore smtpd.DataStore
}

func (c *Context) Close() {
	// Do nothing
}

func NewContext(req *http.Request) (*Context, error) {
	vars := mux.Vars(req)
	sess, err := sessionStore.Get(req, "inbucket")
	ds := smtpd.NewFileDataStore()
	ctx := &Context{
		Vars:      vars,
		Session:   sess,
		DataStore: ds,
	}
	if err != nil {
		return ctx, err
	}
	return ctx, err
}
