package rest

import (
	"net/http"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/rest/model"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
)

// newMsgListenerV1 creates a v1 API listener which relays message headers only; deletions are
// ignored by the socketv1 API.
func newMsgListenerV1(hub *msghub.Hub, mailbox string) *msgListener[*model.JSONMessageHeaderV1] {
	return newMsgListener(hub, mailbox,
		func(msg event.MessageMetadata) *model.JSONMessageHeaderV1 {
			return metadataToHeader(&msg)
		},
		nil)
}

// MonitorAllMessagesV1 is a web handler which upgrades the connection to a websocket and notifies
// the client of all messages received.
func MonitorAllMessagesV1(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) error {
	return serveMonitor(w, req, ctx.MsgHub, "", newMsgListenerV1)
}

// MonitorMailboxMessagesV1 is a web handler which upgrades the connection to a websocket and
// notifies the client of messages received by a particular mailbox.
func MonitorMailboxMessagesV1(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) error {
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	return serveMonitor(w, req, ctx.MsgHub, name, newMsgListenerV1)
}
