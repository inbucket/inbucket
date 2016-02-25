// Package webui powers Inbucket's web GUI
package webui

import (
	"github.com/gorilla/mux"
	"github.com/jhillyerd/inbucket/httpd"
)

// SetupRoutes populates routes for the webui into the provided Router
func SetupRoutes(r *mux.Router) {
	r.Path("/").Handler(httpd.Handler(RootIndex)).Name("RootIndex").Methods("GET")
	r.Path("/status").Handler(httpd.Handler(RootStatus)).Name("RootStatus").Methods("GET")
	r.Path("/link/{name}/{id}").Handler(httpd.Handler(MailboxLink)).Name("MailboxLink").Methods("GET")
	r.Path("/mailbox").Handler(httpd.Handler(MailboxIndex)).Name("MailboxIndex").Methods("GET")
	r.Path("/mailbox/{name}").Handler(httpd.Handler(MailboxList)).Name("MailboxList").Methods("GET")
	r.Path("/mailbox/{name}").Handler(httpd.Handler(MailboxPurge)).Name("MailboxPurge").Methods("DELETE")
	r.Path("/mailbox/{name}/{id}").Handler(httpd.Handler(MailboxShow)).Name("MailboxShow").Methods("GET")
	r.Path("/mailbox/{name}/{id}/html").Handler(httpd.Handler(MailboxHTML)).Name("MailboxHtml").Methods("GET")
	r.Path("/mailbox/{name}/{id}/source").Handler(httpd.Handler(MailboxSource)).Name("MailboxSource").Methods("GET")
	r.Path("/mailbox/{name}/{id}").Handler(httpd.Handler(MailboxDelete)).Name("MailboxDelete").Methods("DELETE")
	r.Path("/mailbox/dattach/{name}/{id}/{num}/{file}").Handler(httpd.Handler(MailboxDownloadAttach)).Name("MailboxDownloadAttach").Methods("GET")
	r.Path("/mailbox/vattach/{name}/{id}/{num}/{file}").Handler(httpd.Handler(MailboxViewAttach)).Name("MailboxViewAttach").Methods("GET")
}
