package rest

import "github.com/gorilla/mux"
import "github.com/jhillyerd/inbucket/httpd"

// SetupRoutes populates the routes for the REST interface
func SetupRoutes(r *mux.Router) {
	// API v1
	r.Path("/api/v1/mailbox/{name}").Handler(
		httpd.Handler(MailboxListV1)).Name("MailboxListV1").Methods("GET")
	r.Path("/api/v1/mailbox/{name}").Handler(
		httpd.Handler(MailboxPurgeV1)).Name("MailboxPurgeV1").Methods("DELETE")
	r.Path("/api/v1/mailbox/{name}/{id}").Handler(
		httpd.Handler(MailboxShowV1)).Name("MailboxShowV1").Methods("GET")
	r.Path("/api/v1/mailbox/{name}/{id}").Handler(
		httpd.Handler(MailboxDeleteV1)).Name("MailboxDeleteV1").Methods("DELETE")
	r.Path("/api/v1/mailbox/{name}/{id}/source").Handler(
		httpd.Handler(MailboxSourceV1)).Name("MailboxSourceV1").Methods("GET")
	r.Path("/api/v1/monitor/all/messages").Handler(
		httpd.Handler(MonitorAllMessagesV1)).Name("MonitorAllMessagesV1").Methods("GET")
}
