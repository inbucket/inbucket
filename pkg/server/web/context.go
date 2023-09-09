package web

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
)

// Context is passed into every request handler function
// TODO remove redundant web config
type Context struct {
	Vars       map[string]string
	MsgHub     *msghub.Hub
	Manager    message.Manager
	RootConfig *config.Root
	WebConfig  config.Web
	IsJSON     bool
}

// Close the Context (currently does nothing)
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

// NewContext returns a Context for the given HTTP Request
func NewContext(req *http.Request) (*Context, error) {
	vars := mux.Vars(req)
	ctx := &Context{
		Vars:       vars,
		MsgHub:     msgHub,
		Manager:    manager,
		RootConfig: rootConfig,
		WebConfig:  rootConfig.Web,
		IsJSON:     headerMatch(req, "Accept", "application/json"),
	}
	return ctx, nil
}
