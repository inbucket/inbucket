package web

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jhillyerd/inbucket"
	"net/http"
)

type Context struct {
	Vars      map[string]string
	Session   *sessions.Session
	DataStore *inbucket.DataStore
}

func (c *Context) Close() {
	// Do nothing
}

func NewContext(req *http.Request) (*Context, error) {
	vars := mux.Vars(req)
	sess, err := sessionStore.Get(req, "inbucket")
	ds := inbucket.NewDataStore()
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
