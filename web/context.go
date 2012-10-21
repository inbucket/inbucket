package web

import (
	"github.com/gorilla/sessions"
	"net/http"
)

type Context struct {
	Session *sessions.Session
}

func (c *Context) Close() {
	// Do nothing
}

func NewContext(req *http.Request) (*Context, error) {
	sess, err := sessionStore.Get(req, "inbucket")
	ctx := &Context{
		Session: sess,
	}
	if err != nil {
		return ctx, err
	}
	return ctx, err
}
