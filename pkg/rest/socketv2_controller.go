package rest

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/rest/model"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
	"github.com/rs/zerolog/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWaitV2 = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriodV2 = (pongWaitV2 * 9) / 10

	// Time allowed to read the next pong message from the peer.
	pongWaitV2 = 60 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSizeV2 = 512
)

// options for gorilla connection upgrader
var upgraderV2 = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// msgListenerV2 handles messages from the msghub
type msgListenerV2 struct {
	hub     *msghub.Hub                    // Global message hub.
	c       chan *model.JSONMonitorEventV2 // Queue of incoming events.
	mailbox string                         // Name of mailbox to monitor, "" == all mailboxes.
}

// newMsgListenerV2 creates a listener and registers it.  Optional mailbox parameter will restrict
// messages sent to WebSocket to that mailbox only.
func newMsgListenerV2(hub *msghub.Hub, mailbox string) *msgListenerV2 {
	ml := &msgListenerV2{
		hub:     hub,
		c:       make(chan *model.JSONMonitorEventV2, 100),
		mailbox: mailbox,
	}
	hub.AddListener(ml)
	return ml
}

// Receive handles an incoming message.
func (ml *msgListenerV2) Receive(msg event.MessageMetadata) error {
	if ml.mailbox != "" && ml.mailbox != msg.Mailbox {
		// Did not match the watched mailbox name.
		return nil
	}

	// Enqueue for websocket.
	ml.c <- &model.JSONMonitorEventV2{
		Variant: "message-stored",
		Header:  metadataToHeader(&msg),
	}

	return nil
}

// Delete handles a deleted message.
func (ml *msgListenerV2) Delete(mailbox string, id string) error {
	if ml.mailbox != "" && ml.mailbox != mailbox {
		// Did not match watched mailbox name.
		return nil
	}

	// Enqueue for websocket.
	ml.c <- &model.JSONMonitorEventV2{
		Variant: "message-deleted",
		Identifier: &model.JSONMessageIDV2{
			Mailbox: mailbox,
			ID:      id,
		},
	}

	return nil
}

// WSReader makes sure the websocket client is still connected, discards any messages from client
func (ml *msgListenerV2) WSReader(conn *websocket.Conn) {
	slog := log.With().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Logger()
	defer ml.Close()

	conn.SetReadLimit(maxMessageSizeV2)
	if err := conn.SetReadDeadline(time.Now().Add(pongWaitV2)); err != nil {
		slog.Warn().Err(err).Msg("Failed to setup read deadline")
	}
	conn.SetPongHandler(func(string) error {
		slog.Debug().Msg("Got pong")
		if err := conn.SetReadDeadline(time.Now().Add(pongWaitV2)); err != nil {
			slog.Warn().Err(err).Msg("Failed to set read deadline in pong")
		}
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				// Unexpected close code
				slog.Warn().Err(err).Msg("Socket error")
			} else {
				slog.Debug().Msg("Closing socket")
			}
			break
		}
	}
}

// WSWriter makes sure the websocket client is still connected
func (ml *msgListenerV2) WSWriter(conn *websocket.Conn) {
	slog := log.With().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Logger()

	ticker := time.NewTicker(pingPeriodV2)
	defer func() {
		ticker.Stop()
		ml.Close()
	}()

	// Handle messages from hub until msgListener is closed
	for {
		select {
		case event, ok := <-ml.c:
			if err := conn.SetWriteDeadline(time.Now().Add(writeWaitV2)); err != nil {
				slog.Warn().Err(err).Msg("Failed to set write deadline for msg")
			}
			if !ok {
				// msgListener closed, exit
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if conn.WriteJSON(event) != nil {
				// Write failed
				return
			}
		case <-ticker.C:
			// Send ping
			if err := conn.SetWriteDeadline(time.Now().Add(writeWaitV2)); err != nil {
				slog.Warn().Err(err).Msg("Failed to set write deadline for ping")
			}
			if conn.WriteMessage(websocket.PingMessage, []byte{}) != nil {
				// Write error
				return
			}
			slog.Debug().Msg("Sent ping")
		}
	}
}

// Close removes the listener registration
func (ml *msgListenerV2) Close() {
	select {
	case <-ml.c:
		// Already closed
	default:
		ml.hub.RemoveListener(ml)
		close(ml.c)
	}
}

// MonitorAllMessagesV2 is a web handler which upgrades the connection to a websocket and notifies
// the client of all messages received.
func MonitorAllMessagesV2(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Upgrade to Websocket.
	conn, err := upgraderV2.Upgrade(w, req, nil)
	if err != nil {
		return err
	}
	web.ExpWebSocketConnectsCurrent.Add(1)
	defer func() {
		_ = conn.Close()
		web.ExpWebSocketConnectsCurrent.Add(-1)
	}()
	log.Debug().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Msg("Upgraded to WebSocket")
	// Create, register listener; then interact with conn.
	ml := newMsgListenerV2(ctx.MsgHub, "")
	go ml.WSWriter(conn)
	ml.WSReader(conn)
	return nil
}

// MonitorMailboxMessagesV2 is a web handler which upgrades the connection to a websocket and
// notifies the client of messages received by a particular mailbox.
func MonitorMailboxMessagesV2(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	// Upgrade to Websocket.
	conn, err := upgraderV2.Upgrade(w, req, nil)
	if err != nil {
		return err
	}
	web.ExpWebSocketConnectsCurrent.Add(1)
	defer func() {
		_ = conn.Close()
		web.ExpWebSocketConnectsCurrent.Add(-1)
	}()
	log.Debug().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Msg("Upgraded to WebSocket")
	// Create, register listener; then interact with conn.
	ml := newMsgListenerV2(ctx.MsgHub, name)
	go ml.WSWriter(conn)
	ml.WSReader(conn)
	return nil
}
