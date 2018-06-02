// Package webui powers Inbucket's web GUI
package webui

import (
	"github.com/gorilla/mux"
	"github.com/jhillyerd/inbucket/pkg/server/web"
)

// SetupRoutes populates routes for the webui into the provided Router.
func SetupRoutes(r *mux.Router) {
	r.Path("/").Handler(
		web.Handler(RootIndex)).Name("RootIndex").Methods("GET")
	r.Path("/monitor").Handler(
		web.Handler(RootMonitor)).Name("RootMonitor").Methods("GET")
	r.Path("/monitor/{name}").Handler(
		web.Handler(RootMonitorMailbox)).Name("RootMonitorMailbox").Methods("GET")
	r.Path("/status").Handler(
		web.Handler(RootStatus)).Name("RootStatus").Methods("GET")
	r.Path("/m/{name}/{id}").Handler(
		web.Handler(MailboxMessage)).Name("MailboxMessage").Methods("GET")
	r.Path("/m/{name}/{id}/html").Handler(
		web.Handler(MailboxHTML)).Name("MailboxHtml").Methods("GET")
	r.Path("/m/{name}/{id}/source").Handler(
		web.Handler(MailboxSource)).Name("MailboxSource").Methods("GET")
	r.Path("/m/attach/{name}/{id}/{num}/{file}").Handler(
		web.Handler(MailboxViewAttach)).Name("MailboxViewAttach").Methods("GET")
}
