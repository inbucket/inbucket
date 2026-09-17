package rest

import (
	"net/http"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/rest/model"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
)

// newMsgListenerV2 creates a v2 API listener which relays message-stored and message-deleted
// events.
func newMsgListenerV2(hub *msghub.Hub, mailbox string) *msgListener[*model.JSONMonitorEventV2] {
	return newMsgListener(hub, mailbox,
		func(msg event.MessageMetadata) *model.JSONMonitorEventV2 {
			return &model.JSONMonitorEventV2{
				Variant: "message-stored",
				Header:  metadataToHeader(&msg),
			}
		},
		func(mailbox string, id string) *model.JSONMonitorEventV2 {
			return &model.JSONMonitorEventV2{
				Variant: "message-deleted",
				Identifier: &model.JSONMessageIDV2{
					Mailbox: mailbox,
					ID:      id,
				},
			}
		})
}

// MonitorAllMessagesV2 is a web handler which upgrades the connection to a websocket and notifies
// the client of all messages received.
func MonitorAllMessagesV2(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) error {
	return serveMonitor(w, req, ctx.MsgHub, "", newMsgListenerV2)
}

// MonitorMailboxMessagesV2 is a web handler which upgrades the connection to a websocket and
// notifies the client of messages received by a particular mailbox.
func MonitorMailboxMessagesV2(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) error {
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	return serveMonitor(w, req, ctx.MsgHub, name, newMsgListenerV2)
}
