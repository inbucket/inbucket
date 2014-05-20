package web

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jhillyerd/inbucket/smtpd"
)

type Context struct {
	Vars      map[string]string
	Session   *sessions.Session
	DataStore smtpd.DataStore
	IsJson    bool
}

func (c *Context) Close() {
	// Do nothing
}

// headerMatch returns true if the request header specified by name contains
// the specified value.  Case is ignored.
func headerMatch(req *http.Request, name string, value string) bool {
	name = http.CanonicalHeaderKey(name)
	value = strings.ToLower(value)

	if header := req.Header[name]; header != nil {
		for _, hv := range header {
			if value == strings.ToLower(hv) {
				return true
			}
		}
	}

	return false
}

func NewContext(req *http.Request) (*Context, error) {
	vars := mux.Vars(req)
	sess, err := sessionStore.Get(req, "inbucket")
	ctx := &Context{
		Vars:      vars,
		Session:   sess,
		DataStore: DataStore,
		IsJson:    headerMatch(req, "Accept", "application/json"),
	}
	if err != nil {
		return ctx, err
	}
	return ctx, err
}
