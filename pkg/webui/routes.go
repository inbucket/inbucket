// Package webui powers Inbucket's web GUI
package webui

import (
	"github.com/gorilla/mux"
	"github.com/jhillyerd/inbucket/pkg/server/web"
)

// SetupRoutes populates routes for the webui into the provided Router
func SetupRoutes(r *mux.Router) {
	r.Path("/").Handler(
		web.Handler(RootIndex)).Name("RootIndex").Methods("GET")
	r.Path("/monitor").Handler(
		web.Handler(RootMonitor)).Name("RootMonitor").Methods("GET")
	r.Path("/monitor/{name}").Handler(
		web.Handler(RootMonitorMailbox)).Name("RootMonitorMailbox").Methods("GET")
	r.Path("/status").Handler(
		web.Handler(RootStatus)).Name("RootStatus").Methods("GET")
	r.Path("/link/{name}/{id}").Handler(
		web.Handler(MailboxLink)).Name("MailboxLink").Methods("GET")
	r.Path("/mailbox").Handler(
		web.Handler(MailboxIndex)).Name("MailboxIndex").Methods("GET")
	r.Path("/mailbox/{name}").Handler(
		web.Handler(MailboxList)).Name("MailboxList").Methods("GET")
	r.Path("/mailbox/{name}/{id}").Handler(
		web.Handler(MailboxShow)).Name("MailboxShow").Methods("GET")
	r.Path("/mailbox/{name}/{id}/html").Handler(
		web.Handler(MailboxHTML)).Name("MailboxHtml").Methods("GET")
	r.Path("/mailbox/{name}/{id}/source").Handler(
		web.Handler(MailboxSource)).Name("MailboxSource").Methods("GET")
	r.Path("/mailbox/dattach/{name}/{id}/{num}/{file}").Handler(
		web.Handler(MailboxDownloadAttach)).Name("MailboxDownloadAttach").Methods("GET")
	r.Path("/mailbox/vattach/{name}/{id}/{num}/{file}").Handler(
		web.Handler(MailboxViewAttach)).Name("MailboxViewAttach").Methods("GET")
	r.Path("/{name}").Handler(
		web.Handler(MailboxIndexFriendly)).Name("MailboxListFriendly").Methods("GET")
}
