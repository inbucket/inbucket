// Package webui powers Inbucket's web GUI
package webui

import (
	"github.com/gorilla/mux"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
)

// SetupRoutes populates routes for the webui into the provided Router.
func SetupRoutes(r *mux.Router) {
	r.Path("/greeting").Handler(
		web.Handler(RootGreeting)).Name("RootGreeting").Methods("GET")
	r.Path("/status").Handler(
		web.Handler(RootStatus)).Name("RootStatus").Methods("GET")
	r.Path("/mailbox/{name}/{id}").Handler(
		web.Handler(MailboxMessage)).Name("MailboxMessage").Methods("GET")
	r.Path("/mailbox/{name}/{id}/html").Handler(
		web.Handler(MailboxHTML)).Name("MailboxHTML").Methods("GET")
	r.Path("/mailbox/{name}/{id}/source").Handler(
		web.Handler(MailboxSource)).Name("MailboxSource").Methods("GET")
	r.Path("/mailbox/{name}/{id}/attach/{num}/{file}").Handler(
		web.Handler(MailboxViewAttach)).Name("MailboxViewAttach").Methods("GET")
}
