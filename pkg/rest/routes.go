package rest

import (
	"github.com/gorilla/mux"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
)

// SetupRoutes populates the routes for the REST interface
func SetupRoutes(r *mux.Router) {
	// API v1
	r.Path("/v1/mailbox/{name}").Handler(
		web.Handler(MailboxListV1)).Name("MailboxListV1").Methods("GET")
	r.Path("/v1/mailbox/{name}").Handler(
		web.Handler(MailboxPurgeV1)).Name("MailboxPurgeV1").Methods("DELETE")
	r.Path("/v1/mailbox/{name}/{id}").Handler(
		web.Handler(MailboxShowV1)).Name("MailboxShowV1").Methods("GET")
	r.Path("/v1/mailbox/{name}/{id}").Handler(
		web.Handler(MailboxMarkSeenV1)).Name("MailboxMarkSeenV1").Methods("PATCH")
	r.Path("/v1/mailbox/{name}/{id}").Handler(
		web.Handler(MailboxDeleteV1)).Name("MailboxDeleteV1").Methods("DELETE")
	r.Path("/v1/mailbox/{name}/{id}/source").Handler(
		web.Handler(MailboxSourceV1)).Name("MailboxSourceV1").Methods("GET")
	r.Path("/v1/monitor/messages").Handler(
		web.Handler(MonitorAllMessagesV1)).Name("MonitorAllMessagesV1").Methods("GET")
	r.Path("/v1/monitor/messages/{name}").Handler(
		web.Handler(MonitorMailboxMessagesV1)).Name("MonitorMailboxMessagesV1").Methods("GET")

	// API v2
	r.Path("/v2/monitor/messages").Handler(
		web.Handler(MonitorAllMessagesV2)).Name("MonitorAllMessagesV2").Methods("GET")
	r.Path("/v2/monitor/messages/{name}").Handler(
		web.Handler(MonitorMailboxMessagesV2)).Name("MonitorMailboxMessagesV2").Methods("GET")
}
